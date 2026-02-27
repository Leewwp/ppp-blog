package com.ppp.plugin.moderation.service;

import com.ppp.plugin.moderation.service.dto.CommentFilterRequest;
import com.ppp.plugin.moderation.service.dto.CommentFilterResponse;
import java.time.Duration;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;

/**
 * Reactive client to comment-filter service.
 */
@Slf4j
@Component
public class FilterServiceClient {

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(3);
    private static final String FILTER_PATH = "/api/v1/filter";

    private final WebClient webClient;

    public FilterServiceClient(WebClient.Builder webClientBuilder) {
        this.webClient = webClientBuilder.build();
    }

    public Mono<CommentFilterResponse> filter(String filterServiceUrl, String content, String author) {
        String normalizedBaseUrl = normalizeBaseUrl(filterServiceUrl);
        CommentFilterRequest request = new CommentFilterRequest(content, author);

        return webClient
            .post()
            .uri(normalizedBaseUrl + FILTER_PATH)
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue(request)
            .retrieve()
            .bodyToMono(CommentFilterResponse.class)
            .timeout(REQUEST_TIMEOUT)
            .onErrorResume(ex -> {
                log.warn("filter service call failed, fallback pass-through. url={}, error={}",
                    normalizedBaseUrl, ex.toString());
                return Mono.just(CommentFilterResponse.passThrough(content));
            })
            .map(response -> response == null ? CommentFilterResponse.passThrough(content) : response);
    }

    private String normalizeBaseUrl(String baseUrl) {
        if (baseUrl == null || baseUrl.isBlank()) {
            return "http://comment-filter:8091";
        }
        String trimmed = baseUrl.trim();
        if (trimmed.endsWith("/")) {
            return trimmed.substring(0, trimmed.length() - 1);
        }
        return trimmed;
    }
}
