package alerter

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ppp-blog/server-monitor/internal/collector"
	"gopkg.in/yaml.v3"
)

const (
	OperatorGT = "gt"
	OperatorLT = "lt"

	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

type Rule struct {
	Name            string  `json:"name" yaml:"name"`
	Metric          string  `json:"metric" yaml:"metric"`
	Operator        string  `json:"operator" yaml:"operator"`
	Threshold       float64 `json:"threshold" yaml:"threshold"`
	DurationMinutes int     `json:"duration_minutes" yaml:"duration_minutes"`
	Severity        string  `json:"severity" yaml:"severity"`
}

type Alert struct {
	RuleName    string    `json:"rule_name"`
	Metric      string    `json:"metric"`
	Operator    string    `json:"operator"`
	Threshold   float64   `json:"threshold"`
	Current     float64   `json:"current"`
	Severity    string    `json:"severity"`
	State       string    `json:"state"` // firing/resolved
	TriggeredAt time.Time `json:"triggered_at"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AlertEvent struct {
	Type         string `json:"type"` // firing/resolved
	Alert        Alert  `json:"alert"`
	ShouldNotify bool   `json:"should_notify"`
}

type Engine struct {
	mu       sync.RWMutex
	rules    []Rule
	states   map[string]*ruleState
	cooldown time.Duration
	filePath string
	logger   *slog.Logger
}

type ruleState struct {
	conditionStart time.Time
	hasCondition   bool
	firing         bool
	notified       bool
	lastFiredAt    time.Time
	alert          Alert
}

type rulesFile struct {
	Rules []Rule `yaml:"rules"`
}

func NewEngine(filePath string, cooldown time.Duration, logger *slog.Logger) (*Engine, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}

	e := &Engine{
		rules:    make([]Rule, 0),
		states:   make(map[string]*ruleState),
		cooldown: cooldown,
		filePath: filePath,
		logger:   logger,
	}

	if err := e.loadOrCreateRules(); err != nil {
		return nil, err
	}
	return e, nil
}

func DefaultRules() []Rule {
	return []Rule{
		{
			Name:            "high_cpu_usage",
			Metric:          "cpu_usage",
			Operator:        OperatorGT,
			Threshold:       85,
			DurationMinutes: 2,
			Severity:        SeverityWarning,
		},
		{
			Name:            "high_memory_usage",
			Metric:          "memory_usage",
			Operator:        OperatorGT,
			Threshold:       90,
			DurationMinutes: 0,
			Severity:        SeverityCritical,
		},
		{
			Name:            "high_disk_usage",
			Metric:          "disk_usage",
			Operator:        OperatorGT,
			Threshold:       80,
			DurationMinutes: 0,
			Severity:        SeverityWarning,
		},
	}
}

func (e *Engine) loadOrCreateRules() error {
	raw, err := os.ReadFile(e.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			e.rules = normalizeRules(DefaultRules())
			e.rebuildStateLocked()
			return e.saveRulesLocked()
		}
		return fmt.Errorf("read alert rules file: %w", err)
	}

	loaded, err := parseRulesYAML(raw)
	if err != nil {
		return fmt.Errorf("parse alert rules: %w", err)
	}
	loaded = normalizeRules(loaded)
	if err := validateRules(loaded); err != nil {
		return err
	}

	e.mu.Lock()
	e.rules = loaded
	e.rebuildStateLocked()
	e.mu.Unlock()
	e.logger.Info("alert rules loaded", "count", len(loaded), "file", e.filePath)
	return nil
}

func parseRulesYAML(raw []byte) ([]Rule, error) {
	var wrapped rulesFile
	if err := yaml.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Rules) > 0 {
		return wrapped.Rules, nil
	}

	var direct []Rule
	if err := yaml.Unmarshal(raw, &direct); err != nil {
		return nil, err
	}
	return direct, nil
}

func (e *Engine) ListRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return copyRules(e.rules)
}

// UpsertRule creates or updates a rule by name and persists the full rule set.
func (e *Engine) UpsertRule(rule Rule) (Rule, error) {
	rule = normalizeRule(rule)
	if err := validateRule(rule); err != nil {
		return Rule{}, err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	updated := false
	for i := range e.rules {
		if e.rules[i].Name == rule.Name {
			e.rules[i] = rule
			updated = true
			break
		}
	}
	if !updated {
		e.rules = append(e.rules, rule)
	}
	sortRules(e.rules)
	e.rebuildStateLocked()
	if err := e.saveRulesLocked(); err != nil {
		return Rule{}, err
	}

	e.logger.Info("alert rule upserted", "name", rule.Name, "updated", updated)
	return rule, nil
}

// ReplaceRules atomically replaces all rules and persists them.
func (e *Engine) ReplaceRules(rules []Rule) error {
	rules = normalizeRules(rules)
	if err := validateRules(rules); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = rules
	sortRules(e.rules)
	e.rebuildStateLocked()
	if err := e.saveRulesLocked(); err != nil {
		return err
	}
	e.logger.Info("alert rules replaced", "count", len(rules))
	return nil
}

func (e *Engine) saveRulesLocked() error {
	if err := os.MkdirAll(filepath.Dir(e.filePath), 0o755); err != nil {
		return fmt.Errorf("create rules directory: %w", err)
	}

	data, err := yaml.Marshal(rulesFile{Rules: e.rules})
	if err != nil {
		return fmt.Errorf("marshal rules yaml: %w", err)
	}

	tmp := e.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp rules file: %w", err)
	}
	if err := os.Rename(tmp, e.filePath); err != nil {
		return fmt.Errorf("rename rules file: %w", err)
	}
	return nil
}

func (e *Engine) rebuildStateLocked() {
	next := make(map[string]*ruleState, len(e.rules))
	for _, rule := range e.rules {
		if state, ok := e.states[rule.Name]; ok {
			next[rule.Name] = state
		} else {
			next[rule.Name] = &ruleState{}
		}
	}
	e.states = next
}

// Evaluate updates firing/resolved states from one snapshot.
// It returns state transition events; firing events can be suppressed by cooldown.
func (e *Engine) Evaluate(snapshot collector.SystemSnapshot) []AlertEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := snapshot.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	events := make([]AlertEvent, 0)

	for _, rule := range e.rules {
		state := e.states[rule.Name]
		if state == nil {
			state = &ruleState{}
			e.states[rule.Name] = state
		}

		value, ok := metricValue(rule.Metric, snapshot)
		if !ok {
			continue
		}

		conditionMet := compare(value, rule.Operator, rule.Threshold)
		durationNeeded := time.Duration(maxInt(0, rule.DurationMinutes)) * time.Minute

		if conditionMet {
			if !state.hasCondition {
				state.hasCondition = true
				state.conditionStart = now
			}

			if !state.firing && now.Sub(state.conditionStart) >= durationNeeded {
				state.firing = true
				state.notified = false
				state.alert = Alert{
					RuleName:    rule.Name,
					Metric:      rule.Metric,
					Operator:    rule.Operator,
					Threshold:   rule.Threshold,
					Current:     value,
					Severity:    rule.Severity,
					State:       "firing",
					TriggeredAt: now,
					UpdatedAt:   now,
				}
			}

			if state.firing {
				state.alert.Current = value
				state.alert.UpdatedAt = now

				if !state.notified && (state.lastFiredAt.IsZero() || now.Sub(state.lastFiredAt) >= e.cooldown) {
					state.notified = true
					state.lastFiredAt = now
					events = append(events, AlertEvent{
						Type:         "firing",
						Alert:        state.alert,
						ShouldNotify: true,
					})
				}
			}
			continue
		}

		// Condition not met.
		state.hasCondition = false
		if state.firing {
			resolved := state.alert
			resolved.State = "resolved"
			resolved.Current = value
			resolved.ResolvedAt = now
			resolved.UpdatedAt = now

			state.firing = false
			state.notified = false
			state.alert = Alert{}

			events = append(events, AlertEvent{
				Type:         "resolved",
				Alert:        resolved,
				ShouldNotify: true,
			})
		}
	}

	return events
}

func (e *Engine) CurrentFiringAlerts() []Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alerts := make([]Alert, 0)
	for _, rule := range e.rules {
		state := e.states[rule.Name]
		if state == nil || !state.firing {
			continue
		}
		alerts = append(alerts, state.alert)
	}

	sort.SliceStable(alerts, func(i, j int) bool {
		if alerts[i].Severity == alerts[j].Severity {
			return alerts[i].TriggeredAt.Before(alerts[j].TriggeredAt)
		}
		return severityRank(alerts[i].Severity) > severityRank(alerts[j].Severity)
	})
	return alerts
}

func severityRank(s string) int {
	switch s {
	case SeverityCritical:
		return 2
	case SeverityWarning:
		return 1
	default:
		return 0
	}
}

func metricValue(metric string, snapshot collector.SystemSnapshot) (float64, bool) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "cpu_usage", "cpu_usage_percent":
		return snapshot.CPUUsagePercent, true
	case "memory_usage", "memory_usage_percent":
		return snapshot.MemoryUsagePercent, true
	case "memory_used":
		return float64(snapshot.MemoryUsedBytes), true
	case "memory_total":
		return float64(snapshot.MemoryTotalBytes), true
	case "disk_usage", "disk_usage_percent":
		return snapshot.DiskUsagePercent, true
	case "disk_used":
		return float64(snapshot.DiskUsedBytes), true
	case "disk_total":
		return float64(snapshot.DiskTotalBytes), true
	case "network_bytes_sent_per_sec", "network_sent":
		return snapshot.NetworkBytesSentPerSec, true
	case "network_bytes_recv_per_sec", "network_recv":
		return snapshot.NetworkBytesRecvPerSec, true
	case "load_1", "load1":
		return snapshot.Load1, true
	case "load_5", "load5":
		return snapshot.Load5, true
	case "load_15", "load15":
		return snapshot.Load15, true
	case "process_count":
		return float64(snapshot.ProcessCount), true
	default:
		return 0, false
	}
}

func compare(value float64, operator string, threshold float64) bool {
	switch strings.ToLower(strings.TrimSpace(operator)) {
	case OperatorGT:
		return value > threshold
	case OperatorLT:
		return value < threshold
	default:
		return false
	}
}

func normalizeRules(rules []Rule) []Rule {
	out := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		out = append(out, normalizeRule(rule))
	}
	sortRules(out)
	return out
}

func normalizeRule(rule Rule) Rule {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Metric = strings.TrimSpace(strings.ToLower(rule.Metric))
	rule.Operator = strings.TrimSpace(strings.ToLower(rule.Operator))
	rule.Severity = strings.TrimSpace(strings.ToLower(rule.Severity))
	if rule.DurationMinutes < 0 {
		rule.DurationMinutes = 0
	}
	return rule
}

func validateRules(rules []Rule) error {
	if len(rules) == 0 {
		return errors.New("at least one alert rule is required")
	}
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if err := validateRule(rule); err != nil {
			return fmt.Errorf("invalid rule %q: %w", rule.Name, err)
		}
		if _, ok := seen[rule.Name]; ok {
			return fmt.Errorf("duplicate rule name %q", rule.Name)
		}
		seen[rule.Name] = struct{}{}
	}
	return nil
}

func validateRule(rule Rule) error {
	if rule.Name == "" {
		return errors.New("name is required")
	}
	if rule.Metric == "" {
		return errors.New("metric is required")
	}
	if rule.Operator != OperatorGT && rule.Operator != OperatorLT {
		return fmt.Errorf("operator must be %q or %q", OperatorGT, OperatorLT)
	}
	if rule.Severity != SeverityWarning && rule.Severity != SeverityCritical {
		return fmt.Errorf("severity must be %q or %q", SeverityWarning, SeverityCritical)
	}
	if rule.DurationMinutes < 0 {
		return errors.New("duration_minutes must be >= 0")
	}
	return nil
}

func sortRules(rules []Rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if severityRank(rules[i].Severity) == severityRank(rules[j].Severity) {
			if rules[i].DurationMinutes == rules[j].DurationMinutes {
				return rules[i].Name < rules[j].Name
			}
			return rules[i].DurationMinutes < rules[j].DurationMinutes
		}
		return severityRank(rules[i].Severity) > severityRank(rules[j].Severity)
	})
}

func copyRules(src []Rule) []Rule {
	out := make([]Rule, len(src))
	copy(out, src)
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
