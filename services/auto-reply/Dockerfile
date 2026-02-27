FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/auto-reply ./cmd

FROM alpine:3.19

RUN apk add --no-cache tzdata ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/auto-reply /usr/local/bin/auto-reply
COPY --from=builder /app/config /app/config

ENV PORT=8092
ENV RULES_FILE=/app/config/rules.json

USER app
EXPOSE 8092

ENTRYPOINT ["/usr/local/bin/auto-reply"]
