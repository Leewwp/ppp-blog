package com.ppp.plugin.stats.setup;

import jakarta.annotation.PostConstruct;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.HashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.UUID;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import run.halo.app.core.extension.content.Post;
import run.halo.app.core.extension.content.SinglePage;
import run.halo.app.core.extension.content.Snapshot;
import run.halo.app.core.user.service.UserService;
import run.halo.app.extension.Metadata;
import run.halo.app.extension.ReactiveExtensionClient;
import run.halo.app.extension.Ref;

/**
 * Ensures a manageable single-page dashboard exists in Halo console.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class DashboardSinglePageInitializer {

    private static final String PAGE_NAME = "stats-dashboard-page";
    private static final String PAGE_SLUG = "dashboard";
    private static final String PAGE_TITLE = "数据看板";
    private static final String PAGE_TEMPLATE = "sheet_only_header_footer";
    private static final String MANAGED_ANNO = "stats-dashboard.halo.run/managed";

    private static final String DASHBOARD_RESOURCE_PATH = "/dashboard/dashboard-sheet-code.html";

    private final ReactiveExtensionClient extensionClient;
    private final String dashboardHtml = loadDashboardHtml();

    @PostConstruct
    public void ensureDashboardSinglePage() {
        ensurePage()
            .doOnSuccess(page -> log.info("stats dashboard page ready, name={}, slug={}",
                page.getMetadata().getName(), page.getSpec().getSlug()))
            .doOnError(error -> log.error("failed to ensure stats dashboard page: {}",
                error.toString(), error))
            .onErrorResume(error -> Mono.empty())
            .subscribe();
    }

    private Mono<SinglePage> ensurePage() {
        return extensionClient.fetch(SinglePage.class, PAGE_NAME)
            .flatMap(this::syncExistingPage)
            .switchIfEmpty(Mono.defer(this::createPageWithContent));
    }

    private Mono<SinglePage> syncExistingPage(SinglePage page) {
        boolean changed = normalizePage(page);
        Mono<SinglePage> pageMono = changed ? extensionClient.update(page) : Mono.just(page);

        return needsSnapshotRefresh(page)
            .flatMap(needsRefresh -> {
                if (!needsRefresh) {
                    return pageMono;
                }
                return pageMono.flatMap(this::replaceSnapshot);
            });
    }

    private Mono<Boolean> needsSnapshotRefresh(SinglePage page) {
        String snapshotName = firstNonBlank(
            page.getSpec().getHeadSnapshot(),
            page.getSpec().getReleaseSnapshot(),
            page.getSpec().getBaseSnapshot()
        );
        if (snapshotName == null) {
            return Mono.just(true);
        }

        return extensionClient.fetch(Snapshot.class, snapshotName)
            .map(snapshot -> shouldRefreshSnapshot(snapshot.getSpec().getRawPatch()))
            .defaultIfEmpty(true);
    }

    private boolean shouldRefreshSnapshot(String rawPatch) {
        if (rawPatch == null || rawPatch.isBlank()) {
            return true;
        }
        return !dashboardHtml.equals(rawPatch);
    }

    private Mono<SinglePage> createPageWithContent() {
        SinglePage page = buildPage();
        return extensionClient.create(page).flatMap(this::replaceSnapshot);
    }

    private Mono<SinglePage> replaceSnapshot(SinglePage page) {
        return createBaseSnapshot(page)
            .flatMap(snapshot -> {
                String snapshotName = snapshot.getMetadata().getName();
                page.getSpec().setBaseSnapshot(snapshotName);
                page.getSpec().setHeadSnapshot(snapshotName);
                page.getSpec().setReleaseSnapshot(snapshotName);
                page.getSpec().setPublishTime(Instant.now());
                return extensionClient.update(page);
            });
    }

    private SinglePage buildPage() {
        SinglePage page = new SinglePage();

        Metadata metadata = new Metadata();
        metadata.setName(PAGE_NAME);
        metadata.setAnnotations(new HashMap<>());
        page.setMetadata(metadata);

        SinglePage.SinglePageSpec spec = new SinglePage.SinglePageSpec();
        spec.setExcerpt(new Post.Excerpt());
        page.setSpec(spec);

        normalizePage(page);
        return page;
    }

    private boolean normalizePage(SinglePage page) {
        boolean changed = false;

        Metadata metadata = page.getMetadata();
        if (metadata == null) {
            metadata = new Metadata();
            page.setMetadata(metadata);
            changed = true;
        }
        if (!Objects.equals(metadata.getName(), PAGE_NAME)) {
            metadata.setName(PAGE_NAME);
            changed = true;
        }
        Map<String, String> annotations = metadata.getAnnotations();
        if (annotations == null) {
            annotations = new HashMap<>();
            metadata.setAnnotations(annotations);
            changed = true;
        }
        if (!Objects.equals(annotations.get(MANAGED_ANNO), Boolean.TRUE.toString())) {
            annotations.put(MANAGED_ANNO, Boolean.TRUE.toString());
            changed = true;
        }

        SinglePage.SinglePageSpec spec = page.getSpec();
        if (spec == null) {
            spec = new SinglePage.SinglePageSpec();
            page.setSpec(spec);
            changed = true;
        }
        changed |= setIfDifferent(spec.getTitle(), PAGE_TITLE, spec::setTitle);
        changed |= setIfDifferent(spec.getSlug(), PAGE_SLUG, spec::setSlug);
        changed |= setIfDifferent(spec.getTemplate(), PAGE_TEMPLATE, spec::setTemplate);
        changed |= setIfDifferent(spec.getOwner(), UserService.GHOST_USER_NAME, spec::setOwner);
        changed |= setIfDifferent(spec.getDeleted(), Boolean.FALSE, spec::setDeleted);
        changed |= setIfDifferent(spec.getPublish(), Boolean.TRUE, spec::setPublish);
        changed |= setIfDifferent(spec.getPinned(), Boolean.FALSE, spec::setPinned);
        changed |= setIfDifferent(spec.getAllowComment(), Boolean.FALSE, spec::setAllowComment);
        changed |= setIfDifferent(spec.getVisible(), Post.VisibleEnum.PUBLIC, spec::setVisible);
        changed |= setIfDifferent(spec.getPriority(), 0, spec::setPriority);

        Post.Excerpt excerpt = spec.getExcerpt();
        if (excerpt == null) {
            excerpt = new Post.Excerpt();
            spec.setExcerpt(excerpt);
            changed = true;
        }
        changed |= setIfDifferent(excerpt.getAutoGenerate(), Boolean.TRUE, excerpt::setAutoGenerate);
        changed |= setIfDifferent(excerpt.getRaw(), "", excerpt::setRaw);

        return changed;
    }

    private <T> boolean setIfDifferent(T current, T expected, java.util.function.Consumer<T> setter) {
        if (Objects.equals(current, expected)) {
            return false;
        }
        setter.accept(expected);
        return true;
    }

    private Mono<Snapshot> createBaseSnapshot(SinglePage page) {
        Snapshot snapshot = new Snapshot();

        Metadata metadata = new Metadata();
        metadata.setName(UUID.randomUUID().toString());
        metadata.setAnnotations(new HashMap<>());
        metadata.getAnnotations().put(Snapshot.KEEP_RAW_ANNO, Boolean.TRUE.toString());
        snapshot.setMetadata(metadata);

        Snapshot.SnapShotSpec spec = new Snapshot.SnapShotSpec();
        spec.setSubjectRef(Ref.of(page));
        spec.setRawType("html");
        spec.setRawPatch(dashboardHtml);
        spec.setContentPatch(dashboardHtml);
        spec.setLastModifyTime(Instant.now());
        spec.setOwner(UserService.GHOST_USER_NAME);
        spec.setContributors(new LinkedHashSet<>(List.of(UserService.GHOST_USER_NAME)));
        snapshot.setSpec(spec);

        return extensionClient.create(snapshot);
    }

    private String loadDashboardHtml() {
        try (InputStream inputStream =
                 DashboardSinglePageInitializer.class.getResourceAsStream(DASHBOARD_RESOURCE_PATH)) {
            if (inputStream == null) {
                throw new IllegalStateException(
                    "Dashboard template resource not found: " + DASHBOARD_RESOURCE_PATH
                );
            }
            return new String(inputStream.readAllBytes(), StandardCharsets.UTF_8);
        } catch (IOException ex) {
            throw new IllegalStateException(
                "Failed to load dashboard template resource: " + DASHBOARD_RESOURCE_PATH,
                ex
            );
        }
    }

    private String firstNonBlank(String... values) {
        for (String value : values) {
            if (value != null && !value.isBlank()) {
                return value;
            }
        }
        return null;
    }
}
