// Package safety implements the safety and resource-control layer for
// sekhmet. Fuzzing is inherently high-volume and can consume unbounded
// CPU/time/disk or accidentally target an unauthorized host. This package
// centralizes the guards that keep a campaign within its declared scope:
// authorization gating, dry-run, input size caps, execution/time/disk limits,
// and notification hooks so a campaign never silently exceeds its bounds.
package safety

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Limits is a set of bounds a campaign must respect.
type Limits struct {
	MaxInputBytes     int64                          // hard cap on a single input payload
	MaxExecutions     int64                          // 0 = unlimited
	MaxRuntime        time.Duration                  // 0 = unlimited
	MaxConcurrent     int                            // worker/goroutine cap
	MaxCorpusBytes    int64                          // corpus storage cap
	MaxOutputBytes    int64                          // per-execution captured output cap
	RequireAuthorized bool                           // refuse to start unless target is authorized
	AllowDryRun       bool                           // permit dry-run (no execution)
	NotifyOnExceed    func(kind string, value int64) // optional callback
}

// DefaultLimits returns conservative defaults sized for a development/CI box.
func DefaultLimits() Limits {
	return Limits{
		MaxInputBytes:     4 << 20, // 4 MiB
		MaxExecutions:     0,       // unlimited
		MaxRuntime:        0,       // unlimited
		MaxConcurrent:     4,
		MaxCorpusBytes:    512 << 20, // 512 MiB
		MaxOutputBytes:    4 << 20,   // 4 MiB
		RequireAuthorized: true,
		AllowDryRun:       true,
	}
}

// Guardian enforces limits safely across concurrent workers.
type Guardian struct {
	limits Limits

	executions atomic.Int64
	started    time.Time
	corpus     atomic.Int64

	mu       sync.Mutex
	exceeded map[string]bool
}

// NewGuardian returns a guardian bound to the given limits.
func NewGuardian(l Limits) *Guardian {
	return &Guardian{limits: l, exceeded: map[string]bool{}}
}

// Start records the campaign start time.
func (g *Guardian) Start() { g.started = time.Now() }

// CheckExec returns an error if the campaign may not proceed (limit hit or
// runtime expired). It is called before each execution.
func (g *Guardian) CheckExec() error {
	if g == nil {
		return nil
	}
	if g.limits.MaxExecutions > 0 && g.executions.Load() >= g.limits.MaxExecutions {
		return g.exceed("executions", g.executions.Load())
	}
	if g.limits.MaxRuntime > 0 && time.Since(g.started) >= g.limits.MaxRuntime {
		return g.exceed("runtime", int64(time.Since(g.started)))
	}
	return nil
}

// Executed records one execution.
func (g *Guardian) Executed() { g.executions.Add(1) }

// CheckInput returns an error if the input exceeds the size cap.
func (g *Guardian) CheckInput(n int) error {
	if g == nil || g.limits.MaxInputBytes <= 0 {
		return nil
	}
	if int64(n) > g.limits.MaxInputBytes {
		return g.exceed("input_bytes", int64(n))
	}
	return nil
}

// CheckAuthorized returns an error if authorization is required but not set.
func (g *Guardian) CheckAuthorized(authorized bool) error {
	if g == nil || !g.limits.RequireAuthorized {
		return nil
	}
	if !authorized {
		return fmt.Errorf("campaign requires an authorized target")
	}
	return nil
}

// AddCorpusBytes tracks corpus growth.
func (g *Guardian) AddCorpusBytes(n int) {
	if g == nil {
		return
	}
	cur := g.corpus.Add(int64(n))
	if g.limits.MaxCorpusBytes > 0 && cur > g.limits.MaxCorpusBytes {
		_ = g.exceed("corpus_bytes", cur)
	}
}

// Exceeded returns the kinds of limits that were exceeded.
func (g *Guardian) Exceeded() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]string, 0, len(g.exceeded))
	for k := range g.exceeded {
		out = append(out, k)
	}
	return out
}

func (g *Guardian) exceed(kind string, value int64) error {
	g.mu.Lock()
	if !g.exceeded[kind] {
		g.exceeded[kind] = true
		if g.limits.NotifyOnExceed != nil {
			g.limits.NotifyOnExceed(kind, value)
		}
	}
	g.mu.Unlock()
	return fmt.Errorf("safety limit exceeded: %s = %d", kind, value)
}
