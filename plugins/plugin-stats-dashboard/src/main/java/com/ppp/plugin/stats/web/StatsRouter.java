package com.ppp.plugin.stats.web;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;
import org.springframework.web.reactive.function.server.RouterFunction;
import org.springframework.web.reactive.function.server.RouterFunctions;
import org.springframework.web.reactive.function.server.ServerResponse;
import run.halo.app.core.extension.endpoint.CustomEndpoint;
import run.halo.app.extension.GroupVersion;

/**
 * Router definition for stats dashboard APIs.
 */
@Component
@RequiredArgsConstructor
public class StatsRouter implements CustomEndpoint {

    private final StatsHandler statsHandler;

    @Override
    public RouterFunction<ServerResponse> endpoint() {
        return RouterFunctions.route()
            .GET("/stats", statsHandler::getStats)
            .GET("/stats/history", statsHandler::getHistory)
            .build();
    }

    @Override
    public GroupVersion groupVersion() {
        return new GroupVersion("ppp.run", "v1alpha1");
    }
}
