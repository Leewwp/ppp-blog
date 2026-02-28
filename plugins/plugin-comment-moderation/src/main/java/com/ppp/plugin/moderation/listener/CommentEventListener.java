package com.ppp.plugin.moderation.listener;

import com.ppp.plugin.moderation.config.ModerationProperties;
import com.ppp.plugin.moderation.config.ModerationProperties.FilterStrategy;
import com.ppp.plugin.moderation.service.FilterServiceClient;
import com.ppp.plugin.moderation.service.dto.CommentFilterResponse;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Schedulers;
import run.halo.app.core.extension.content.Comment;
import run.halo.app.extension.Extension;
import run.halo.app.extension.ReactiveExtensionClient;
import run.halo.app.extension.Watcher;

/**
 * Reactive listener for comment creation events.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class CommentEventListener implements Watcher {

    private static final String ANNO_STATUS = "comment-moderation.halo.run/status";
    private static final String ANNO_HIT_WORDS = "comment-moderation.halo.run/hit-words";
    private static final String STATUS_PASS = "PASS";
    private static final String STATUS_LOG_ONLY = "LOG_ONLY";

    private final FilterServiceClient filterServiceClient;
    private final ReactiveExtensionClient extensionClient;
    private final ModerationProperties moderationProperties;
    private volatile boolean disposed = false;
    private Runnable disposeHook;

    @PostConstruct
    public void registerWatcher() {
        extensionClient.watch(this);
        log.info("comment moderation watcher registered");
    }

    @PreDestroy
    public void onDestroy() {
        dispose();
    }

    @Override
    public void onAdd(Extension extension) {
        if (isDisposed() || !(extension instanceof Comment comment)) {
            return;
        }
        onCommentCreated(comment);
    }

    public void onCommentCreated(Comment comment) {
        if (comment == null || comment.getMetadata() == null || comment.getMetadata().getName() == null) {
            log.warn("skip moderation: invalid comment event payload");
            return;
        }

        ModerationProperties.Settings settings = moderationProperties.current();
        if (!settings.enabled()) {
            log.debug("comment moderation is disabled, skip comment={}",
                comment.getMetadata().getName());
            return;
        }

        String commentName = comment.getMetadata().getName();
        String content = safeContent(comment);
        String author = safeAuthor(comment);

        filterServiceClient
            .filter(settings.filterServiceUrl(), content, author)
            .flatMap(result -> applyDecision(commentName, comment, settings.filterStrategy(), result))
            .doOnError(error -> log.error("comment moderation failed, comment={}, error={}",
                commentName, error.toString(), error))
            .onErrorResume(error -> Mono.empty())
            .subscribeOn(Schedulers.boundedElastic())
            .subscribe();
    }

    @Override
    public void registerDisposeHook(Runnable dispose) {
        this.disposeHook = dispose;
    }

    @Override
    public void dispose() {
        disposed = true;
        if (this.disposeHook != null) {
            this.disposeHook.run();
        }
    }

    @Override
    public boolean isDisposed() {
        return this.disposed;
    }

    private Mono<Void> applyDecision(
        String commentName,
        Comment eventComment,
        FilterStrategy strategy,
        CommentFilterResponse result
    ) {
        log.info(
            "comment moderation decision: comment={}, passed={}, strategy={}, hitWords={}",
            commentName,
            result.passed(),
            strategy,
            Optional.ofNullable(result.hitWords()).orElseGet(java.util.List::of)
        );

        if (result.passed()) {
            return markCommentStatus(commentName, eventComment, STATUS_PASS, result);
        }

        return switch (strategy) {
            case LOG_ONLY -> markCommentStatus(commentName, eventComment, STATUS_LOG_ONLY, result);
            case MARK -> updateComment(commentName, eventComment, false, false, "MARK", result);
            case REJECT -> updateComment(commentName, eventComment, false, true, "REJECT", result);
        };
    }

    private Mono<Void> markCommentStatus(String commentName, Comment fallbackComment,
        String moderationStatus, CommentFilterResponse result) {
        return extensionClient
            .fetch(Comment.class, commentName)
            .defaultIfEmpty(fallbackComment)
            .flatMap(comment -> {
                if (comment.getMetadata() != null) {
                    Map<String, String> annotations = comment.getMetadata().getAnnotations();
                    if (annotations == null) {
                        annotations = new HashMap<>();
                        comment.getMetadata().setAnnotations(annotations);
                    }
                    annotations.put(ANNO_STATUS, moderationStatus);
                    annotations.put(ANNO_HIT_WORDS, String.join(",",
                        Optional.ofNullable(result.hitWords()).orElseGet(java.util.List::of)));
                }
                return extensionClient.update(comment);
            })
            .doOnSuccess(updated -> log.debug("comment moderation annotated comment={}, status={}",
                commentName, moderationStatus))
            .onErrorResume(error -> {
                log.warn("failed to annotate moderated comment={}, status={}, error={}",
                    commentName, moderationStatus, error.toString());
                return Mono.empty();
            })
            .then();
    }

    private Mono<Void> updateComment(
        String commentName,
        Comment fallbackComment,
        boolean approved,
        boolean hidden,
        String moderationStatus,
        CommentFilterResponse result
    ) {
        return extensionClient
            .fetch(Comment.class, commentName)
            .defaultIfEmpty(fallbackComment)
            .flatMap(comment -> {
                if (comment.getSpec() == null) {
                    return Mono.error(new IllegalStateException("comment spec is null for " + commentName));
                }

                comment.getSpec().setApproved(approved);
                comment.getSpec().setHidden(hidden);
                if (result.filteredContent() != null && !result.filteredContent().isBlank()) {
                    comment.getSpec().setContent(result.filteredContent());
                }

                if (comment.getMetadata() != null) {
                    Map<String, String> annotations = comment.getMetadata().getAnnotations();
                    if (annotations == null) {
                        annotations = new HashMap<>();
                        comment.getMetadata().setAnnotations(annotations);
                    }
                    annotations.put(ANNO_STATUS, moderationStatus);
                    annotations.put(ANNO_HIT_WORDS, String.join(",",
                        Optional.ofNullable(result.hitWords()).orElseGet(java.util.List::of)));
                }

                return extensionClient.update(comment);
            })
            .doOnSuccess(updated -> log.info("comment moderation updated comment={}, status={}",
                commentName, moderationStatus))
            .doOnError(error -> log.error("failed to update moderated comment={}, status={}, error={}",
                commentName, moderationStatus, error.toString(), error))
            .then();
    }

    private String safeContent(Comment comment) {
        if (comment.getSpec() == null || comment.getSpec().getContent() == null) {
            return "";
        }
        return comment.getSpec().getContent();
    }

    private String safeAuthor(Comment comment) {
        if (comment.getSpec() == null || comment.getSpec().getOwner() == null) {
            return "anonymous";
        }
        String name = comment.getSpec().getOwner().getDisplayName();
        if (name == null || name.isBlank()) {
            name = comment.getSpec().getOwner().getName();
        }
        return name == null || name.isBlank() ? "anonymous" : name;
    }
}
