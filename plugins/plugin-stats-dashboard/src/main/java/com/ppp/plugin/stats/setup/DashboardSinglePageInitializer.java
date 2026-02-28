package com.ppp.plugin.stats.setup;

import jakarta.annotation.PostConstruct;
import java.time.Instant;
import java.util.HashMap;
import java.util.LinkedHashSet;
import java.util.List;
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
    private static final String PAGE_SLUG = "stats-dashboard";
    private static final String PAGE_TITLE = "数据看板";
    private static final String MANAGED_ANNO = "stats-dashboard.halo.run/managed";

    private static final String DASHBOARD_HTML = """
        <section style="max-width:960px;margin:0 auto;padding:24px;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
          <h2 style="margin:0 0 16px;">站点数据看板</h2>
          <p style="margin:0 0 20px;color:#555;">由 plugin-stats-dashboard 提供，数据每次刷新实时拉取。</p>
          <div id="stats-cards" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:12px;margin-bottom:24px;"></div>
          <h3 style="margin:0 0 12px;">最近 7 天访客趋势</h3>
          <div id="stats-history" style="border:1px solid #e5e7eb;border-radius:10px;padding:12px;background:#fafafa;"></div>
          <p id="stats-error" style="display:none;color:#b91c1c;margin-top:16px;"></p>
        </section>
        <script>
          const cardsEl = document.getElementById("stats-cards");
          const historyEl = document.getElementById("stats-history");
          const errorEl = document.getElementById("stats-error");

          const cardDefs = [
            ["onlineVisitors", "在线访客"],
            ["todayViews", "今日 PV"],
            ["totalPosts", "文章总数"],
            ["totalComments", "评论总数"],
            ["totalViews", "总访问量"]
          ];

          function renderCards(stats) {
            cardsEl.innerHTML = cardDefs.map(([key, label]) => {
              const value = Number(stats[key] ?? 0).toLocaleString();
              return `<div style="border:1px solid #e5e7eb;border-radius:10px;padding:14px;background:#fff;">
                        <div style="font-size:12px;color:#6b7280;">${label}</div>
                        <div style="margin-top:8px;font-size:26px;font-weight:700;color:#111827;">${value}</div>
                      </div>`;
            }).join("");
          }

          function renderHistory(history) {
            if (!Array.isArray(history) || history.length === 0) {
              historyEl.innerHTML = "<div style='color:#6b7280;'>暂无历史数据</div>";
              return;
            }
            historyEl.innerHTML = history.map(item => {
              const date = item.date ?? "-";
              const views = Number(item.views ?? 0).toLocaleString();
              return `<div style="display:flex;justify-content:space-between;padding:6px 0;border-bottom:1px dashed #e5e7eb;">
                        <span>${date}</span><strong>${views}</strong>
                      </div>`;
            }).join("");
          }

          async function loadStats() {
            try {
              const [statsRes, historyRes] = await Promise.all([
                fetch("/apis/ppp.run/v1alpha1/stats/public", { credentials: "include" }),
                fetch("/apis/ppp.run/v1alpha1/stats/history/public", { credentials: "include" })
              ]);
              if (!statsRes.ok || !historyRes.ok) {
                throw new Error(`stats api failed: ${statsRes.status}/${historyRes.status}`);
              }
              const stats = await statsRes.json();
              const history = await historyRes.json();
              renderCards(stats);
              renderHistory(history.history || []);
            } catch (err) {
              errorEl.style.display = "block";
              errorEl.textContent = "看板数据加载失败，请检查插件和接口状态。";
              console.error(err);
            }
          }

          loadStats();
        </script>
        """;

    private final ReactiveExtensionClient extensionClient;

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
            .switchIfEmpty(Mono.defer(this::createPageWithContent));
    }

    private Mono<SinglePage> createPageWithContent() {
        SinglePage page = buildPage();
        return extensionClient.create(page)
            .flatMap(created -> createBaseSnapshot(created)
                .flatMap(snapshot -> {
                    String snapshotName = snapshot.getMetadata().getName();
                    created.getSpec().setBaseSnapshot(snapshotName);
                    created.getSpec().setHeadSnapshot(snapshotName);
                    created.getSpec().setReleaseSnapshot(snapshotName);
                    created.getSpec().setPublishTime(Instant.now());
                    return extensionClient.update(created);
                }));
    }

    private SinglePage buildPage() {
        SinglePage page = new SinglePage();

        Metadata metadata = new Metadata();
        metadata.setName(PAGE_NAME);
        metadata.setAnnotations(new HashMap<>());
        metadata.getAnnotations().put(MANAGED_ANNO, Boolean.TRUE.toString());
        page.setMetadata(metadata);

        SinglePage.SinglePageSpec spec = new SinglePage.SinglePageSpec();
        spec.setTitle(PAGE_TITLE);
        spec.setSlug(PAGE_SLUG);
        spec.setOwner(UserService.GHOST_USER_NAME);
        spec.setDeleted(Boolean.FALSE);
        spec.setPublish(Boolean.TRUE);
        spec.setPinned(Boolean.FALSE);
        spec.setAllowComment(Boolean.FALSE);
        spec.setVisible(Post.VisibleEnum.PUBLIC);
        spec.setPriority(0);

        Post.Excerpt excerpt = new Post.Excerpt();
        excerpt.setAutoGenerate(Boolean.TRUE);
        excerpt.setRaw("");
        spec.setExcerpt(excerpt);

        page.setSpec(spec);
        return page;
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
        spec.setRawPatch(DASHBOARD_HTML);
        spec.setContentPatch(DASHBOARD_HTML);
        spec.setLastModifyTime(Instant.now());
        spec.setOwner(UserService.GHOST_USER_NAME);
        spec.setContributors(new LinkedHashSet<>(List.of(UserService.GHOST_USER_NAME)));
        snapshot.setSpec(spec);

        return extensionClient.create(snapshot);
    }
}
