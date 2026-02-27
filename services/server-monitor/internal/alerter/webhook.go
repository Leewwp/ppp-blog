package alerter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type WebhookNotifier struct {
	webhookURL string
	client     *http.Client
	logger     *slog.Logger
}

func NewWebhookNotifier(webhookURL string, logger *slog.Logger) *WebhookNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookNotifier{
		webhookURL: strings.TrimSpace(webhookURL),
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
		logger: logger,
	}
}

func (n *WebhookNotifier) Enabled() bool {
	return n.webhookURL != ""
}

func (n *WebhookNotifier) Notify(ctx context.Context, event AlertEvent) error {
	if !event.ShouldNotify || !n.Enabled() {
		return nil
	}

	payload := n.buildDingTalkPayload(event)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create webhook request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := n.client.Do(req)
		if err == nil && resp != nil {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			err = fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
		lastErr = err

		if attempt < 3 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	n.logger.Error("webhook notification failed after retries", "error", lastErr, "rule", event.Alert.RuleName, "type", event.Type)
	return fmt.Errorf("notify webhook failed: %w", lastErr)
}

func (n *WebhookNotifier) buildDingTalkPayload(event AlertEvent) map[string]any {
	title := fmt.Sprintf("[%s] %s", strings.ToUpper(event.Alert.Severity), event.Alert.RuleName)
	text := buildMarkdownText(event)

	return map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}
}

func buildMarkdownText(event AlertEvent) string {
	a := event.Alert
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("### %s 告警\n\n", strings.ToUpper(event.Type)))
	builder.WriteString(fmt.Sprintf("- 规则: `%s`\n", a.RuleName))
	builder.WriteString(fmt.Sprintf("- 级别: `%s`\n", a.Severity))
	builder.WriteString(fmt.Sprintf("- 指标: `%s`\n", a.Metric))
	builder.WriteString(fmt.Sprintf("- 条件: `%s %.2f`\n", a.Operator, a.Threshold))
	builder.WriteString(fmt.Sprintf("- 当前值: `%.2f`\n", a.Current))
	builder.WriteString(fmt.Sprintf("- 触发时间: `%s`\n", a.TriggeredAt.Format(time.RFC3339)))
	if !a.ResolvedAt.IsZero() {
		builder.WriteString(fmt.Sprintf("- 恢复时间: `%s`\n", a.ResolvedAt.Format(time.RFC3339)))
	}
	return builder.String()
}
