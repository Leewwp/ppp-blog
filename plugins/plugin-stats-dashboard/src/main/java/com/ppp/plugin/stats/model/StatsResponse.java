package com.ppp.plugin.stats.model;

/**
 * Dashboard summary response model.
 */
public record StatsResponse(
    long onlineVisitors,
    long totalPosts,
    long totalComments,
    long totalViews,
    long todayViews
) {
}
