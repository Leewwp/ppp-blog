FROM golang:1.22-alpine AS builder

WORKDIR /app

# ===== 新增：配置国内Go代理（核心解决依赖下载超时） =====
ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on

# ===== 新增：安装git（避免依赖从Git仓库拉取时失败） =====
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download \
    # ===== 新增：验证依赖完整性（增强构建安全性） =====
    && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/auto-reply ./cmd

FROM alpine:3.19

RUN apk add --no-cache tzdata ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/auto-reply /usr/local/bin/auto-reply
COPY --from=builder /app/config /app/config

# ===== 新增：调整config目录权限（确保app用户可读取规则文件） =====
RUN chown -R app:app /app/config

ENV PORT=8092
ENV RULES_FILE=/app/config/rules.json

USER app
EXPOSE 8092

ENTRYPOINT ["/usr/local/bin/auto-reply"]