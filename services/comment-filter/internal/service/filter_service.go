package service

import "log/slog"

type FilterResult struct {
	Passed          bool     `json:"passed"`
	FilteredContent string   `json:"filtered_content"`
	HitWords        []string `json:"hit_words"`
	AIReviewed      bool     `json:"ai_reviewed,omitempty"`
	AIAllowed       bool     `json:"ai_allowed,omitempty"`
	AIReason        string   `json:"ai_reason,omitempty"`
}

type FilterService struct {
	store    *WordStore
	reviewer *AIReviewer
	logger   *slog.Logger
	onHit    func()
}

func NewFilterService(store *WordStore, reviewer *AIReviewer, logger *slog.Logger, onHit func()) *FilterService {
	if logger == nil {
		logger = slog.Default()
	}

	return &FilterService{
		store:    store,
		reviewer: reviewer,
		logger:   logger,
		onHit:    onHit,
	}
}

func (s *FilterService) Filter(content, author string) FilterResult {
	matches := s.store.Match(content)
	if len(matches) == 0 {
		return FilterResult{
			Passed:          true,
			FilteredContent: content,
			HitWords:        []string{},
		}
	}

	hitSet := make(map[string]struct{})
	hitWords := make([]string, 0)
	for _, m := range matches {
		if _, ok := hitSet[m.Word]; ok {
			continue
		}
		hitSet[m.Word] = struct{}{}
		hitWords = append(hitWords, m.Word)
	}

	filtered := maskByMatches(content, matches)
	if s.onHit != nil {
		s.onHit()
	}
	s.logger.Warn("sensitive words detected", "author", author, "hits", hitWords, "hit_count", len(hitWords))

	if s.reviewer != nil && s.reviewer.Enabled() {
		decision, err := s.reviewer.Review(content, author, hitWords)
		if err != nil {
			s.logger.Warn("ai review failed, keep reject for safety", "author", author, "error", err)
		} else if decision.Allow {
			s.logger.Info("ai review allowed comment", "author", author, "hits", hitWords, "reason", decision.Reason)
			return FilterResult{
				Passed:          true,
				FilteredContent: content,
				HitWords:        hitWords,
				AIReviewed:      true,
				AIAllowed:       true,
				AIReason:        decision.Reason,
			}
		} else {
			return FilterResult{
				Passed:          false,
				FilteredContent: filtered,
				HitWords:        hitWords,
				AIReviewed:      true,
				AIAllowed:       false,
				AIReason:        decision.Reason,
			}
		}
	}

	return FilterResult{
		Passed:          false,
		FilteredContent: filtered,
		HitWords:        hitWords,
	}
}

func maskByMatches(content string, matches []Match) string {
	runes := []rune(content)
	mask := make([]bool, len(runes))

	for _, m := range matches {
		start := m.Start
		end := m.End
		if start < 0 {
			start = 0
		}
		if end >= len(runes) {
			end = len(runes) - 1
		}
		for i := start; i <= end && i < len(mask); i++ {
			mask[i] = true
		}
	}

	for i := range runes {
		if mask[i] {
			runes[i] = '*'
		}
	}
	return string(runes)
}
