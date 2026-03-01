# AI Moderation and Auto-Reply Guide

This project implements:
- sensitive-word filtering first
- AI re-check only when sensitive words are hit
- AI-generated auto-reply only for moderation-passed comments
- quota and cooldown limits to control model costs

## 1) Required server env

Set in `.env`:

```env
MINIMAX_API_KEY=your_real_key
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

Apply changes:

```bash
docker-compose up -d --no-deps --build --force-recreate comment-filter auto-reply
```

## 2) Plugin settings in Halo console

### plugin-comment-moderation
- `enabled`
- `filterServiceUrl` (default: `http://comment-filter:8091`)
- `filterStrategy` (recommend `REJECT`)

### plugin-auto-reply
- `enabled`
- `replyServiceUrl` (default: `http://auto-reply:8092`)
- `replyAuthorName` (display name, e.g. `²©Ö÷`)
- `replyAuthorEmail` (stable owner identity)
- `replyAuthorAvatar` (optional avatar URL)

## 3) Rule management APIs

### comment-filter words
- `GET /api/v1/words`
- `POST /api/v1/words`
- `DELETE /api/v1/words`

### auto-reply rules
- `GET /api/v1/rules`
- `POST /api/v1/rules`
- `PUT /api/v1/rules/{id}`
- `DELETE /api/v1/rules/{id}`

## 4) Behavior and limits

1. If no sensitive word is matched, moderation passes directly.
2. If sensitive words are matched:
- AI reviewer decides allow/reject.
- if AI fails, default is reject for safety.
3. Auto-reply only runs on moderation-passed comments.
4. If AI quota/network/auth fails, auto-reply is skipped silently.
5. When daily quota is exhausted, system does NOT reply with error text to visitors.

## 5) Quick diagnostics

```bash
# comment moderation decisions
docker logs --since 10m halo | grep -Ei "comment moderation decision|annotated comment"

# auto-reply plugin pipeline
docker logs --since 10m halo | grep -Ei "auto-reply created|skip auto-reply|failed to create auto-reply"

# auto-reply service AI side
docker logs --since 10m auto-reply | grep -Ei "/api/v1/reply|ai generation failed|quota|cooldown"
```
