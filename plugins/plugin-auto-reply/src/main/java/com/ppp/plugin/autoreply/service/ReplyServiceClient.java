package com.ppp.plugin.autoreply.service;

import com.ppp.plugin.autoreply.config.AutoReplyProperties;
import com.ppp.plugin.autoreply.service.dto.AutoReplyRequest;
import com.ppp.plugin.autoreply.service.dto.AutoReplyResponse;
import java.time.Duration;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;

/**
 * Reactive client to auto-reply service.
 */
@Slf4j
@Component
public class ReplyServiceClient {

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(10);
    private static final String REPLY_PATH = "/api/v1/reply";

    private final WebClient webClient;

    public ReplyServiceClient() {
        this.webClient = WebClient.builder().build();
    }

    public Mono<AutoReplyResponse> generateReply(String replyServiceUrl, AutoReplyRequest request) {
        String normalizedBaseUrl = normalizeBaseUrl(replyServiceUrl);

        return webClient
            .post()
            .uri(normalizedBaseUrl + REPLY_PATH)
            .contentType(MediaType.APPLICATION_JSON)
            .bodyValue(request)
            .retrieve()
            .bodyToMono(AutoReplyResponse.class)
            .timeout(REQUEST_TIMEOUT)
            .onErrorResume(ex -> {
                log.warn("auto-reply service call failed, skip reply. url={}, error={}",
                    normalizedBaseUrl, ex.toString());
                return Mono.just(AutoReplyResponse.noReply());
            })
            .map(response -> response == null ? AutoReplyResponse.noReply() : response);
    }

    private String normalizeBaseUrl(String baseUrl) {
        if (baseUrl == null || baseUrl.isBlank()) {
            return AutoReplyProperties.DEFAULT_REPLY_SERVICE_URL;
        }
        String trimmed = baseUrl.trim();
        if (trimmed.endsWith("/")) {
            return trimmed.substring(0, trimmed.length() - 1);
        }
        return trimmed;
    }
}
