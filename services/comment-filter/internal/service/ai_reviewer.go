package service

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

var ErrReviewDisabled = errors.New("ai review is disabled")

type ReviewerConfig struct {
	Enabled        bool
	APIKey         string
	APIURL         string
	Model          string
	Timeout        time.Duration
	MaxContentChar int
}

type ReviewDecision struct {
	Allow  bool
	Reason string
}

type AIReviewer struct {
	cfg        ReviewerConfig
	httpClient *http.Client
	logger     *slog.Logger
}

const anthropicVersion = "2023-06-01"

func NewAIReviewer(cfg ReviewerConfig, logger *slog.Logger) *AIReviewer {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.MaxContentChar <= 0 {
		cfg.MaxContentChar = 500
	}
	cfg.APIKey = sanitizeAPIKey(cfg.APIKey)
	cfg.APIURL = strings.TrimSpace(cfg.APIURL)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		cfg.Model = "MiniMax-Text-01"
	}

	if cfg.APIKey == "" || cfg.APIURL == "" {
		cfg.Enabled = false
	}
	return &AIReviewer{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}
}

func (r *AIReviewer) Enabled() bool {
	return r.cfg.Enabled
}

func (r *AIReviewer) Review(content, author string, hitWords []string) (ReviewDecision, error) {
	if !r.cfg.Enabled {
		return ReviewDecision{}, ErrReviewDisabled
	}
	content = strings.TrimSpace(content)
	if len([]rune(content)) > r.cfg.MaxContentChar {
		return ReviewDecision{
			Allow:  false,
			Reason: "content too long for review",
		}, nil
	}

	prompt := fmt.Sprintf(
		"评论作者：%s\n命中词：%s\n评论内容：%s\n\n请判断该评论是否属于误判可放行。仅输出 JSON：{\"allow\":true/false,\"reason\":\"...\"}",
		safeCut(author, 40),
		safeCut(strings.Join(hitWords, ","), 120),
		safeCut(content, r.cfg.MaxContentChar),
	)

	mode := r.protocolMode()
	payload := r.buildPayload(mode, prompt)

	body, err := json.Marshal(payload)
	if err != nil {
		return ReviewDecision{}, fmt.Errorf("marshal review payload: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, r.cfg.APIURL, bytes.NewReader(body))
	if err != nil {
		return ReviewDecision{}, fmt.Errorf("create review request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+r.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	if mode == "anthropic" {
		httpReq.Header.Set("anthropic-version", anthropicVersion)
	}

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return ReviewDecision{}, fmt.Errorf("request review api: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReviewDecision{}, fmt.Errorf("read review response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReviewDecision{}, fmt.Errorf("review api status %d: %s", resp.StatusCode, safeCut(string(respBytes), 240))
	}

	rawText, err := extractResponseText(respBytes)
	if err != nil {
		return ReviewDecision{}, err
	}

	decision, err := parseDecision(rawText)
	if err != nil {
		r.logger.Warn("failed to parse ai review decision, fallback reject", "raw", safeCut(rawText, 120))
		return ReviewDecision{Allow: false, Reason: "parse_failed"}, nil
	}
	return decision, nil
}

func extractResponseText(raw []byte) (string, error) {
	var resp map[string]any
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("unmarshal review response: %w", err)
	}
	if text := tryAnthropicText(resp); text != "" {
		return text, nil
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
	if text := tryPathString(resp, "error", "message"); text != "" {
		return "", fmt.Errorf("review response error: %s", text)
	}
	return "", errors.New("review response has no content")
}

func parseDecision(text string) (ReviewDecision, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return ReviewDecision{}, errors.New("empty decision text")
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err == nil {
		allow, ok := obj["allow"].(bool)
		if ok {
			reason, _ := obj["reason"].(string)
			return ReviewDecision{Allow: allow, Reason: strings.TrimSpace(reason)}, nil
		}
	}

	lower := strings.ToLower(text)
	if strings.Contains(lower, "\"allow\":true") || strings.Contains(lower, "allow=true") ||
		strings.Contains(lower, "误判") || strings.Contains(lower, "可放行") {
		return ReviewDecision{Allow: true, Reason: "heuristic_allow"}, nil
	}
	return ReviewDecision{Allow: false, Reason: "heuristic_reject"}, nil
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

func safeCut(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) > max {
		return string(runes[:max])
	}
	return text
}

func sanitizeAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) >= 2 {
		if (strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"")) ||
			(strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'")) {
			key = key[1 : len(key)-1]
		}
	}
	return strings.TrimSpace(key)
}

func (r *AIReviewer) protocolMode() string {
	urlLower := strings.ToLower(r.cfg.APIURL)
	if strings.HasPrefix(r.cfg.APIKey, "sk-cp-") || strings.Contains(urlLower, "/anthropic/") {
		return "anthropic"
	}
	return "chatcompletion"
}

func (r *AIReviewer) buildPayload(mode, userPrompt string) map[string]any {
	systemPrompt := "你是评论审核助手。标准：宁可严谨，不放过明显违规。若仅是无害日常表达且命中词为歧义，可判定 allow=true。"
	if mode == "anthropic" {
		return map[string]any{
			"model":       r.cfg.Model,
			"system":      systemPrompt,
			"max_tokens":  120,
			"temperature": 0.0,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": userPrompt,
				},
			},
		}
	}
	return map[string]any{
		"model": r.cfg.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"temperature":        0.0,
		"max_tokens":         120,
		"tokens_to_generate": 120,
	}
}

func tryAnthropicText(resp map[string]any) string {
	content, ok := resp["content"]
	if !ok {
		return ""
	}
	parts := make([]string, 0)
	switch arr := content.(type) {
	case []any:
		for _, item := range arr {
			switch typed := item.(type) {
			case map[string]any:
				if text, ok := typed["text"].(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			case string:
				if strings.TrimSpace(typed) != "" {
					parts = append(parts, strings.TrimSpace(typed))
				}
			}
		}
	case []map[string]any:
		for _, item := range arr {
			if text, ok := item["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, strings.TrimSpace(text))
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
