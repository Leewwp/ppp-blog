package com.ppp.plugin.autoreply.service.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record AutoReplyRequest(
    @JsonProperty("comment_id")
    String commentId,
    @JsonProperty("content")
    String content,
    @JsonProperty("post_title")
    String postTitle,
    @JsonProperty("author")
    String author
) {
}
