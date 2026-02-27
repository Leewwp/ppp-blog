FROM golang:1.22-alpine AS builder

WORKDIR /app

# ===== 新增：配置国内Go代理（解决依赖下载超时） =====
ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on

# ===== 可选新增：安装git（避免部分依赖需要git拉取时失败） =====
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download \
    # ===== 可选新增：验证依赖完整性（增强构建安全性） =====
    && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/server-monitor ./cmd

FROM alpine:3.19

# ===== 新增：安装ca-certificates（解决HTTPS请求失败问题） =====
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/server-monitor /usr/local/bin/server-monitor
COPY --from=builder /app/config /app/config

# ===== 新增：调整config目录权限（确保非root用户可读取） =====
RUN chown -R app:app /app/config

ENV PORT=8093
ENV COLLECT_INTERVAL_SECONDS=30
ENV ALERT_COOLDOWN_MINUTES=5
ENV ALERT_RULES_FILE=/app/config/alert_rules.yaml
ENV HOST_PROC=/host/proc
ENV HOST_SYS=/host/sys

# Run container with:
# -v /proc:/host/proc:ro -v /sys:/host/sys:ro
VOLUME ["/host/proc", "/host/sys"]

USER app
EXPOSE 8093

ENTRYPOINT ["/usr/local/bin/server-monitor"]