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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/comment-filter ./cmd

FROM alpine:3.19

# ===== 新增：安装ca-certificates（解决HTTPS请求证书问题） =====
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/comment-filter /usr/local/bin/comment-filter
COPY --from=builder /app/data /app/data

# ===== 新增：调整data目录权限（确保app用户可读取敏感词文件） =====
RUN chown -R app:app /app/data

ENV PORT=8091
ENV WORDS_FILE=/app/data/sensitive_words.txt

USER app
EXPOSE 8091

ENTRYPOINT ["/usr/local/bin/comment-filter"]