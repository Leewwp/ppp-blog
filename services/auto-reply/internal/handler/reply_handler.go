package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/auto-reply/internal/ai"
	"github.com/ppp-blog/auto-reply/internal/engine"
	"github.com/ppp-blog/auto-reply/internal/store"
)

type ReplyHandler struct {
	ruleStore  *store.RuleStore
	ruleEngine *engine.RuleEngine
	aiClient   *ai.Client
	limiter    *ai.QuotaLimiter
	logger     *slog.Logger
	onDecision func(shouldReply bool, ruleID, ruleName string)
}

type replyRequest struct {
	CommentID string `json:"comment_id"`
	Content   string `json:"content"`
	PostTitle string `json:"post_title"`
	Author    string `json:"author"`
}

type replyResponse struct {
	ShouldReply  bool   `json:"should_reply"`
	ReplyContent string `json:"reply_content"`
	DelaySeconds int    `json:"delay_seconds"`
	MatchedRule  string `json:"matched_rule"`
}

func NewReplyHandler(
	ruleStore *store.RuleStore,
	ruleEngine *engine.RuleEngine,
	aiClient *ai.Client,
	limiter *ai.QuotaLimiter,
	logger *slog.Logger,
	onDecision func(shouldReply bool, ruleID, ruleName string),
) *ReplyHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReplyHandler{
		ruleStore:  ruleStore,
		ruleEngine: ruleEngine,
		aiClient:   aiClient,
		limiter:    limiter,
		logger:     logger,
		onDecision: onDecision,
	}
}

func (h *ReplyHandler) Reply(c *gin.Context) {
	var req replyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid reply request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	rules := h.ruleStore.ListRules()
	decision, matchedRule, err := h.ruleEngine.Evaluate(rules, engine.CommentContext{
		CommentID: req.CommentID,
		Content:   req.Content,
		PostTitle: strings.TrimSpace(req.PostTitle),
		Author:    strings.TrimSpace(req.Author),
	})
	if err != nil {
		h.logger.Error("rule evaluation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to evaluate reply rules"})
		return
	}

	var matchedRuleID, matchedRuleName string
	if matchedRule != nil {
		matchedRuleID = matchedRule.ID
		matchedRuleName = matchedRule.Name
	}

	finalDecision := decision
	if finalDecision.ShouldReply {
		if h.limiter != nil {
			allowed, reason := h.limiter.Allow(req.Author, req.Content)
			if !allowed {
				h.logger.Info("skip auto-reply by limiter", "reason", reason, "author", req.Author)
				finalDecision = engine.ReplyDecision{}
			}
		}
	}

	if finalDecision.ShouldReply {
		if h.aiClient == nil || !h.aiClient.Enabled() {
			h.logger.Warn("skip auto-reply: ai client is disabled")
			finalDecision = engine.ReplyDecision{}
		} else {
			generated, genErr := h.aiClient.GenerateReply(c.Request.Context(), ai.GenerateRequest{
				Author:         req.Author,
				PostTitle:      req.PostTitle,
				CommentContent: req.Content,
				RuleHint:       finalDecision.ReplyContent,
			})
			if genErr != nil {
				if errors.Is(genErr, ai.ErrQuotaExceeded) && h.limiter != nil {
					h.limiter.MarkQuotaExhausted()
				}
				h.logger.Warn("skip auto-reply: ai generation failed", "error", genErr)
				finalDecision = engine.ReplyDecision{}
			} else {
				finalDecision.ReplyContent = generated
			}
		}
	}

	if h.onDecision != nil {
		h.onDecision(finalDecision.ShouldReply, matchedRuleID, matchedRuleName)
	}

	c.JSON(http.StatusOK, replyResponse{
		ShouldReply:  finalDecision.ShouldReply,
		ReplyContent: finalDecision.ReplyContent,
		DelaySeconds: finalDecision.DelaySeconds,
		MatchedRule:  finalDecision.MatchedRule,
	})
}
