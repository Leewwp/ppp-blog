package com.ppp.plugin.autoreply.policy;

import java.nio.charset.StandardCharsets;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;
import lombok.extern.slf4j.Slf4j;
import org.springframework.core.Ordered;
import org.springframework.http.HttpMethod;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.server.ServerWebExchange;
import org.springframework.web.server.WebFilterChain;
import reactor.core.publisher.Mono;
import run.halo.app.security.AdditionalWebFilter;

/**
 * Global guard for email verification sending quota.
 */
@Slf4j
@Component
public class EmailVerificationDailyLimitFilter implements AdditionalWebFilter {

    private static final int DAILY_LIMIT = 200;
    private static final ZoneId COUNTER_ZONE = ZoneId.of("Asia/Shanghai");
    private static final String LIMIT_DETAIL =
        "\u90ae\u7bb1\u9a8c\u8bc1\u7801\u4eca\u65e5\u53d1\u9001\u5df2\u8fbe\u4e0a\u9650\uff08200/\u5929\uff09\uff0c\u8bf7\u660e\u5929\u518d\u8bd5\u3002";
    private static final String PROBLEM_TYPE = "https://halo.run/probs/request-not-permitted";
    private static final String PROBLEM_TITLE = "Request Not Permitted";
    private static final Set<String> TARGET_PATHS = Set.of(
        "/apis/api.console.halo.run/v1alpha1/users/-/send-email-verification-code",
        "/signup/send-email-code"
    );

    private final AtomicReference<LocalDate> currentDay =
        new AtomicReference<>(LocalDate.now(COUNTER_ZONE));
    private final AtomicInteger successCount = new AtomicInteger(0);

    @Override
    public int getOrder() {
        return Ordered.HIGHEST_PRECEDENCE + 50;
    }

    @Override
    public Mono<Void> filter(ServerWebExchange exchange, WebFilterChain chain) {
        if (!shouldApply(exchange)) {
            return chain.filter(exchange);
        }

        rotateIfNeeded();
        int count = successCount.incrementAndGet();
        if (count > DAILY_LIMIT) {
            successCount.decrementAndGet();
            log.warn("email verification quota reached: limit={}, path={}",
                DAILY_LIMIT, exchange.getRequest().getPath().value());
            return writeQuotaExceeded(exchange);
        }

        return chain.filter(exchange)
            .doOnSuccess(ignored -> {
                HttpStatusCode statusCode = exchange.getResponse().getStatusCode();
                if (statusCode != null && !statusCode.is2xxSuccessful()) {
                    successCount.decrementAndGet();
                }
            })
            .doOnError(error -> successCount.decrementAndGet());
    }

    private boolean shouldApply(ServerWebExchange exchange) {
        if (exchange.getRequest().getMethod() != HttpMethod.POST) {
            return false;
        }
        String path = exchange.getRequest().getPath().value();
        return TARGET_PATHS.contains(path);
    }

    private void rotateIfNeeded() {
        LocalDate today = LocalDate.now(COUNTER_ZONE);
        LocalDate observed = currentDay.get();
        if (today.equals(observed)) {
            return;
        }
        if (currentDay.compareAndSet(observed, today)) {
            successCount.set(0);
            log.info("email verification quota counter rotated: day={}", today);
        }
    }

    private Mono<Void> writeQuotaExceeded(ServerWebExchange exchange) {
        String instance = exchange.getRequest().getPath().value();
        String body = """
            {"type":"%s","title":"%s","status":429,"detail":"%s","instance":"%s"}
            """.formatted(PROBLEM_TYPE, PROBLEM_TITLE, LIMIT_DETAIL, instance);

        var response = exchange.getResponse();
        response.setStatusCode(HttpStatus.TOO_MANY_REQUESTS);
        response.getHeaders().setContentType(MediaType.APPLICATION_PROBLEM_JSON);
        var dataBuffer = response.bufferFactory().wrap(body.getBytes(StandardCharsets.UTF_8));
        return response.writeWith(Mono.just(dataBuffer));
    }
}
