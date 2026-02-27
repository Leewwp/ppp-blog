FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/comment-filter ./cmd

FROM alpine:3.19

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/comment-filter /usr/local/bin/comment-filter
COPY --from=builder /app/data /app/data

ENV PORT=8091
ENV WORDS_FILE=/app/data/sensitive_words.txt

USER app
EXPOSE 8091

ENTRYPOINT ["/usr/local/bin/comment-filter"]
