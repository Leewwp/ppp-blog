package com.ppp.plugin.autoreply.listener;

import com.ppp.plugin.autoreply.config.AutoReplyProperties;
import com.ppp.plugin.autoreply.service.ReplyServiceClient;
import com.ppp.plugin.autoreply.service.dto.AutoReplyRequest;
import com.ppp.plugin.autoreply.service.dto.AutoReplyResponse;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import java.time.Duration;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Schedulers;
import run.halo.app.core.extension.User;
import run.halo.app.core.extension.content.Comment;
import run.halo.app.core.extension.content.Post;
import run.halo.app.core.extension.content.Reply;
import run.halo.app.core.user.service.UserService;
import run.halo.app.extension.Extension;
import run.halo.app.extension.Metadata;
import run.halo.app.extension.ReactiveExtensionClient;
import run.halo.app.extension.Watcher;

/**
 * Reactive listener for auto-reply when a comment is created.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class CommentReplyListener implements Watcher {

    private final ReplyServiceClient replyServiceClient;
    private final ReactiveExtensionClient extensionClient;
    private final AutoReplyProperties autoReplyProperties;
    private volatile boolean disposed = false;
    private Runnable disposeHook;

    @PostConstruct
    public void registerWatcher() {
        extensionClient.watch(this);
        log.info("auto-reply watcher registered");
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
            log.warn("skip auto-reply: invalid comment event payload");
            return;
        }

        AutoReplyProperties.Settings settings = autoReplyProperties.current();
        if (!settings.enabled()) {
            log.debug("auto-reply is disabled, skip comment={}", comment.getMetadata().getName());
            return;
        }

        String commentName = comment.getMetadata().getName();
        String content = safeContent(comment);
        String author = safeAuthor(comment);

        resolvePostTitle(comment)
            .flatMap(postTitle -> {
                AutoReplyRequest request = new AutoReplyRequest(commentName, content, postTitle, author);
                return replyServiceClient.generateReply(settings.replyServiceUrl(), request);
            })
            .flatMap(response -> maybeCreateReply(commentName, settings.replyAuthorName(), response))
            .doOnError(error -> log.error("auto-reply processing failed, comment={}, error={}",
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

    private Mono<Void> maybeCreateReply(String commentName, String replyAuthorName,
        AutoReplyResponse response) {
        if (response == null || !response.shouldReply()) {
            return Mono.empty();
        }

        String replyContent = Optional.ofNullable(response.replyContent())
            .map(String::trim)
            .orElse("");
        if (replyContent.isEmpty()) {
            log.warn("auto-reply response has empty reply_content, skip comment={}, rule={}",
                commentName, response.matchedRule());
            return Mono.empty();
        }

        long delaySeconds = normalizeDelaySeconds(response.delaySeconds());

        return Mono.delay(Duration.ofSeconds(delaySeconds))
            .flatMap(ignore -> createReply(commentName, replyContent, replyAuthorName))
            .doOnSuccess(ignore -> log.info(
                "auto-reply created, comment={}, delaySeconds={}, matchedRule={}",
                commentName, delaySeconds, response.matchedRule()))
            .then();
    }

    private Mono<Reply> createReply(String commentName, String replyContent, String replyAuthorName) {
        Reply reply = new Reply();
        reply.setMetadata(new Metadata());
        reply.getMetadata().setName(UUID.randomUUID().toString());

        Reply.ReplySpec spec = new Reply.ReplySpec();
        reply.setSpec(spec);
        spec.setCommentName(commentName);
        spec.setRaw(replyContent);
        spec.setContent(replyContent);
        spec.setOwner(systemOwner(replyAuthorName));
        spec.setCreationTime(Instant.now());
        spec.setPriority(0);
        spec.setTop(false);
        spec.setAllowNotification(false);
        spec.setApproved(true);
        spec.setApprovedTime(Instant.now());
        spec.setHidden(false);

        return extensionClient.create(reply)
            .doOnError(error -> log.error("failed to create auto-reply, comment={}, error={}",
                commentName, error.toString(), error));
    }

    private Mono<String> resolvePostTitle(Comment comment) {
        if (comment.getSpec() == null || comment.getSpec().getSubjectRef() == null) {
            return Mono.just("");
        }

        String subjectKind = comment.getSpec().getSubjectRef().getKind();
        String subjectName = comment.getSpec().getSubjectRef().getName();
        if (subjectName == null || subjectName.isBlank()) {
            return Mono.just("");
        }

        if (!Post.KIND.equalsIgnoreCase(subjectKind)) {
            return Mono.just(subjectName);
        }

        return extensionClient.fetch(Post.class, subjectName)
            .map(this::safePostTitle)
            .defaultIfEmpty(subjectName)
            .onErrorResume(error -> {
                log.warn("failed to resolve post title, comment={}, post={}, error={}",
                    comment.getMetadata().getName(), subjectName, error.toString());
                return Mono.just(subjectName);
            });
    }

    private String safePostTitle(Post post) {
        if (post == null) {
            return "";
        }
        if (post.getSpec() != null && post.getSpec().getTitle() != null
            && !post.getSpec().getTitle().isBlank()) {
            return post.getSpec().getTitle();
        }
        if (post.getMetadata() != null && post.getMetadata().getName() != null) {
            return post.getMetadata().getName();
        }
        return "";
    }

    private Comment.CommentOwner systemOwner(String replyAuthorName) {
        Comment.CommentOwner owner = new Comment.CommentOwner();
        owner.setKind(User.KIND);
        owner.setName(UserService.GHOST_USER_NAME);
        owner.setDisplayName(replyAuthorName);
        return owner;
    }

    private long normalizeDelaySeconds(Integer delaySeconds) {
        if (delaySeconds == null || delaySeconds < 0) {
            return 0L;
        }
        return delaySeconds;
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
