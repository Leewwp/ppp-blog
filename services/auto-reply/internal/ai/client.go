package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	ErrDisabled      = errors.New("ai client is disabled")
	ErrQuotaExceeded = errors.New("ai quota exceeded")
)

type ClientConfig struct {
	Enabled       bool
	APIKey        string
	APIURL        string
	Model         string
	Timeout       time.Duration
	MaxReplyChars int
}

type GenerateRequest struct {
	Author         string
	PostTitle      string
	CommentContent string
	RuleHint       string
}

type Client struct {
	cfg        ClientConfig
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(cfg ClientConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxReplyChars <= 0 {
		cfg.MaxReplyChars = 120
	}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.APIURL) == "" {
		cfg.Enabled = false
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled
}

func (c *Client) GenerateReply(ctx context.Context, req GenerateRequest) (string, error) {
	if !c.cfg.Enabled {
		return "", ErrDisabled
	}

	userPrompt := fmt.Sprintf(
		"文章标题：%s\n评论作者：%s\n评论内容：%s\n参考风格：%s\n\n请生成一段简短中文回复。",
		safeTrim(req.PostTitle, 80),
		safeTrim(req.Author, 40),
		safeTrim(req.CommentContent, 300),
		safeTrim(req.RuleHint, 120),
	)

	payload := map[string]any{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "你是博客作者助手。请输出自然中文，1-2句话，简洁直接，不要提及系统或额度，不要输出 markdown。",
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"temperature":        0.2,
		"max_tokens":         160,
		"tokens_to_generate": 160,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal ai payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create ai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request ai api: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ai response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || isQuotaError(string(respBytes)) {
		return "", ErrQuotaExceeded
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai api status %d: %s", resp.StatusCode, safeTrim(string(respBytes), 300))
	}

	content, err := extractContent(respBytes)
	if err != nil {
		return "", err
	}
	content = sanitizeReply(content, c.cfg.MaxReplyChars)
	if content == "" {
		return "", errors.New("empty ai reply content")
	}
	return content, nil
}

func extractContent(raw []byte) (string, error) {
	var resp map[string]any
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("unmarshal ai response: %w", err)
	}

	if text := tryPathString(resp, "choices", 0, "message", "content"); text != "" {
		return text, nil
	}
	if text := tryPathString(resp, "reply"); text != "" {
		return text, nil
	}
	if text := tryPathString(resp, "output_text"); text != "" {
		return text, nil
	}
	if text := tryPathString(resp, "data", "reply"); text != "" {
		return text, nil
	}
	if text := tryPathString(resp, "base_resp", "status_msg"); text != "" {
		return "", fmt.Errorf("ai response has no content, status_msg=%s", text)
	}
	return "", errors.New("ai response has no textual content")
}

func tryPathString(data any, path ...any) string {
	current := data
	for _, p := range path {
		switch key := p.(type) {
		case string:
			obj, ok := current.(map[string]any)
			if !ok {
				return ""
			}
			current, ok = obj[key]
			if !ok {
				return ""
			}
		case int:
			arr, ok := current.([]any)
			if !ok || key < 0 || key >= len(arr) {
				return ""
			}
			current = arr[key]
		default:
			return ""
		}
	}
	str, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

func sanitizeReply(reply string, maxChars int) string {
	reply = strings.TrimSpace(reply)
	reply = strings.ReplaceAll(reply, "\n", " ")
	reply = strings.ReplaceAll(reply, "\r", " ")
	reply = strings.Join(strings.Fields(reply), " ")
	if maxChars <= 0 {
		return reply
	}
	runes := []rune(reply)
	if len(runes) > maxChars {
		reply = string(runes[:maxChars])
	}
	return strings.TrimSpace(reply)
}

func safeTrim(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) > maxChars {
		return string(runes[:maxChars])
	}
	return value
}

func isQuotaError(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "quota") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "insufficient")
}
