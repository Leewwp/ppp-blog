package com.ppp.plugin.autoreply.service.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

@JsonIgnoreProperties(ignoreUnknown = true)
public record AutoReplyResponse(
    @JsonProperty("should_reply")
    boolean shouldReply,
    @JsonProperty("reply_content")
    String replyContent,
    @JsonProperty("delay_seconds")
    Integer delaySeconds,
    @JsonProperty("matched_rule")
    String matchedRule
) {

    public static AutoReplyResponse noReply() {
        return new AutoReplyResponse(false, "", 0, "");
    }
}
