# Manual Configuration Checklist

This repository now includes code-level support for:
- configurable auto-reply author identity (name/email/avatar)
- per-user like state isolation and unlike handling
- request `TraceID` in Go services (`X-Trace-Id`)
- Loki + Promtail + Grafana logs
- optional SkyWalking tracing stack (Docker Compose profile: `tracing`)

Complete the following manual steps on server/console.

## 1. Halo comment and identity settings

1. Enable guest comments if you want non-login users to comment.
- Halo Console -> Comment settings.
- Turn off `systemUserOnly` (or equivalent "only registered users can comment").

2. Decide review behavior:
- If you want auto-reply immediately after posting, disable "comment requires review".
- If review is enabled, auto-reply waits until moderation result is available and comment becomes visible.

3. In plugin settings:
- `plugin-comment-moderation`: keep `enabled=true`, `filterServiceUrl=http://comment-filter:8091`, strategy `REJECT`.
- `plugin-auto-reply`: set:
  - `replyServiceUrl=http://auto-reply:8092`
  - `replyAuthorName=博主` (or your desired name)
  - `replyAuthorEmail` (stable identity email)
  - `replyAuthorAvatar` (public avatar URL, optional)

## 2. User registration (GitHub + email verification)

These are Halo console integrations, not custom plugin code.

1. GitHub registration/login:
- Create GitHub OAuth App:
  - Authorization callback URL: `https://<your-domain>/login/oauth2/code/github`
- In Halo auth provider settings, add GitHub provider with:
  - Client ID
  - Client Secret
- Enable self-service registration if your policy allows it.

2. Email verification registration:
- Configure SMTP in Halo (mail server, username, password, sender).
- Enable email verification for registration/login flows.
- Test by creating a new user and confirming verification email is delivered.

## 3. AI moderation and AI auto-reply runtime config

Set in server `.env`:

```env
MINIMAX_API_KEY=...
MINIMAX_API_URL=https://api.minimaxi.com/anthropic/v1/messages
MINIMAX_MODEL=MiniMax-M2.5

COMMENT_REVIEW_AI_ENABLED=true
COMMENT_REVIEW_AI_TIMEOUT_SECONDS=30
COMMENT_REVIEW_AI_MAX_CONTENT_CHARS=500

AUTO_REPLY_AI_ENABLED=true
AUTO_REPLY_AI_TIMEOUT_SECONDS=30
AUTO_REPLY_MAX_COMMENT_CHARS=180
AUTO_REPLY_MAX_REPLY_CHARS=120
AUTO_REPLY_DAILY_CALL_LIMIT=300
AUTO_REPLY_DAILY_AUTHOR_LIMIT=20
AUTO_REPLY_AUTHOR_COOLDOWN_SECONDS=60
```

Apply:

```bash
docker-compose up -d --no-deps --build --force-recreate comment-filter auto-reply
```

## 4. Rule management endpoints

1. Comment-filter sensitive words:
- `GET /api/v1/words`
- `POST /api/v1/words`
- `DELETE /api/v1/words`

2. Auto-reply rules:
- `GET /api/v1/rules`
- `POST /api/v1/rules`
- `PUT /api/v1/rules/{id}`
- `DELETE /api/v1/rules/{id}`

## 5. Observability stack (logs + metrics + tracing)

1. Start metrics + logs:

```bash
docker-compose up -d prometheus grafana loki promtail
```

2. Start SkyWalking (optional, profile-based):

```bash
docker-compose --profile tracing up -d skywalking-oap skywalking-ui
```

TTL note for SkyWalking 9.7.0:
- `SW_CORE_METRICS_DATA_TTL` must be >= `2`.
- `SW_CORE_RECORD_DATA_TTL` should also be >= `2`.

3. Download SkyWalking Java agent (required only if you want Halo traces in SkyWalking):

```bash
mkdir -p monitoring/skywalking/agent
cd monitoring/skywalking/agent
# Use a valid Java agent release (for SkyWalking OAP 9.7.0, 9.1.0 is safe).
curl -fL -o agent.tgz https://archive.apache.org/dist/skywalking/java-agent/9.1.0/apache-skywalking-java-agent-9.1.0.tgz
tar -xzf agent.tgz --strip-components=1
rm -f agent.tgz
test -f skywalking-agent.jar
```

4. Enable Halo Java agent in `.env`:

```env
HALO_JVM_OPTS=-Xms256m -Xmx768m -XX:+UseG1GC -XX:MaxRAMPercentage=75.0 -javaagent:/opt/skywalking/agent/skywalking-agent.jar
SW_AGENT_NAME=halo
SW_AGENT_COLLECTOR_BACKEND_SERVICES=skywalking-oap:11800
```

5. Recreate Halo:

```bash
docker-compose up -d --no-deps --force-recreate halo
```

## 6. Grafana alerts (email / GitHub webhook)

1. Email alerting:
- Set SMTP env values in `.env` (`GF_SMTP_*`).
- Restart grafana.
- In Grafana Alerting UI, create Contact Point (Email) and Notification Policy.

2. GitHub notification:
- Option A: use Grafana Webhook contact point -> GitHub Actions workflow_dispatch endpoint proxy.
- Option B: use `server-monitor` `WEBHOOK_URL` to send alert payload to your own relay service (recommended).

## 7. Verification commands

1. Auto-reply author identity:

```bash
docker logs --since 10m halo | grep -Ei "auto-reply created|failed to create auto-reply"
```

2. TraceID in Go service logs:

```bash
docker logs --since 5m auto-reply | grep -Ei "trace_id|http request"
docker logs --since 5m comment-filter | grep -Ei "trace_id|http request"
docker logs --since 5m server-monitor | grep -Ei "trace_id|http request"
```

3. Loki datasource connectivity:
- Grafana -> Connections -> Data sources -> Loki should be healthy.

4. SkyWalking UI:
- `http://<server-ip>:8081`

Notes:
- `curl -I http://127.0.0.1:8081` may return `405 Method Not Allowed` because UI route doesn't support HEAD.
- Use `curl http://127.0.0.1:8081` (GET) or open the URL in browser.
- OAP health endpoint is lowercase: `http://127.0.0.1:12800/healthcheck`
