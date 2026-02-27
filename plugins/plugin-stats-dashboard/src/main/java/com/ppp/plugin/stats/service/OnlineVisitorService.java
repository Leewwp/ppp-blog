package com.ppp.plugin.stats.service;

import java.time.Duration;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.atomic.AtomicLong;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Schedulers;
import run.halo.app.plugin.SettingFetcher;

/**
 * Service for visitor presence tracking and daily page-view counters.
 * Uses in-memory storage to avoid external runtime dependencies.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class OnlineVisitorService {

    private static final String GROUP_BASIC = "basic";
    private static final String ONLINE_VISITOR_KEY_PREFIX = "blog:online:visitors:";
    private static final int DEFAULT_VISITOR_TIMEOUT_SECONDS = 300;
    private static final int DAILY_PV_KEEP_DAYS = 30;
    private static final DateTimeFormatter DATE_FORMATTER = DateTimeFormatter.ISO_LOCAL_DATE;
    private static final ZoneId DAILY_PV_ZONE = ZoneId.of("Asia/Shanghai");

    private final SettingFetcher settingFetcher;
    private final ConcurrentMap<String, Long> onlineVisitors = new ConcurrentHashMap<>();
    private final ConcurrentMap<LocalDate, AtomicLong> dailyViews = new ConcurrentHashMap<>();

    /**
     * Records one IP as online with a TTL.
     */
    public Mono<Void> recordVisit(String ip) {
        String normalizedIp = normalizeIp(ip);
        if (normalizedIp.isEmpty()) {
            return Mono.empty();
        }

        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.empty();
        }

        Duration ttl = Duration.ofSeconds(Math.max(30, settings.visitorTimeoutSeconds()));
        return Mono.fromRunnable(() -> {
                long now = Instant.now().toEpochMilli();
                long expireAt = now + ttl.toMillis();
                onlineVisitors.put(ONLINE_VISITOR_KEY_PREFIX + normalizedIp, expireAt);
                pruneExpiredVisitors(now);
            })
            .subscribeOn(Schedulers.boundedElastic())
            .onErrorResume(error -> {
                log.warn("failed to record online visitor, ip={}, error={}",
                    normalizedIp, error.toString());
                return Mono.empty();
            });
    }

    /**
     * Counts currently active visitors by pruning expired entries first.
     */
    public Mono<Long> countOnlineVisitors() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.just(0L);
        }

        return Mono.fromCallable(() -> {
                long now = Instant.now().toEpochMilli();
                pruneExpiredVisitors(now);
                return (long) onlineVisitors.size();
            })
            .subscribeOn(Schedulers.boundedElastic())
            .onErrorResume(error -> {
                log.warn("failed to count online visitors, fallback to 0, error={}", error.toString());
                return Mono.just(0L);
            });
    }

    /**
     * Increments page-view counter for current day.
     */
    public Mono<Void> incrementDailyPageView() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.empty();
        }

        LocalDate today = LocalDate.now(DAILY_PV_ZONE);
        return Mono.fromRunnable(() -> {
                dailyViews.computeIfAbsent(today, ignored -> new AtomicLong(0L)).incrementAndGet();
                pruneOldDailyViews(today);
            })
            .subscribeOn(Schedulers.boundedElastic())
            .onErrorResume(error -> {
                log.warn("failed to increment daily pv, date={}, error={}",
                    today.format(DATE_FORMATTER), error.toString());
                return Mono.empty();
            });
    }

    /**
     * Reads today's page views.
     */
    public Mono<Long> countTodayViews() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.just(0L);
        }
        LocalDate today = LocalDate.now(DAILY_PV_ZONE);
        return Mono.fromCallable(() -> Optional.ofNullable(dailyViews.get(today))
                .map(AtomicLong::get)
                .orElse(0L))
            .subscribeOn(Schedulers.boundedElastic())
            .onErrorResume(error -> {
                log.warn("failed to read today's pv, fallback to 0, error={}", error.toString());
                return Mono.just(0L);
            });
    }

    /**
     * Reads the latest N days page views (oldest to newest).
     */
    public Mono<List<DailyPv>> getRecentDailyViews(int days) {
        if (days <= 0) {
            return Mono.just(List.of());
        }

        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.just(zeroHistory(days));
        }

        return Mono.fromCallable(() -> {
                LocalDate today = LocalDate.now(DAILY_PV_ZONE);
                pruneOldDailyViews(today);

                List<DailyPv> values = new ArrayList<>(days);
                for (int i = 0; i < days; i++) {
                    LocalDate date = today.minusDays(days - 1L - i);
                    long views = Optional.ofNullable(dailyViews.get(date))
                        .map(AtomicLong::get)
                        .orElse(0L);
                    values.add(new DailyPv(date.format(DATE_FORMATTER), views));
                }
                return values;
            })
            .subscribeOn(Schedulers.boundedElastic())
            .onErrorResume(error -> {
                log.warn("failed to read pv history, fallback to zeros, error={}", error.toString());
                return Mono.just(zeroHistory(days));
            });
    }

    private Settings currentSettings() {
        BasicGroup basic = settingFetcher.fetch(GROUP_BASIC, BasicGroup.class)
            .orElseGet(BasicGroup::new);

        boolean redisEnabled = Optional.ofNullable(basic.getRedisEnabled()).orElse(Boolean.TRUE);
        int timeoutSeconds = Optional.ofNullable(basic.getVisitorTimeout())
            .filter(value -> value > 0)
            .orElse(DEFAULT_VISITOR_TIMEOUT_SECONDS);
        return new Settings(redisEnabled, timeoutSeconds);
    }

    private List<DailyPv> zeroHistory(int days) {
        LocalDate today = LocalDate.now(DAILY_PV_ZONE);
        List<DailyPv> values = new ArrayList<>(days);
        for (int i = 0; i < days; i++) {
            LocalDate date = today.minusDays(days - 1L - i);
            values.add(new DailyPv(date.format(DATE_FORMATTER), 0L));
        }
        return values;
    }

    private String normalizeIp(String ip) {
        if (ip == null || ip.isBlank()) {
            return "";
        }

        String trimmed = ip.trim();
        if (trimmed.equalsIgnoreCase("unknown")) {
            return "";
        }
        return trimmed;
    }

    private void pruneExpiredVisitors(long nowMillis) {
        onlineVisitors.entrySet().removeIf(entry -> entry.getValue() <= nowMillis);
    }

    private void pruneOldDailyViews(LocalDate today) {
        LocalDate oldestToKeep = today.minusDays(Math.max(0, DAILY_PV_KEEP_DAYS - 1L));
        dailyViews.keySet().removeIf(date -> date.isBefore(oldestToKeep));
    }

    @Data
    public static class BasicGroup {
        private Boolean redisEnabled = Boolean.TRUE;
        private Integer visitorTimeout = DEFAULT_VISITOR_TIMEOUT_SECONDS;
    }

    public record Settings(boolean redisEnabled, int visitorTimeoutSeconds) {
    }

    public record DailyPv(String date, long views) {
    }
}
