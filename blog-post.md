# Halo 博客系统生产级部署：从 Docker Compose 到 AI 评论审核的完整架构

**TL;DR:** 本文介绍一个基于 Halo 2.x 的博客系统生产级部署方案，包含 Halo 主服务、MySQL、Redis、多语言微服务集群（Go）、Prometheus + Grafana + Loki 监控体系，以及 GitHub Actions 三链路 CI/CD 自动化部署。特别实现了基于敏感词过滤 + MiniMax AI 的评论审核与自动回复系统。

---

## 一、为什么选择 Halo

Halo 是目前最活跃的开源博客框架之一，基于 Spring Boot + Vue 3 构建，支持插件扩展、主题定制，适用于个人博客到企业内容管理多种场景。

本项目构建了一个完整的**生产级部署架构**，解决了以下问题：

- 多服务协同（博客 + 数据库 + 缓存）
- 自定义业务逻辑扩展（评论审核、AI 回复）
- 服务监控与日志收集
- 自动化 CI/CD 部署

> 本项目面向有 Docker 基础、了解基本 CI/CD 概念的开发者在阅读前建议熟悉：Docker Compose、GitHub Actions、REST API 基础。

---

## 二、系统架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              GitHub (代码仓库)                               │
│                         ┌─────────────────────────┐                        │
│                         │   Deploy Full Stack     │ ← 配置文件变更触发      │
│                         │   Deploy Services       │ ← Go 服务变更触发        │
│                         │   Deploy Plugins       │ ← Halo 插件变更触发      │
│                         └──────────┬──────────────┘                        │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │ SSH
                                     ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                           服务器 (Docker Compose)                            │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Halo Network (Bridge)                              │   │
│  │                                                                       │   │
│  │  ┌──────────┐  ┌───────┐  ┌───────┐  ┌──────────────────────────┐  │   │
│  │  │  Halo    │  │ MySQL │  │ Redis │  │   comment-filter (Go)   │  │   │
│  │  │  2.22    │  │  8.0  │  │  7-alpine │  │   评论敏感词+AI审核     │  │   │
│  │  │  :8090   │  │ :3306 │  │ :6379 │  │       :8091             │  │   │
│  │  └──────────┘  └───────┘  └───────┘  └───────────┬──────────────┘  │   │
│  │                                                        │              │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌────────┴────────┐        │   │
│  │  │   auto-reply    │  │  server-monitor │  │ comment-service │        │   │
│  │  │      (Go)       │  │      (Go)       │  │     (Go)       │        │   │
│  │  │  AI自动回复     │  │   系统监控      │  │ Kafka消费者    │        │   │
│  │  │     :8092       │  │     :8093       │  │     :8094       │        │   │
│  │  └─────────────────┘  └─────────────────┘  └────────┬────────┘        │   │
│  │                                                        │                │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────────────┐ │                │   │
│  │  │Prometheus │  │ Grafana  │  │       Loki        │ │                │   │
│  │  │  :9090    │  │  :3000   │  │      :3100       │ │                │   │
│  │  └───────────┘  └───────────┘  └───────────────────┘ │                │   │
│  │                                                        │                │   │
│  │  ┌──────────────────────────────────────────────────┐ │                │   │
│  │  │                   Kafka (KRaft)                 │ │                │   │
│  │  │                    :9092                         │ │                │   │
│  │  └──────────────────────────────────────────────────┘ │                │   │
│  │                                                                       │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────────────────┘
```

### 核心服务说明

| 服务 | 技术栈 | 端口 | 用途 |
|------|--------|------|------|
| Halo | Java 21 / Spring Boot | 8090 | 博客主服务 |
| MySQL | MySQL 8.0 | 3306 | 主数据库（Halo + comment-service） |
| Redis | Redis 7-alpine | 6379 | 缓存、Session、限流 |
| comment-filter | Go | 8091 | 敏感词过滤 + AI 审核 |
| auto-reply | Go | 8092 | AI 自动回复 |
| server-monitor | Go | 8093 | 服务器监控（/proc、/sys） |
| comment-service | Go | 8094 | Kafka 消费者，处理评论事件 |
| Prometheus | Prometheus v2.50 | 9090 | 指标收集 |
| Grafana | Grafana 10.3 | 3000 | 可视化仪表盘 |
| Loki | Loki 2.9.8 | 3100 | 日志聚合 |
| Kafka | Apache Kafka 3.7 | 9092 | 事件流（评论事件） |

---

## 三、Docker Compose 资源配置

Halo JVM 内存配置（768m heap，1g limit）：

```yaml
# halo 服务配置
halo:
  image: halohub/halo:2.22
  environment:
    - JVM_OPTS=-Xms256m -Xmx768m -XX:+UseG1GC -XX:MaxRAMPercentage=75.0
    - SPRING_R2DBC_URL=r2dbc:pool:mysql://mysql:3306/halo
    - SPRING_DATA_REDIS_HOST=redis
    - MANAGEMENT_ENDPOINTS_WEB_EXPOSURE_INCLUDE=health,info,prometheus
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8090/actuator/health/readiness"]
    interval: 30s
    timeout: 10s
    retries: 5
```

关键设计点：

- **健康检查**：Halo 使用 `readiness` 探针，其他服务依赖其状态才启动
- **资源隔离**：各服务设置 `mem_limit` 和 `cpus`，避免单服务抢占资源
- **端口绑定**：MySQL/Redis/Kafka 只绑定 `127.0.0.1`，不对外暴露

---

## 四、自定义微服务实现

### 4.1 评论审核服务（comment-filter）

采用**两级过滤策略**：敏感词先行，AI 复核兜底。

```
用户评论
    ↓
敏感词匹配（字符串匹配，O(n)）
    ↓ 命中 → AI 审核（MiniMax API）→ 允许/拒绝
    ↓ 未命中 → 直接通过
```

```go
// 环境变量
PORT=8091
WORDS_FILE=/app/data/sensitive_words.txt
COMMENT_REVIEW_AI_ENABLED=true
COMMENT_REVIEW_AI_TIMEOUT_SECONDS=30
MINIMAX_API_KEY=xxx
```

关键实现：

- 敏感词库使用 `sensitive_words.txt` 文件管理，支持动态更新
- AI 审核失败默认**拒绝**（安全优先）
- 仅在命中敏感词时才调用 AI，节省 API 调用成本

### 4.2 AI 自动回复服务（auto-reply）

仅对**审核通过**的评论生成 AI 回复：

```
审核通过的评论
    ↓
检查配额（每日 300 次，作者级 20 次）
    ↓ 配额充足 → 调用 MiniMax 生成回复
    ↓ 配额耗尽 → 静默跳过
```

```go
// 限流配置
AUTO_REPLY_DAILY_CALL_LIMIT=300      // 每日总限额
AUTO_REPLY_DAILY_AUTHOR_LIMIT=20    // 每作者限额
AUTO_REPLY_AUTHOR_COOLDOWN_SECONDS=60  // 作者冷却期
```

关键实现：

- 配额使用 Redis 存储，支持平滑限流
- 失败时静默跳过，不向用户暴露错误
- 回复内容长度限制（评论 180 字，回复 120 字）

### 4.3 评论事件服务（comment-service）

基于 Kafka 的异步评论处理架构：

```yaml
# docker-compose.yml
kafka:
  environment:
    - KAFKA_PROCESS_ROLES=broker,controller  # KRaft 模式，无需 ZK
    - KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1

comment-service:
  environment:
    - KAFKA_BROKERS=kafka:9092
    - KAFKA_TOPIC_COMMENT=comment-events
    - KAFKA_CONSUMER_GROUP=comment-service-group
    - SHARD_COUNT=8  # 8 分片并发处理
```

---

## 五、监控体系

### 5.1 指标监控（Prometheus + Grafana）

Halo 暴露 Prometheus 端点：

```yaml
halo:
  environment:
    - MANAGEMENT_ENDPOINT_PROMETHEUS_ENABLED=true
    - MANAGEMENT_METRICS_EXPORT_PROMETHEUS_ENABLED=true
```

服务发现配置（`monitoring/prometheus/prometheus.yml`）：

```yaml
scrape_configs:
  - job_name: "halo"
    static_configs:
      - targets: ["halo:8090"]
  - job_name: "comment-filter"
    static_configs:
      - targets: ["comment-filter:8091"]
  - job_name: "auto-reply"
    static_configs:
      - targets: ["auto-reply:8092"]
```

### 5.2 日志收集（Loki + Promtail）

Docker 容器日志通过 Promtail 收集到 Loki：

```yaml
promtail:
  volumes:
    - /var/lib/docker/containers:/var/lib/docker/containers:ro
```

### 5.3 快速诊断命令

```bash
# 评论审核决策
docker logs --since 10m halo | grep -Ei "comment moderation decision"

# AI 回复流程
docker logs --since 10m halo | grep -Ei "auto-reply created|skip auto-reply"

# AI 服务端日志
docker logs --since 10m auto-reply | grep -Ei "/api/v1/reply|ai generation failed|quota"
```

---

## 六、CI/CD 自动化部署

### 6.1 三链路分离策略

根据变更内容，触发不同的部署流水线，避免不必要的服务重启：

| Workflow | 触发条件 | 部署目标 |
|---------|---------|---------|
| **Deploy Full Stack** | `docker-compose.yml`、`.env`、主题等变更 | Halo + MySQL + Redis + 所有容器 |
| **Deploy Services** | `services/**` 变更 | comment-filter / auto-reply / server-monitor / comment-service |
| **Deploy Plugins** | `plugins/**` 变更 | Halo 插件构建 + 部署 |

### 6.2 Deploy Full Stack（主链路）

```yaml
# .github/workflows/deploy.yml
on:
  push:
    branches: [main, master]
    paths-ignore:
      - "services/**"        # 忽略微服务
      - "plugins/**"          # 忽略插件
      - "docs/dashboard-*"    # 忽略文档

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1.0.0
        with:
          host: ${{ secrets.SERVER_HOST }}
          script: |
            cd /opt/halo-blog
            git fetch --prune origin
            git reset --hard origin/main
            docker-compose pull || true
            docker-compose up -d --remove-orphans
```

### 6.3 Deploy Services（Go 服务链路）

```yaml
# .github/workflows/deploy-services.yml
on:
  push:
    branches: [main]
    paths:
      - services/**
      - .github/workflows/deploy-services.yml

jobs:
  deploy:
    steps:
      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1.0.0
        with:
          script: |
            # 关键：Halo 保持运行，仅重启/重建 Go 服务
            docker-compose up -d --no-deps --build --force-recreate \
              comment-filter auto-reply server-monitor
```

### 6.4 服务健康等待

Services workflow 中包含 Halo 健康检查逻辑：

```bash
wait_halo_ready() {
  local retries=24
  while [ "$retries" -gt 0 ]; do
    health="$(docker inspect -f '{{.State.Health.Status}}' halo)"
    if [ "$health" = "healthy" ] || [ "$health" = "unknown" ]; then
      return 0
    fi
    sleep 5
  done
  return 1
}
```

---

## 七、AI 评论审核与回复流程

### 7.1 完整审核链路

```
用户提交评论
       ↓
Halo 评论事件触发
       ↓
comment-filter 接收请求（POST /api/v1/moderate）
       ↓
敏感词匹配
       ├── 未命中 → 直接允许通过
       └── 命中 → 调用 MiniMax AI 审核
                    ├── 允许 → 写入数据库
                    └── 拒绝 → 返回审核失败
       ↓
auto-reply 监听评论创建事件（仅处理审核通过的评论）
       ↓
检查配额（Redis）
       ├── 配额耗尽 → 静默跳过
       └── 配额充足 → 调用 MiniMax 生成回复
                    ├── 成功 → 创建回复评论
                    └── 失败 → 静默跳过
```

### 7.2 配额管理（Redis）

```go
// 每日全局限额
key := fmt.Sprintf("auto_reply:daily:%s", today)
// INCR + EXPIRE 24h

// 每作者限额
authorKey := fmt.Sprintf("auto_reply:author:%s:%s", author, today)
// INCR + EXPIRE 24h

// 作者冷却期
cooldownKey := fmt.Sprintf("auto_reply:cooldown:%s", author)
// SETNX + EXPIRE 60s
```

---

## 八、项目结构

```
ppp-blog/
├── docker-compose.yml          # 完整服务编排
├── docker-compose.override.yml # 本地开发覆盖
├── .env                       # 环境变量（生产）
├── halo/                      # Halo 主题/配置
├── halo-data/                  # Halo 数据持久化
├── halo-theme-joe2.0/          # Joe3.0 主题
├── services/                   # Go 微服务源码
│   ├── comment-filter/         # 评论审核服务
│   ├── auto-reply/            # AI 自动回复服务
│   ├── comment-service/        # Kafka 消费者
│   └── server-monitor/         # 服务器监控服务
├── plugins/                    # Halo 插件
│   ├── plugin-comment-moderation/
│   ├── plugin-auto-reply/
│   └── plugin-stats-dashboard/
├── monitoring/                # 监控配置
│   ├── prometheus/
│   ├── grafana/
│   └── loki/
├── .github/workflows/
│   ├── deploy.yml             # 主栈部署
│   ├── deploy-services.yml    # Go 服务部署
│   └── deploy-plugins.yml    # 插件部署
├── mysql-data/                # MySQL 数据持久化
├── redis-data/                # Redis 数据持久化
└── kafka-data/                # Kafka 数据持久化
```

---

## 九、环境变量配置

### 核心环境变量（`.env`）

```env
# Halo
HALO_EXTERNAL_URL=https://your-domain.com

# MySQL
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_PASSWORD=your_halo_password

# Redis
REDIS_PASSWORD=your_redis_password

# MiniMax AI
MINIMAX_API_KEY=your_api_key
MINIMAX_API_URL=https://api.minimaxi.com/anthropic/v1/messages
MINIMAX_MODEL=MiniMax-M2.5

# AI 审核
COMMENT_REVIEW_AI_ENABLED=true
COMMENT_REVIEW_AI_TIMEOUT_SECONDS=30

# AI 自动回复
AUTO_REPLY_AI_ENABLED=true
AUTO_REPLY_DAILY_CALL_LIMIT=300
AUTO_REPLY_DAILY_AUTHOR_LIMIT=20
AUTO_REPLY_AUTHOR_COOLDOWN_SECONDS=60
```

---

## 十、Trade-offs 与局限性

| 决策 | Trade-off | 改进方向 |
|------|----------|---------|
| KRaft 无 ZK | 运维简单，但多节点部署受限 | 生产环境建议加 ZK 或 KRaft 多节点 |
| Go 微服务 | 独立部署灵活，但增加复杂度 | 可考虑合并为 Sidecar |
| 敏感词 + AI 两级 | 成本与准确性平衡 | 可添加更多分级策略 |
| Redis 限流 | 实现简单，但重启丢配额 | 可持久化到 MySQL |
| 单机 Docker Compose | 简单高效，但无高可用 | K8s 迁移 |

---

## Further Reading

- [Halo 官方文档](https://docs.halo.run)
- [Docker Compose 文档](https://docs.docker.com/compose/)
- [GitHub Actions 文档](https://docs.github.com/actions)
- [Apache Kafka KRaft 模式](https://developer.confluent.io/tutorials/raft-local/kafka-local.html)
- [Prometheus + Grafana 监控指南](https://prometheus.io/docs/guides/)
