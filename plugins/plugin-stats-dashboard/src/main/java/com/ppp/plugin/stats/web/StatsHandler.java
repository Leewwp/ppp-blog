package com.ppp.plugin.stats.web;

import com.ppp.plugin.stats.model.StatsResponse;
import com.ppp.plugin.stats.service.BlogStatsService;
import com.ppp.plugin.stats.service.OnlineVisitorService;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.security.authentication.AnonymousAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.ReactiveSecurityContextHolder;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.server.ServerRequest;
import org.springframework.web.reactive.function.server.ServerResponse;
import reactor.core.publisher.Mono;

/**
 * Reactive handler for stats endpoints.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class StatsHandler {

    private final OnlineVisitorService onlineVisitorService;
    private final BlogStatsService blogStatsService;

    public Mono<ServerResponse> getStats(ServerRequest request) {
        return withAuth(buildStatsResponse());
    }

    public Mono<ServerResponse> getHistory(ServerRequest request) {
        return withAuth(buildHistoryResponse());
    }

    public Mono<ServerResponse> getPublicStats(ServerRequest request) {
        return buildStatsResponse();
    }

    public Mono<ServerResponse> getPublicHistory(ServerRequest request) {
        return buildHistoryResponse();
    }

    private Mono<ServerResponse> buildStatsResponse() {
        return Mono.zip(
                onlineVisitorService.countOnlineVisitors(),
                blogStatsService.getAggregatedStats(),
                onlineVisitorService.countTodayViews()
            )
            .flatMap(tuple -> {
                BlogStatsService.AggregatedStats aggregatedStats = tuple.getT2();
                StatsResponse response = new StatsResponse(
                    tuple.getT1(),
                    aggregatedStats.totalPosts(),
                    aggregatedStats.totalComments(),
                    aggregatedStats.totalViews(),
                    tuple.getT3()
                );
                return ServerResponse.ok()
                    .contentType(MediaType.APPLICATION_JSON)
                    .bodyValue(response);
            });
    }

    private Mono<ServerResponse> buildHistoryResponse() {
        return onlineVisitorService.getRecentDailyViews(7)
            .flatMap(history -> ServerResponse.ok()
                .contentType(MediaType.APPLICATION_JSON)
                .bodyValue(Map.of("history", history)));
    }

    private Mono<ServerResponse> withAuth(Mono<ServerResponse> onAuthorized) {
        return ReactiveSecurityContextHolder.getContext()
            .map(SecurityContext::getAuthentication)
            .filter(this::isAuthenticated)
            .hasElement()
            .flatMap(authenticated -> {
                if (authenticated) {
                    return onAuthorized;
                }
                return ServerResponse.status(HttpStatus.UNAUTHORIZED)
                    .contentType(MediaType.APPLICATION_JSON)
                    .bodyValue(Map.of("message", "unauthorized"));
            });
    }

    private boolean isAuthenticated(Authentication authentication) {
        return authentication != null
            && authentication.isAuthenticated()
            && !(authentication instanceof AnonymousAuthenticationToken);
    }
}
