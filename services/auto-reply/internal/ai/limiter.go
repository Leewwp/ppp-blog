package ai

import (
	"log/slog"
	"strings"
	"sync"
	"time"
)

type QuotaLimiterConfig struct {
	DailyGlobalLimit int
	DailyPerAuthor   int
	AuthorCooldown   time.Duration
	MaxCommentChars  int
}

type QuotaLimiter struct {
	cfg      QuotaLimiterConfig
	location *time.Location
	logger   *slog.Logger

	mu            sync.Mutex
	dayKey        string
	globalCount   int
	authorCount   map[string]int
	authorLastReq map[string]time.Time
	quotaLocked   bool
}

func NewQuotaLimiter(cfg QuotaLimiterConfig, logger *slog.Logger) *QuotaLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return &QuotaLimiter{
		cfg:           cfg,
		location:      loc,
		logger:        logger,
		authorCount:   make(map[string]int),
		authorLastReq: make(map[string]time.Time),
	}
}

func (l *QuotaLimiter) Allow(author, content string) (bool, string) {
	content = strings.TrimSpace(content)
	if l.cfg.MaxCommentChars > 0 && len([]rune(content)) > l.cfg.MaxCommentChars {
		return false, "comment_too_long"
	}

	now := time.Now().In(l.location)
	dayKey := now.Format("2006-01-02")

	l.mu.Lock()
	defer l.mu.Unlock()

	l.resetIfNeeded(dayKey)
	if l.quotaLocked {
		return false, "quota_exhausted"
	}

	authorKey := normalizeAuthor(author)
	if l.cfg.AuthorCooldown > 0 {
		if t, ok := l.authorLastReq[authorKey]; ok && now.Sub(t) < l.cfg.AuthorCooldown {
			return false, "author_cooldown"
		}
	}

	if l.cfg.DailyGlobalLimit > 0 && l.globalCount >= l.cfg.DailyGlobalLimit {
		return false, "daily_global_limit"
	}
	if l.cfg.DailyPerAuthor > 0 && l.authorCount[authorKey] >= l.cfg.DailyPerAuthor {
		return false, "daily_author_limit"
	}

	l.globalCount++
	l.authorCount[authorKey]++
	l.authorLastReq[authorKey] = now
	return true, ""
}

func (l *QuotaLimiter) MarkQuotaExhausted() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quotaLocked = true
	l.logger.Warn("auto-reply ai quota marked exhausted for current day")
}

func (l *QuotaLimiter) resetIfNeeded(dayKey string) {
	if l.dayKey == dayKey {
		return
	}
	l.dayKey = dayKey
	l.globalCount = 0
	l.authorCount = make(map[string]int)
	l.authorLastReq = make(map[string]time.Time)
	l.quotaLocked = false
}

func normalizeAuthor(author string) string {
	author = strings.TrimSpace(strings.ToLower(author))
	if author == "" {
		return "anonymous"
	}
	return author
}
