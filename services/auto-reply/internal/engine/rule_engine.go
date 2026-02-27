package engine

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	clockLayout            = "15:04"
	defaultMinDelaySeconds = 3
	defaultMaxDelaySeconds = 15
)

// TimeRange defines an active window in local clock format HH:MM.
// Empty start and end means always active.
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Rule is one reply rule in the engine.
// Priority is evaluated in descending order, and first match wins.
type Rule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Keywords  []string  `json:"keywords"`
	TimeRange TimeRange `json:"time_range"`
	Templates []string  `json:"templates"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
}

// CommentContext is the input used for rule matching and template rendering.
type CommentContext struct {
	CommentID string
	Content   string
	PostTitle string
	Author    string
}

// ReplyDecision is the engine output for one comment.
type ReplyDecision struct {
	ShouldReply  bool
	ReplyContent string
	DelaySeconds int
	MatchedRule  string
}

// RuleEngine encapsulates matching and rendering behavior.
// It keeps timezone and RNG state, making the engine easy to extend.
type RuleEngine struct {
	location *time.Location
	nowFn    func() time.Time
	rnd      *rand.Rand
	rndMu    sync.Mutex
}

func NewRuleEngine() (*RuleEngine, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, fmt.Errorf("load timezone Asia/Shanghai: %w", err)
	}

	return &RuleEngine{
		location: loc,
		nowFn:    time.Now,
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Evaluate checks rules by priority and returns the first matched rule result.
func (e *RuleEngine) Evaluate(rules []Rule, comment CommentContext) (ReplyDecision, *Rule, error) {
	ordered := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ordered = append(ordered, normalizeRule(rule))
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Priority == ordered[j].Priority {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Priority > ordered[j].Priority
	})

	now := e.nowFn().In(e.location)
	contentLower := strings.ToLower(comment.Content)

	for i := range ordered {
		rule := ordered[i]
		if !rule.Enabled {
			continue
		}

		inWindow, err := isInTimeWindow(now, rule.TimeRange)
		if err != nil {
			// Invalid time window is considered non-match so bad rules do not stop service.
			continue
		}
		if !inWindow {
			continue
		}
		if !keywordMatch(contentLower, rule.Keywords) {
			continue
		}

		tpl, err := e.pickTemplate(rule.Templates)
		if err != nil {
			return ReplyDecision{}, nil, fmt.Errorf("rule %s has invalid templates: %w", rule.ID, err)
		}
		rendered, err := renderTemplate(tpl, comment)
		if err != nil {
			return ReplyDecision{}, nil, fmt.Errorf("render template for rule %s: %w", rule.ID, err)
		}

		matched := rule
		return ReplyDecision{
			ShouldReply:  true,
			ReplyContent: rendered,
			DelaySeconds: e.randomDelaySeconds(),
			MatchedRule:  rule.Name,
		}, &matched, nil
	}

	return ReplyDecision{
		ShouldReply:  false,
		ReplyContent: "",
		DelaySeconds: 0,
		MatchedRule:  "",
	}, nil, nil
}

func normalizeRule(rule Rule) Rule {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Name = strings.TrimSpace(rule.Name)

	keywords := make([]string, 0, len(rule.Keywords))
	for _, kw := range rule.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		keywords = append(keywords, kw)
	}
	rule.Keywords = keywords

	templates := make([]string, 0, len(rule.Templates))
	for _, tpl := range rule.Templates {
		tpl = strings.TrimSpace(tpl)
		if tpl == "" {
			continue
		}
		templates = append(templates, tpl)
	}
	rule.Templates = templates
	rule.TimeRange.Start = strings.TrimSpace(rule.TimeRange.Start)
	rule.TimeRange.End = strings.TrimSpace(rule.TimeRange.End)
	return rule
}

// keywordMatch uses OR semantics:
// if any keyword is contained in content (case-insensitive), the rule matches.
// Empty keyword list acts as catch-all.
func keywordMatch(contentLower string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}

	hasValidKeyword := false
	for _, kw := range keywords {
		normalized := strings.ToLower(strings.TrimSpace(kw))
		if normalized == "" {
			continue
		}
		hasValidKeyword = true
		if strings.Contains(contentLower, normalized) {
			return true
		}
	}

	// If all keywords are empty, treat as catch-all for resilience.
	return !hasValidKeyword
}

// isInTimeWindow supports normal and cross-midnight windows.
// Example: 22:00-08:00 is true for [22:00,24:00) U [00:00,08:00).
func isInTimeWindow(now time.Time, window TimeRange) (bool, error) {
	start := strings.TrimSpace(window.Start)
	end := strings.TrimSpace(window.End)
	if start == "" && end == "" {
		return true, nil
	}
	if start == "" || end == "" {
		return false, errors.New("start and end must both be set")
	}

	startMinute, err := parseClockMinute(start)
	if err != nil {
		return false, err
	}
	endMinute, err := parseClockMinute(end)
	if err != nil {
		return false, err
	}

	nowMinute := now.Hour()*60 + now.Minute()
	if startMinute == endMinute {
		return true, nil
	}
	if startMinute < endMinute {
		return nowMinute >= startMinute && nowMinute < endMinute, nil
	}
	return nowMinute >= startMinute || nowMinute < endMinute, nil
}

func parseClockMinute(value string) (int, error) {
	t, err := time.Parse(clockLayout, value)
	if err != nil {
		return 0, fmt.Errorf("invalid time %q: %w", value, err)
	}
	return t.Hour()*60 + t.Minute(), nil
}

func renderTemplate(tpl string, comment CommentContext) (string, error) {
	t, err := template.New("reply").Option("missingkey=zero").Parse(tpl)
	if err != nil {
		return "", err
	}

	data := struct {
		Author    string
		PostTitle string
		Content   string
	}{
		Author:    comment.Author,
		PostTitle: comment.PostTitle,
		Content:   comment.Content,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func (e *RuleEngine) pickTemplate(templates []string) (string, error) {
	if len(templates) == 0 {
		return "", errors.New("templates are empty")
	}
	idx := e.randomInt(0, len(templates)-1)
	return templates[idx], nil
}

func (e *RuleEngine) randomDelaySeconds() int {
	return e.randomInt(defaultMinDelaySeconds, defaultMaxDelaySeconds)
}

func (e *RuleEngine) randomInt(min, max int) int {
	if max <= min {
		return min
	}
	e.rndMu.Lock()
	defer e.rndMu.Unlock()
	return min + e.rnd.Intn(max-min+1)
}
