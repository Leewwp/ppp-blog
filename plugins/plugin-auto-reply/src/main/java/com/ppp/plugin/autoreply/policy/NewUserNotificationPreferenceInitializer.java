package com.ppp.plugin.autoreply.policy;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import run.halo.app.core.extension.User;
import run.halo.app.core.user.service.UserPostCreatingHandler;
import run.halo.app.extension.ConfigMap;
import run.halo.app.extension.Metadata;
import run.halo.app.extension.ReactiveExtensionClient;

/**
 * Initializes notification defaults for newly created users to reduce email cost.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class NewUserNotificationPreferenceInitializer implements UserPostCreatingHandler {

    private static final String CONFIG_PREFIX = "user-preferences-";
    private static final String NOTIFICATION_KEY = "notification";
    private static final String DEFAULT_NOTIFICATION_JSON =
        "{\"reasonTypeNotifier\":{"
            + "\"new-device-login\":{\"notifiers\":[]},"
            + "\"someone-replied-to-you\":{\"notifiers\":[]},"
            + "\"new-comment-on-single-page\":{\"notifiers\":[]},"
            + "\"new-comment-on-post\":{\"notifiers\":[]}"
            + "}}";

    private final ReactiveExtensionClient extensionClient;

    @Override
    public Mono<Void> postCreating(User user) {
        String username = Optional.ofNullable(user)
            .map(User::getMetadata)
            .map(metadata -> metadata.getName())
            .orElse(null);
        if (username == null || username.isBlank()) {
            return Mono.empty();
        }

        String configName = CONFIG_PREFIX + username;
        return extensionClient.fetch(ConfigMap.class, configName)
            .flatMap(this::initPreferenceForExistingConfig)
            .switchIfEmpty(Mono.defer(() -> createPreferenceConfig(configName)))
            .doOnError(error -> log.warn(
                "failed to initialize notification preference for user={}, error={}",
                username, error.toString()))
            .onErrorResume(error -> Mono.empty());
    }

    private Mono<Void> initPreferenceForExistingConfig(ConfigMap configMap) {
        Map<String, String> data = configMap.getData();
        if (data == null) {
            data = new HashMap<>();
            configMap.setData(data);
        }
        String existing = data.get(NOTIFICATION_KEY);
        if (existing != null && !existing.isBlank()) {
            return Mono.empty();
        }
        data.put(NOTIFICATION_KEY, DEFAULT_NOTIFICATION_JSON);
        return extensionClient.update(configMap).then();
    }

    private Mono<Void> createPreferenceConfig(String configName) {
        ConfigMap configMap = new ConfigMap();
        Metadata metadata = new Metadata();
        metadata.setName(configName);
        configMap.setMetadata(metadata);

        Map<String, String> data = new HashMap<>();
        data.put(NOTIFICATION_KEY, DEFAULT_NOTIFICATION_JSON);
        configMap.setData(data);
        return extensionClient.create(configMap).then();
    }
}
