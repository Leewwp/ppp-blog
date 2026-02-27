package com.ppp.plugin.stats.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.domain.Sort;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;
import run.halo.app.core.extension.content.Comment;
import run.halo.app.core.extension.content.Post;
import run.halo.app.extension.ListOptions;
import run.halo.app.extension.ReactiveExtensionClient;

/**
 * Aggregates blog stats from Halo extensions.
 */
@Slf4j
@Service
public class BlogStatsService {

    private static final Duration CACHE_TTL = Duration.ofSeconds(60);
    private static final ListOptions EMPTY_OPTIONS = ListOptions.builder().build();

    private final ReactiveExtensionClient extensionClient;
    private final ObjectMapper objectMapper = new ObjectMapper();

    private final AtomicReference<CacheEntry> cache = new AtomicReference<>(CacheEntry.expired());

    public BlogStatsService(ReactiveExtensionClient extensionClient) {
        this.extensionClient = extensionClient;
    }

    /**
     * Returns cached stats when valid, otherwise refreshes from Halo.
     */
    public Mono<AggregatedStats> getAggregatedStats() {
        CacheEntry cached = cache.get();
        if (cached.isValid(Instant.now())) {
            return Mono.just(cached.stats());
        }

        return loadFreshStats()
            .doOnSuccess(stats -> cache.set(CacheEntry.of(stats, Instant.now().plus(CACHE_TTL))));
    }

    private Mono<AggregatedStats> loadFreshStats() {
        Mono<Long> totalPublishedPosts = extensionClient
            .listAll(Post.class, EMPTY_OPTIONS, Sort.unsorted())
            .filter(Post::isPublished)
            .count();

        Mono<Long> totalApprovedComments = extensionClient
            .listAll(Comment.class, EMPTY_OPTIONS, Sort.unsorted())
            .filter(this::isApprovedComment)
            .count();

        Mono<Long> totalViews = extensionClient
            .listAll(Post.class, EMPTY_OPTIONS, Sort.unsorted())
            .filter(Post::isPublished)
            .map(this::extractViews)
            .reduce(0L, Long::sum);

        return Mono.zip(totalPublishedPosts, totalApprovedComments, totalViews)
            .map(tuple -> new AggregatedStats(tuple.getT1(), tuple.getT2(), tuple.getT3()))
            .onErrorResume(error -> {
                log.warn("failed to aggregate blog stats, fallback to zeros, error={}",
                    error.toString(), error);
                return Mono.just(AggregatedStats.empty());
            });
    }

    private boolean isApprovedComment(Comment comment) {
        return comment != null
            && comment.getSpec() != null
            && Boolean.TRUE.equals(comment.getSpec().getApproved());
    }

    private long extractViews(Post post) {
        if (post == null || post.getMetadata() == null) {
            return 0L;
        }

        Map<String, String> annotations = post.getMetadata().getAnnotations();
        if (annotations == null || !annotations.containsKey(Post.STATS_ANNO)) {
            return 0L;
        }

        String statsJson = annotations.get(Post.STATS_ANNO);
        if (statsJson == null || statsJson.isBlank()) {
            return 0L;
        }

        try {
            JsonNode node = objectMapper.readTree(statsJson);
            return node.path("visit").asLong(0L);
        } catch (Exception ex) {
            if (post.getMetadata().getName() != null) {
                log.debug("failed to parse post stats annotation, post={}, error={}",
                    post.getMetadata().getName(), ex.toString());
            }
            return 0L;
        }
    }

    private record CacheEntry(AggregatedStats stats, Instant expireAt) {
        static CacheEntry expired() {
            return new CacheEntry(AggregatedStats.empty(), Instant.EPOCH);
        }

        static CacheEntry of(AggregatedStats stats, Instant expireAt) {
            return new CacheEntry(stats, expireAt);
        }

        boolean isValid(Instant now) {
            return now.isBefore(expireAt);
        }
    }

    public record AggregatedStats(long totalPosts, long totalComments, long totalViews) {
        public static AggregatedStats empty() {
            return new AggregatedStats(0L, 0L, 0L);
        }
    }
}
