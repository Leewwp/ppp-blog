package com.ppp.plugin.stats.filter;

import com.ppp.plugin.stats.service.OnlineVisitorService;
import java.net.InetSocketAddress;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.server.reactive.ServerHttpRequest;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebFilter;
import org.springframework.web.server.WebFilterChain;
import reactor.core.publisher.Mono;

/**
 * Tracks online visitors and daily PV for every incoming request.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class VisitorTrackingFilter implements WebFilter {

    private final OnlineVisitorService onlineVisitorService;

    @Override
    public Mono<Void> filter(ServerWebExchange exchange, WebFilterChain chain) {
        String path = exchange.getRequest().getPath().value();
        if (shouldSkipPath(path)) {
            return chain.filter(exchange);
        }

        String ip = extractClientIp(exchange.getRequest());
        Mono<Void> tracking = onlineVisitorService.recordVisit(ip)
            .then(onlineVisitorService.incrementDailyPageView())
            .onErrorResume(error -> {
                log.warn("visitor tracking failed, ip={}, error={}", ip, error.toString(), error);
                return Mono.empty();
            });

        return tracking.then(chain.filter(exchange));
    }

    private boolean shouldSkipPath(String path) {
        if (path == null || path.isBlank()) {
            return true;
        }

        String normalized = path.toLowerCase();
        if (normalized.startsWith("/actuator")
            || normalized.startsWith("/apis/")
            || normalized.startsWith("/console")
            || normalized.startsWith("/upload/")
            || normalized.startsWith("/themes/")
            || normalized.startsWith("/plugins/")) {
            return true;
        }

        return normalized.matches(".*\\.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|map)$");
    }

    private String extractClientIp(ServerHttpRequest request) {
        String forwardedFor = request.getHeaders().getFirst("X-Forwarded-For");
        if (forwardedFor != null && !forwardedFor.isBlank()) {
            String first = forwardedFor.split(",")[0].trim();
            if (!first.isEmpty()) {
                return stripPort(first);
            }
        }

        InetSocketAddress remoteAddress = request.getRemoteAddress();
        if (remoteAddress == null || remoteAddress.getAddress() == null) {
            return "";
        }

        String host = remoteAddress.getAddress().getHostAddress();
        if (host == null || host.isBlank()) {
            return "";
        }
        return stripPort(host.trim());
    }

    private String stripPort(String ip) {
        if (ip == null || ip.isBlank()) {
            return "";
        }

        String trimmed = ip.trim();
        if (trimmed.startsWith("[") && trimmed.contains("]")) {
            int endBracket = trimmed.indexOf(']');
            return trimmed.substring(1, endBracket);
        }

        int firstColon = trimmed.indexOf(':');
        int lastColon = trimmed.lastIndexOf(':');
        if (firstColon == lastColon && lastColon > 0 && trimmed.contains(".")) {
            return trimmed.substring(0, lastColon);
        }

        return trimmed;
    }
}
