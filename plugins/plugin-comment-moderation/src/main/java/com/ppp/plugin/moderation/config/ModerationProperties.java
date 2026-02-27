package com.ppp.plugin.moderation.config;

import java.util.Optional;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.PluginConfigUpdatedEvent;
import run.halo.app.plugin.SettingFetcher;

/**
 * Reads moderation settings from Halo plugin ConfigMap via SettingFetcher.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class ModerationProperties {

    public static final String GROUP_BASIC = "basic";
    public static final String DEFAULT_FILTER_SERVICE_URL = "http://comment-filter:8091";

    private final SettingFetcher settingFetcher;

    public Settings current() {
        BasicGroup basic = settingFetcher
            .fetch(GROUP_BASIC, BasicGroup.class)
            .orElseGet(BasicGroup::new);

        String envUrl = Optional.ofNullable(System.getenv("FILTER_SERVICE_URL"))
            .map(String::trim)
            .filter(v -> !v.isEmpty())
            .orElse(DEFAULT_FILTER_SERVICE_URL);

        String url = Optional.ofNullable(basic.getFilterServiceUrl())
            .map(String::trim)
            .filter(v -> !v.isEmpty())
            .orElse(envUrl);

        FilterStrategy strategy = FilterStrategy.fromValue(basic.getFilterStrategy());
        boolean enabled = Optional.ofNullable(basic.getEnabled()).orElse(Boolean.TRUE);

        return new Settings(url, strategy, enabled);
    }

    @EventListener
    public void onPluginConfigUpdated(PluginConfigUpdatedEvent event) {
        Settings settings = current();
        log.info("comment moderation settings updated: enabled={}, filterServiceUrl={}, strategy={}",
            settings.enabled(), settings.filterServiceUrl(), settings.filterStrategy());
    }

    @Data
    public static class BasicGroup {
        private String filterServiceUrl = DEFAULT_FILTER_SERVICE_URL;
        private String filterStrategy = FilterStrategy.REJECT.name();
        private Boolean enabled = Boolean.TRUE;
    }

    public enum FilterStrategy {
        REJECT,
        MARK,
        LOG_ONLY;

        public static FilterStrategy fromValue(String value) {
            if (value == null || value.isBlank()) {
                return REJECT;
            }
            try {
                return FilterStrategy.valueOf(value.trim().toUpperCase());
            } catch (IllegalArgumentException ex) {
                return REJECT;
            }
        }
    }

    public record Settings(String filterServiceUrl, FilterStrategy filterStrategy, boolean enabled) {
    }
}
