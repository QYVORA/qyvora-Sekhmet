// Package scheduler implements seed scheduling and power scheduling. It
// decides which seed to execute next, weighting novelty, interestingness, size
// and historical yield so the hot path favors high-value seeds, and supports
// configurable scheduling strategies (fast/explore/exploit/rare/balanced/
// adaptive).
package scheduler

import (
	"sync"

	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Strategy selects the power-scheduling policy.
type Strategy string

const (
	StrategyFast     Strategy = "fast"
	StrategyExplore  Strategy = "explore"
	StrategyExploit  Strategy = "exploit"
	StrategyRare     Strategy = "rare"
	StrategyBalanced Strategy = "balanced"
	StrategyAdaptive Strategy = "adaptive"
)

// ParseStrategy normalizes a strategy name.
func ParseStrategy(s string) Strategy {
	switch Strategy(s) {
	case StrategyFast, StrategyExplore, StrategyExploit, StrategyRare,
		StrategyBalanced, StrategyAdaptive:
		return Strategy(s)
	default:
		return StrategyAdaptive
	}
}

// Scheduler picks the next seed from the corpus under a power policy. It is
// safe for concurrent use by multiple workers.
type Scheduler struct {
	mu       sync.Mutex
	strategy Strategy
	frame    int
}

// New returns a scheduler with the given strategy.
func New(strategy Strategy) *Scheduler {
	return &Scheduler{strategy: ParseStrategy(string(strategy))}
}

// Strategy returns the current strategy.
func (s *Scheduler) Strategy() Strategy { return s.strategy }

// Next returns the next seed to execute under the current policy. The corpus
// maintains a priority order; the scheduler maps the strategy onto selection
// so low-priority seeds are not starved under explore/rare policies.
func (s *Scheduler) Next(c *corpus.Store) *models.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := c.Prioritized()
	if len(list) == 0 {
		return nil
	}
	s.frame++
	switch s.strategy {
	case StrategyExplore, StrategyRare:
		if len(list) > 2 {
			idx := 1 + (s.frame*7)%(len(list)-1)
			return list[idx]
		}
	}
	return list[0]
}
