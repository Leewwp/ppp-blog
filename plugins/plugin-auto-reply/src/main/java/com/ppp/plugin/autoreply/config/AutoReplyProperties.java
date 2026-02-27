package com.ppp.plugin.autoreply.config;

import java.util.Optional;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.PluginConfigUpdatedEvent;
import run.halo.app.plugin.SettingFetcher;

/**
 * Reads auto reply settings from Halo plugin ConfigMap via SettingFetcher.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class AutoReplyProperties {

    public static final String GROUP_BASIC = "basic";
    public static final String DEFAULT_REPLY_SERVICE_URL = "http://auto-reply:8092";
    public static final String DEFAULT_REPLY_AUTHOR_NAME = "博主";

    private final SettingFetcher settingFetcher;

    public Settings current() {
        BasicGroup basic = settingFetcher
            .fetch(GROUP_BASIC, BasicGroup.class)
            .orElseGet(BasicGroup::new);

        String envUrl = Optional.ofNullable(System.getenv("AUTO_REPLY_SERVICE_URL"))
            .map(String::trim)
            .filter(v -> !v.isEmpty())
            .orElse(DEFAULT_REPLY_SERVICE_URL);

        String replyServiceUrl = Optional.ofNullable(basic.getReplyServiceUrl())
            .map(String::trim)
            .filter(v -> !v.isEmpty())
            .orElse(envUrl);

        String replyAuthorName = Optional.ofNullable(basic.getReplyAuthorName())
            .map(String::trim)
            .filter(v -> !v.isEmpty())
            .orElse(DEFAULT_REPLY_AUTHOR_NAME);

        boolean enabled = Optional.ofNullable(basic.getEnabled()).orElse(Boolean.TRUE);

        return new Settings(replyServiceUrl, replyAuthorName, enabled);
    }

    @EventListener
    public void onPluginConfigUpdated(PluginConfigUpdatedEvent event) {
        Settings settings = current();
        log.info("auto reply settings updated: enabled={}, replyServiceUrl={}, replyAuthorName={}",
            settings.enabled(), settings.replyServiceUrl(), settings.replyAuthorName());
    }

    @Data
    public static class BasicGroup {
        private String replyServiceUrl = DEFAULT_REPLY_SERVICE_URL;
        private String replyAuthorName = DEFAULT_REPLY_AUTHOR_NAME;
        private Boolean enabled = Boolean.TRUE;
    }

    public record Settings(String replyServiceUrl, String replyAuthorName, boolean enabled) {
    }
}
