package com.ppp.plugin.stats;

import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.BasePlugin;
import run.halo.app.plugin.PluginContext;

/**
 * Main entry for the stats dashboard plugin.
 */
@Slf4j
@Component
public class StatsDashboardPlugin extends BasePlugin {

    public StatsDashboardPlugin(PluginContext pluginContext) {
        super(pluginContext);
    }

    @Override
    public void start() {
        log.info("plugin-stats-dashboard started");
    }

    @Override
    public void stop() {
        log.info("plugin-stats-dashboard stopped");
    }
}
