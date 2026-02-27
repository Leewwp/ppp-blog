package com.ppp.plugin.stats.service;

import java.time.Duration;
import java.time.LocalDate;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.data.redis.core.ReactiveRedisTemplate;
import org.springframework.data.redis.core.ScanOptions;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;
import run.halo.app.plugin.SettingFetcher;

/**
 * Service for visitor presence tracking and daily page-view counters.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class OnlineVisitorService {

    private static final String GROUP_BASIC = "basic";
    private static final String ONLINE_VISITOR_KEY_PREFIX = "blog:online:visitors:";
    private static final String DAILY_PV_KEY_PREFIX = "blog:pv:daily:";
    private static final int DEFAULT_VISITOR_TIMEOUT_SECONDS = 300;
    private static final int DAILY_PV_KEEP_DAYS = 30;
    private static final DateTimeFormatter DATE_FORMATTER = DateTimeFormatter.ISO_LOCAL_DATE;
    private static final ZoneId DAILY_PV_ZONE = ZoneId.of("Asia/Shanghai");

    private final ObjectProvider<ReactiveRedisTemplate<String, String>> redisTemplateProvider;
    private final SettingFetcher settingFetcher;

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

        ReactiveRedisTemplate<String, String> redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return Mono.empty();
        }

        String key = ONLINE_VISITOR_KEY_PREFIX + normalizedIp;
        Duration ttl = Duration.ofSeconds(Math.max(30, settings.visitorTimeoutSeconds()));
        return redisTemplate.opsForValue()
            .set(key, normalizedIp, ttl)
            .then()
            .onErrorResume(error -> {
                log.warn("failed to record online visitor, ip={}, error={}",
                    normalizedIp, error.toString());
                return Mono.empty();
            });
    }

    /**
     * Counts online visitor keys using SCAN to avoid blocking Redis.
     */
    public Mono<Long> countOnlineVisitors() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.just(0L);
        }

        ReactiveRedisTemplate<String, String> redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return Mono.just(0L);
        }

        ScanOptions options = ScanOptions.scanOptions()
            .match(ONLINE_VISITOR_KEY_PREFIX + "*")
            .count(1000)
            .build();

        return redisTemplate.scan(options)
            .count()
            .onErrorResume(error -> {
                log.warn("failed to count online visitors, fallback to 0, error={}", error.toString());
                return Mono.just(0L);
            });
    }

    /**
     * Increments page-view counter for current day and refreshes TTL.
     */
    public Mono<Void> incrementDailyPageView() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.empty();
        }

        ReactiveRedisTemplate<String, String> redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return Mono.empty();
        }

        String key = dailyPvKey(LocalDate.now(DAILY_PV_ZONE));
        return redisTemplate.opsForValue()
            .increment(key)
            .flatMap(ignored -> redisTemplate.expire(key, Duration.ofDays(DAILY_PV_KEEP_DAYS)))
            .then()
            .onErrorResume(error -> {
                log.warn("failed to increment daily pv, key={}, error={}", key, error.toString());
                return Mono.empty();
            });
    }

    /**
     * Reads today's page views from Redis.
     */
    public Mono<Long> countTodayViews() {
        Settings settings = currentSettings();
        if (!settings.redisEnabled()) {
            return Mono.just(0L);
        }
        return readDailyViews(LocalDate.now(DAILY_PV_ZONE));
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

        LocalDate today = LocalDate.now(DAILY_PV_ZONE);
        return Flux.range(0, days)
            .map(i -> today.minusDays(days - 1L - i))
            .concatMap(date -> readDailyViews(date)
                .map(views -> new DailyPv(date.format(DATE_FORMATTER), views)))
            .collectList()
            .onErrorResume(error -> {
                log.warn("failed to read pv history, fallback to zeros, error={}", error.toString());
                return Mono.just(zeroHistory(days));
            });
    }

    private Mono<Long> readDailyViews(LocalDate date) {
        ReactiveRedisTemplate<String, String> redisTemplate = redisTemplateProvider.getIfAvailable();
        if (redisTemplate == null) {
            return Mono.just(0L);
        }

        String key = dailyPvKey(date);
        return redisTemplate.opsForValue()
            .get(key)
            .map(this::safeParseLong)
            .defaultIfEmpty(0L)
            .onErrorResume(error -> {
                log.warn("failed to read daily pv, key={}, fallback to 0, error={}",
                    key, error.toString());
                return Mono.just(0L);
            });
    }

    private String dailyPvKey(LocalDate date) {
        return DAILY_PV_KEY_PREFIX + date.format(DATE_FORMATTER);
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

    private long safeParseLong(String value) {
        if (value == null || value.isBlank()) {
            return 0L;
        }
        try {
            return Long.parseLong(value.trim());
        } catch (NumberFormatException ex) {
            return 0L;
        }
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
