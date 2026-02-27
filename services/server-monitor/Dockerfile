FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/server-monitor ./cmd

FROM alpine:3.19

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/server-monitor /usr/local/bin/server-monitor
COPY --from=builder /app/config /app/config

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
