// Package minimization reduces an interesting input to a smaller equivalent
// input that preserves the triggering behaviour. It is a first-class
// subsystem: minimization is queued (never run synchronously on the hot path)
// and is reversible against the target via a predicate.
package minimization

import (
	"context"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Predicate determines whether a given input still triggers the target
// behaviour of interest (crash, hang, anomaly, new coverage). The caller owns
// the comparison logic.
type Predicate func(res *models.ExecutionResult) bool

// Minimize reduces data using delta debugging (chunk removal), preserving the
// predicate. It returns the smallest input found, or the original if no
// reduction is possible. The work is bounded by context cancellation so it
// never starves the rest of the system.
func Minimize(ctx context.Context, runner execution.Runner, data []byte, timeout time.Duration, keep Predicate) []byte {
	if len(data) < 2 {
		return data
	}
	best := append([]byte(nil), data...)
	return deltaReduce(ctx, runner, best, timeout, keep)
}

func deltaReduce(ctx context.Context, runner execution.Runner, best []byte, timeout time.Duration,
	keep Predicate) []byte {

	// Phase 1: structural chunk removal (delta debugging over powers of two).
	candidate := append([]byte(nil), best...)
	n := len(candidate)
	granularity := maxInt(n/2, 1)
	for granularity >= 1 {
		select {
		case <-ctx.Done():
			return candidate
		default:
		}
		for start := 0; start < len(candidate); start += granularity {
			if start+granularity > len(candidate) {
				continue
			}
			reduced := make([]byte, 0, len(candidate)-granularity)
			reduced = append(reduced, candidate[:start]...)
			reduced = append(reduced, candidate[start+granularity:]...)
			if reducedEmpty(reduced) {
				continue
			}
			res, err := runner.Exec(ctx, reduced, timeout)
			if err != nil {
				continue
			}
			if keep(res) {
				candidate = reduced
				// Restart from chunk 0 at the same granularity after a
				// successful removal.
				start = -granularity
			}
		}
		granularity /= 2
	}

	// Phase 2: byte-level deletion pass over the surviving candidate.
	for i := 0; i < len(candidate); i++ {
		select {
		case <-ctx.Done():
			return candidate
		default:
		}
		reduced := make([]byte, 0, len(candidate)-1)
		reduced = append(reduced, candidate[:i]...)
		reduced = append(reduced, candidate[i+1:]...)
		if reducedEmpty(reduced) {
			continue
		}
		res, err := runner.Exec(ctx, reduced, timeout)
		if err != nil {
			continue
		}
		if keep(res) {
			candidate = reduced
			i--
		}
	}
	return candidate
}

func reducedEmpty(b []byte) bool {
	return len(b) == 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
