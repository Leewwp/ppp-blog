FROM golang:1.22-alpine AS builder

WORKDIR /app

ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn

ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB} \
    GO111MODULE=on

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download \
    && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/comment-service ./cmd

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder /out/comment-service /usr/local/bin/comment-service

USER app
EXPOSE 8094

ENTRYPOINT ["/usr/local/bin/comment-service"]
