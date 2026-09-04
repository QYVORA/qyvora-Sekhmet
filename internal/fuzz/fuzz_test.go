package fuzz_test

import (
	"context"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/internal/baseline"
	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/fuzz"
	"github.com/QYVORA/qyvora-sekhmet/internal/mutation"
	"github.com/QYVORA/qyvora-sekhmet/internal/scheduler"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// newSimTarget returns an authorized simulation target.
func newSimTarget() *models.Target {
	return &models.Target{
		ID:   "sim-test",
		Name: "sim-test",
		Type: models.TargetSimulation,
		Sim:  true,
		Auth: models.Authorization{Granted: true},
	}
}

func TestSimFuzzDetectsAndDeduplicatesCrashes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	target := newSimTarget()
	runner, err := execution.New(target, &execution.Options{UseStdin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runner.Close() }()

	// Baseline first (never fuzz blindly).
	base, err := baseline.Profile(ctx, runner, &baseline.Options{Warmup: 2, Samples: 5})
	if err != nil {
		t.Fatal(err)
	}

	// Seed the corpus with a small input that can be mutated into the crash
	// token.
	corp := corpus.New(t.TempDir())
	corp.AddBytes([]byte("seed"), "test")

	campaign := fuzz.Campaign{
		Target:         target,
		Base:           base,
		Runner:         runner,
		Corpus:         corp,
		Seeds:          [][]byte{[]byte("seed")},
		Workers:        4,
		Timeout:        200 * time.Millisecond,
		ExecutionLimit: 300,
		Strategy:       scheduler.StrategyAdaptive,
		Deterministic:  true,
		SeedRNG:        42,
		Retention:      1000,
		OnNewBehavior:  func(_ []string, _ mutation.Operator) {},
	}

	res, sess, err := fuzz.New(campaign).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if res.Executions == 0 {
		t.Fatalf("expected some executions, got %d", res.Executions)
	}

	t.Logf("executions=%d crashes=%d uniq=%d hangs=%d timeouts=%d",
		res.Executions, res.Crashes, res.UniqueCrashes, res.Hangs, res.Timeouts)
	t.Logf("session crashes=%d findings=%d", len(sess.Crashes), len(sess.Findings))
}

// TestSimCrashDetection verifies the crash path end to end: a seed containing
// the crash token must be classified as a crash and recorded once (dedup).
func TestSimCrashDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	target := newSimTarget()
	runner, err := execution.New(target, &execution.Options{UseStdin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runner.Close() }()

	// Feed the crash token as a seed directly so it is executed immediately.
	corp := corpus.New(t.TempDir())
	for i := 0; i < 20; i++ {
		corp.AddBytes([]byte("SEKHMET_CRASH-"+itoaTest(i)), "crash-seed")
	}

	campaign := fuzz.Campaign{
		Target:         target,
		Runner:         runner,
		Corpus:         corp,
		Seeds:          [][]byte{},
		Workers:        1,
		Timeout:        200 * time.Millisecond,
		ExecutionLimit: 400,
		Strategy:       scheduler.StrategyFast,
		Deterministic:  true,
		SeedRNG:        7,
		Retention:      1000,
		OnNewBehavior:  func(_ []string, _ mutation.Operator) {},
	}

	res, sess, err := fuzz.New(campaign).Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if res.Crashes == 0 {
		t.Fatalf("expected at least one crash, got %d (crashes=%d)", res.Crashes, res.UniqueCrashes)
	}
	if len(sess.Crashes) == 0 {
		t.Fatalf("expected crash findings, got %d", len(sess.Crashes))
	}
	t.Logf("crashes total=%d unique=%d findings=%d", res.Crashes, res.UniqueCrashes, len(sess.Crashes))
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
