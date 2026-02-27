package com.ppp.plugin.moderation.service.dto;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

@JsonIgnoreProperties(ignoreUnknown = true)
public record CommentFilterResponse(
    @JsonProperty("passed")
    boolean passed,
    @JsonProperty("filtered_content")
    String filteredContent,
    @JsonProperty("hit_words")
    List<String> hitWords
) {

    public static CommentFilterResponse passThrough(String content) {
        return new CommentFilterResponse(true, content, List.of());
    }
}
