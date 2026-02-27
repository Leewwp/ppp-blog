package service

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
)

type WordStore struct {
	mu        sync.RWMutex
	words     map[string]struct{}
	automaton *AhoCorasick
	wordsFile string
	logger    *slog.Logger
}

func NewWordStore(wordsFile string, logger *slog.Logger) (*WordStore, error) {
	if logger == nil {
		logger = slog.Default()
	}

	store := &WordStore{
		words:     make(map[string]struct{}),
		automaton: NewAhoCorasick(nil),
		wordsFile: wordsFile,
		logger:    logger,
	}

	if err := store.loadFromFile(wordsFile); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *WordStore) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open words file %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	loaded := make(map[string]struct{})
	for scanner.Scan() {
		line := normalizeWord(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		loaded[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan words file %q: %w", path, err)
	}

	words := make([]string, 0, len(loaded))
	for w := range loaded {
		words = append(words, w)
	}
	sort.Strings(words)

	s.mu.Lock()
	s.words = loaded
	s.automaton.Build(words)
	s.mu.Unlock()

	s.logger.Info("sensitive words loaded", "count", len(words), "file", path)
	return nil
}

func (s *WordStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.words)
}

func (s *WordStore) ListWords() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	words := make([]string, 0, len(s.words))
	for w := range s.words {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func (s *WordStore) AddWord(word string) error {
	w := normalizeWord(word)
	if w == "" {
		return errors.New("word is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.words[w]; exists {
		return nil
	}
	s.words[w] = struct{}{}
	words := make([]string, 0, len(s.words))
	for ww := range s.words {
		words = append(words, ww)
	}

	sort.Strings(words)
	s.automaton.Build(words)
	s.logger.Info("sensitive word added", "word", w, "count", len(words))
	return nil
}

func (s *WordStore) RemoveWord(word string) bool {
	w := normalizeWord(word)
	if w == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.words[w]; !exists {
		return false
	}
	delete(s.words, w)
	words := make([]string, 0, len(s.words))
	for ww := range s.words {
		words = append(words, ww)
	}

	sort.Strings(words)
	s.automaton.Build(words)
	s.logger.Info("sensitive word removed", "word", w, "count", len(words))
	return true
}

func (s *WordStore) Match(content string) []Match {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.automaton.FindAll(content)
}

func (s *WordStore) WordsFile() string {
	return s.wordsFile
}
