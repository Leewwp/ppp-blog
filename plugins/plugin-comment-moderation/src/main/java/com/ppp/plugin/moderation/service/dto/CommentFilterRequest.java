package com.ppp.plugin.moderation.service.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record CommentFilterRequest(
    @JsonProperty("content")
    String content,
    @JsonProperty("author")
    String author
) {
}
