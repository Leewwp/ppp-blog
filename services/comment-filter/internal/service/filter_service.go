package service

import "log/slog"

type FilterResult struct {
	Passed          bool     `json:"passed"`
	FilteredContent string   `json:"filtered_content"`
	HitWords        []string `json:"hit_words"`
}

type FilterService struct {
	store  *WordStore
	logger *slog.Logger
	onHit  func()
}

func NewFilterService(store *WordStore, logger *slog.Logger, onHit func()) *FilterService {
	if logger == nil {
		logger = slog.Default()
	}

	return &FilterService{
		store:  store,
		logger: logger,
		onHit:  onHit,
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
