package com.ppp.plugin.autoreply;

import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.BasePlugin;
import run.halo.app.plugin.PluginContext;

/**
 * Main plugin entry for auto reply.
 */
@Slf4j
@Component
public class AutoReplyPlugin extends BasePlugin {

    public AutoReplyPlugin(PluginContext pluginContext) {
        super(pluginContext);
    }

    @Override
    public void start() {
        log.info("plugin-auto-reply started");
    }

    @Override
    public void stop() {
        log.info("plugin-auto-reply stopped");
    }
}
