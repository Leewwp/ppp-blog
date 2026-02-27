package service

import (
	"strings"
	"sync"
)

// Match describes one detected sensitive word in rune offsets.
// Start and End are inclusive rune indexes in the original content.
type Match struct {
	Word  string
	Start int
	End   int
}

type acNode struct {
	children map[rune]*acNode
	fail     *acNode
	outputs  []string
}

func newACNode() *acNode {
	return &acNode{
		children: make(map[rune]*acNode),
	}
}

// AhoCorasick implements a trie + failure-link automaton.
// The structure is rebuilt under a write lock to support hot reload.
// Matching runs under a read lock and is safe for concurrent requests.
type AhoCorasick struct {
	mu   sync.RWMutex
	root *acNode
}

func NewAhoCorasick(words []string) *AhoCorasick {
	ac := &AhoCorasick{
		root: newACNode(),
	}
	ac.Build(words)
	return ac
}

// Build reconstructs the full automaton from the provided words.
// Rebuilding from scratch keeps the logic simple and avoids stale links.
func (a *AhoCorasick) Build(words []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	root := newACNode()

	for _, word := range words {
		w := normalizeWord(word)
		if w == "" {
			continue
		}

		node := root
		for _, r := range []rune(w) {
			next, ok := node.children[r]
			if !ok {
				next = newACNode()
				node.children[r] = next
			}
			node = next
		}
		node.outputs = append(node.outputs, w)
	}

	// BFS to build failure links:
	// 1) root children fail to root
	// 2) for each edge (state, rune -> next), follow fail links from state
	//    until a state with same rune edge is found or root is reached
	queue := make([]*acNode, 0)
	for _, child := range root.children {
		child.fail = root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for r, next := range curr.children {
			f := curr.fail
			for f != nil && f != root {
				if candidate, ok := f.children[r]; ok {
					f = candidate
					goto found
				}
				f = f.fail
			}
			if f != nil {
				if candidate, ok := f.children[r]; ok {
					f = candidate
				} else {
					f = root
				}
			} else {
				f = root
			}

		found:
			next.fail = f
			if f != nil && len(f.outputs) > 0 {
				next.outputs = append(next.outputs, f.outputs...)
			}
			queue = append(queue, next)
		}
	}

	root.fail = root
	a.root = root
}

func normalizeWord(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

// FindAll returns all matches in text, including overlapping matches.
func (a *AhoCorasick) FindAll(text string) []Match {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.root == nil {
		return nil
	}

	contentRunes := []rune(strings.ToLower(text))
	if len(contentRunes) == 0 {
		return nil
	}

	matches := make([]Match, 0)
	state := a.root

	for idx, r := range contentRunes {
		for state != a.root {
			if _, ok := state.children[r]; ok {
				break
			}
			state = state.fail
		}

		if next, ok := state.children[r]; ok {
			state = next
		}

		if len(state.outputs) == 0 {
			continue
		}

		for _, word := range state.outputs {
			wordLen := len([]rune(word))
			start := idx - wordLen + 1
			if start < 0 {
				continue
			}
			matches = append(matches, Match{
				Word:  word,
				Start: start,
				End:   idx,
			})
		}
	}

	return matches
}
