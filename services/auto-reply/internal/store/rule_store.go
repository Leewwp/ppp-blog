package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ppp-blog/auto-reply/internal/engine"
)

var ErrRuleNotFound = errors.New("rule not found")

type RuleStore struct {
	mu        sync.RWMutex
	rules     []engine.Rule
	rulesFile string
	logger    *slog.Logger
}

func NewRuleStore(rulesFile string, logger *slog.Logger) (*RuleStore, error) {
	if logger == nil {
		logger = slog.Default()
	}

	s := &RuleStore{
		rulesFile: rulesFile,
		logger:    logger,
	}

	if err := s.loadFromFile(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *RuleStore) loadFromFile() error {
	raw, err := os.ReadFile(s.rulesFile)
	if err != nil {
		return fmt.Errorf("read rules file %q: %w", s.rulesFile, err)
	}

	rules := make([]engine.Rule, 0)
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &rules); err != nil {
			return fmt.Errorf("unmarshal rules file %q: %w", s.rulesFile, err)
		}
	}

	normalized := make([]engine.Rule, 0, len(rules))
	for _, r := range rules {
		r = normalizeRule(r)
		if err := validateRule(r, false); err != nil {
			return fmt.Errorf("invalid rule %q in file: %w", r.ID, err)
		}
		normalized = append(normalized, r)
	}
	sortRules(normalized)

	s.mu.Lock()
	s.rules = normalized
	s.mu.Unlock()

	s.logger.Info("rules loaded", "count", len(normalized), "file", s.rulesFile)
	return nil
}

func (s *RuleStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules)
}

func (s *RuleStore) ListRules() []engine.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyRules(s.rules)
}

func (s *RuleStore) GetRule(id string) (engine.Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.rules {
		if r.ID == id {
			return r, true
		}
	}
	return engine.Rule{}, false
}

func (s *RuleStore) CreateRule(rule engine.Rule) (engine.Rule, error) {
	rule = normalizeRule(rule)
	if err := validateRule(rule, true); err != nil {
		return engine.Rule{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ID == "" {
		rule.ID = s.generateRuleIDLocked()
	}
	if s.hasRuleIDLocked(rule.ID) {
		return engine.Rule{}, fmt.Errorf("rule id %q already exists", rule.ID)
	}

	s.rules = append(s.rules, rule)
	sortRules(s.rules)

	if err := s.saveLocked(); err != nil {
		return engine.Rule{}, err
	}

	s.logger.Info("rule created", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

func (s *RuleStore) UpdateRule(id string, rule engine.Rule) (engine.Rule, error) {
	rule = normalizeRule(rule)
	rule.ID = strings.TrimSpace(id)
	if err := validateRule(rule, false); err != nil {
		return engine.Rule{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.rules {
		if s.rules[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return engine.Rule{}, ErrRuleNotFound
	}

	s.rules[idx] = rule
	sortRules(s.rules)

	if err := s.saveLocked(); err != nil {
		return engine.Rule{}, err
	}

	s.logger.Info("rule updated", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

func (s *RuleStore) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.rules {
		if s.rules[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrRuleNotFound
	}

	deleted := s.rules[idx]
	s.rules = append(s.rules[:idx], s.rules[idx+1:]...)

	if err := s.saveLocked(); err != nil {
		return err
	}

	s.logger.Info("rule deleted", "id", deleted.ID, "name", deleted.Name)
	return nil
}

func (s *RuleStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.rulesFile), 0o755); err != nil {
		return fmt.Errorf("create rules directory: %w", err)
	}

	data, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}
	data = append(data, '\n')

	tmpPath := s.rulesFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write tmp rules file: %w", err)
	}
	if err := os.Rename(tmpPath, s.rulesFile); err != nil {
		return fmt.Errorf("rename tmp rules file: %w", err)
	}
	return nil
}

func (s *RuleStore) hasRuleIDLocked(id string) bool {
	for _, r := range s.rules {
		if r.ID == id {
			return true
		}
	}
	return false
}

func (s *RuleStore) generateRuleIDLocked() string {
	for {
		id := fmt.Sprintf("rule-%d", time.Now().UnixNano())
		if !s.hasRuleIDLocked(id) {
			return id
		}
	}
}

func normalizeRule(rule engine.Rule) engine.Rule {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Name = strings.TrimSpace(rule.Name)
	rule.TimeRange.Start = strings.TrimSpace(rule.TimeRange.Start)
	rule.TimeRange.End = strings.TrimSpace(rule.TimeRange.End)

	keywordSet := make(map[string]struct{})
	keywords := make([]string, 0, len(rule.Keywords))
	for _, kw := range rule.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		lower := strings.ToLower(kw)
		if _, ok := keywordSet[lower]; ok {
			continue
		}
		keywordSet[lower] = struct{}{}
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
	return rule
}

func validateRule(rule engine.Rule, allowEmptyID bool) error {
	if !allowEmptyID && rule.ID == "" {
		return errors.New("rule id is required")
	}
	if rule.Name == "" {
		return errors.New("rule name is required")
	}
	if len(rule.Templates) == 0 {
		return errors.New("at least one template is required")
	}

	start := strings.TrimSpace(rule.TimeRange.Start)
	end := strings.TrimSpace(rule.TimeRange.End)
	if (start == "") != (end == "") {
		return errors.New("time_range start and end must both be set")
	}
	if start != "" {
		if _, err := time.Parse("15:04", start); err != nil {
			return fmt.Errorf("invalid time_range.start: %w", err)
		}
		if _, err := time.Parse("15:04", end); err != nil {
			return fmt.Errorf("invalid time_range.end: %w", err)
		}
	}

	return nil
}

func sortRules(rules []engine.Rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority > rules[j].Priority
	})
}

func copyRules(rules []engine.Rule) []engine.Rule {
	result := make([]engine.Rule, 0, len(rules))
	for _, r := range rules {
		clone := r
		clone.Keywords = append([]string(nil), r.Keywords...)
		clone.Templates = append([]string(nil), r.Templates...)
		result = append(result, clone)
	}
	return result
}
