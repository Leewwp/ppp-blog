package com.ppp.plugin.moderation;

import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.BasePlugin;
import run.halo.app.plugin.PluginContext;

/**
 * Main plugin entry for comment moderation.
 */
@Slf4j
@Component
public class CommentModerationPlugin extends BasePlugin {

    public CommentModerationPlugin(PluginContext pluginContext) {
        super(pluginContext);
    }

    @Override
    public void start() {
        log.info("plugin-comment-moderation started");
    }

    @Override
    public void stop() {
        log.info("plugin-comment-moderation stopped");
    }
}
