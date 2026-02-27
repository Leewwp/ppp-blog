package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ppp-blog/auto-reply/internal/engine"
	"github.com/ppp-blog/auto-reply/internal/store"
)

type ReplyHandler struct {
	ruleStore  *store.RuleStore
	ruleEngine *engine.RuleEngine
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
	logger *slog.Logger,
	onDecision func(shouldReply bool, ruleID, ruleName string),
) *ReplyHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReplyHandler{
		ruleStore:  ruleStore,
		ruleEngine: ruleEngine,
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
	if h.onDecision != nil {
		h.onDecision(decision.ShouldReply, matchedRuleID, matchedRuleName)
	}

	c.JSON(http.StatusOK, replyResponse{
		ShouldReply:  decision.ShouldReply,
		ReplyContent: decision.ReplyContent,
		DelaySeconds: decision.DelaySeconds,
		MatchedRule:  decision.MatchedRule,
	})
}
