// Package baseline profiles a target's normal behaviour before fuzzing. This
// is the architectural core of "do not fuzz blindly": it measures timing,
// response and exit profiles over warm-up + sampled executions, and produces a
// structured Baseline against which live executions are classified.
package baseline

import (
	"context"
	"sort"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// State collects raw samples for statistical reduction.
type State struct {
	values []float64
}

func (s *State) add(v float64) { s.values = append(s.values, v) }

// Stats reduces raw samples to the descriptive Stat set.
func (s *State) Stats() models.Stat {
	n := len(s.values)
	stat := models.Stat{Count: n}
	if n == 0 {
		return stat
	}
	sorted := append([]float64(nil), s.values...)
	sort.Float64s(sorted)
	stat.Min = sorted[0]
	stat.Max = sorted[n-1]
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	stat.Mean = sum / float64(n)
	stat.Median = percentile(sorted, 50)
	stat.P90 = percentile(sorted, 90)
	stat.P95 = percentile(sorted, 95)
	stat.P99 = percentile(sorted, 99)
	var vs float64
	for _, v := range sorted {
		d := v - stat.Mean
		vs += d * d
	}
	if n > 1 {
		stat.Variance = vs / float64(n-1)
	}
	return stat
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	return sorted[idx]
}

// Options configures profiling.
type Options struct {
	Warmup  int
	Samples int
	Seed    []byte // deterministic seed input for baseline executions
}

func (o *Options) withDefaults() *Options {
	if o == nil {
		o = &Options{}
	}
	if o.Warmup <= 0 {
		o.Warmup = 5
	}
	if o.Samples <= 0 {
		o.Samples = 30
	}
	return o
}

// Profile runs warm-up then sampled executions against runner and returns a
// structured Baseline describing normal target behaviour. It never mutates the
// target; it only measures it.
func Profile(ctx context.Context, runner execution.Runner, opts *Options) (*models.Baseline, error) {
	opts = opts.withDefaults()
	base := &models.Baseline{
		Coverage:      models.CoverageProfile{Kind: "behavioral", TrackingEnabled: true},
		DynamicFields: []string{},
		Behaviors:     []models.BehaviorSig{},
	}

	// Warm-up: let the target reach steady state before sampling.
	for i := 0; i < opts.Warmup; i++ {
		if _, err := runner.Exec(ctx, opts.Seed, 5*time.Second); err != nil {
			return base, err
		}
	}

	timing := &State{}
	stdoutSize := &State{}
	stderrSize := &State{}
	exitSet := map[int]struct{}{}
	sigSet := map[string]struct{}{}
	statusSet := map[int]struct{}{}

	for i := 0; i < opts.Samples; i++ {
		res, err := runner.Exec(ctx, opts.Seed, 5*time.Second)
		if err != nil {
			return base, err
		}
		if res.Failed {
			return base, &ProfileLaunchError{err: res.Error}
		}
		timing.add(float64(res.Duration) / float64(time.Millisecond))
		stdoutSize.add(float64(res.StdoutSize))
		stderrSize.add(float64(res.StderrSize))
		exitSet[res.ExitCode] = struct{}{}
		if res.Signal != "" {
			sigSet[res.Signal] = struct{}{}
		}
		if res.StatusCode != 0 {
			statusSet[res.StatusCode] = struct{}{}
		}
	}

	ts := timing.Stats()
	base.Timing = &models.TimingProfile{
		Normal:    ts,
		SlowFloor: ts.P95,
		Timeout:   ts.P95*1.5 + 50, // adaptive: p95 plus margin, ms
		Warmup:    opts.Warmup,
		Samples:   opts.Samples,
	}

	base.ExitProfile = &models.ResponseProfile{
		StdoutSize: stdoutSize.Stats(),
		StderrSize: stderrSize.Stats(),
		ExitCodes:  keys(exitSet),
		Signals:    sortedKeys(sigSet),
		StatusCode: keys(statusSet),
		BodySizes:  models.Stat{},
	}

	// Record the baseline's normal behavior signature so the live engine has
	// a concrete "normal" reference.
	base.Behaviors = append(base.Behaviors, models.BehaviorSig{
		Name:   "baseline.normal",
		Label:  "nominal execution",
		Normal: true,
		Count:  opts.Samples,
		Fields: map[string]string{"exit_code_total": itoa(len(base.ExitProfile.ExitCodes))},
	})

	return base, nil
}

// ProfileLaunchError signals that baseline profiling could not launch the
// target at all, distinct from a normal target failure.
type ProfileLaunchError struct{ err string }

func (e *ProfileLaunchError) Error() string { return "target launch failed during baseline: " + e.err }

func keys(m map[int]struct{}) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func itoa(n int) string {
	return time.Duration(n).String()
}
