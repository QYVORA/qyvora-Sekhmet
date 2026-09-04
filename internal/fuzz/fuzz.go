// Package fuzz implements the fuzzing campaign orchestrator. It ties together
// the target runner, corpus, mutation engine, feedback tracker, scheduler,
// and detection to run the adaptive hot path:
//
//	SELECT SEED → MUTATE → EXECUTE → CLASSIFY → FEEDBACK → QUEUE INTERESTING
//
// Expensive analysis (crash triage, minimization, evidence) is pushed to an
// asynchronous deep path so the hot path stays fast.
package fuzz

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	"github.com/QYVORA/qyvora-sekhmet/internal/detection"
	"github.com/QYVORA/qyvora-sekhmet/internal/events"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/feedback"
	"github.com/QYVORA/qyvora-sekhmet/internal/mutation"
	"github.com/QYVORA/qyvora-sekhmet/internal/safety"
	"github.com/QYVORA/qyvora-sekhmet/internal/scheduler"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Campaign is the configuration for one fuzz run.
type Campaign struct {
	Target         *models.Target
	Base           *models.Baseline
	Runner         execution.Runner
	Corpus         *corpus.Store
	Seeds          [][]byte // initial seeds
	Workers        int
	Timeout        time.Duration // adaptive (per-execution) timeout
	HardRuntime    time.Duration // 0 = until limit or interrupt
	ExecutionLimit int64         // 0 = unlimited
	Strategy       scheduler.Strategy
	Deterministic  bool
	SeedRNG        int64 // used when deterministic
	Emit           func(level, name string, data map[string]any)
	OnNewBehavior  func(features []string, mutation mutation.Operator)
	Retention      int // corpus retention cap
	Safety         *safety.Guardian
}

// Result aggregates campaign statistics.
type Result struct {
	Executions     int64         `json:"executions"`
	ExecPerSec     float64       `json:"exec_per_sec"`
	CorpusSize     int           `json:"corpus_size"`
	CoverageGained int           `json:"coverage_gained"`
	NewBehaviors   int           `json:"new_behaviors"`
	UniqueCrashes  int           `json:"unique_crashes"`
	Crashes        int           `json:"crashes_total"`
	Hangs          int           `json:"hangs"`
	Anomalies      int           `json:"anomalies"`
	Timeouts       int           `json:"timeouts"`
	Duration       time.Duration `json:"duration_ns"`
	Strategy       string        `json:"strategy"`
}

// Engine holds the campaign state and counters.
type Engine struct {
	cfg    Campaign
	engine *mutation.Engine
	sched  *scheduler.Scheduler
	track  *feedback.Tracker
	sess   *models.Session

	countExec     atomic.Int64
	countCrash    atomic.Int64
	countHang     atomic.Int64
	countTimeout  atomic.Int64
	countAnomaly  atomic.Int64
	countBehavior atomic.Int64
	countCoverage atomic.Int64

	crashMu  sync.Mutex
	crashSet map[string]bool

	resultMu sync.Mutex
	result   *Result

	start time.Time
}

// New builds an engine for the campaign.
func New(cfg Campaign) *Engine {
	kind := feedback.KindBehavioral
	if cfg.Base != nil {
		switch cfg.Base.Coverage.Kind {
		case "edges":
			kind = feedback.KindEdges
		case "blocks":
			kind = feedback.KindBlocks
		}
	}
	track := feedback.NewTracker(kind, nil)

	mutCfg := mutation.Config{Dictionary: nil}
	if cfg.Deterministic {
		mutCfg.RNG = newSeededRand(cfg.SeedRNG)
	}
	e := &Engine{
		cfg:      cfg,
		engine:   mutation.New(mutCfg),
		sched:    scheduler.New(cfg.Strategy),
		track:    track,
		crashSet: make(map[string]bool),
		result:   &Result{Strategy: string(cfg.Strategy)},
	}
	if cfg.Retention <= 0 {
		cfg.Retention = 100000
	}
	e.cfg = cfg
	return e
}

// Run executes the campaign until the limit is hit or the context is
// cancelled. It returns a Result and the populated session (findings/crashes).
func (e *Engine) Run(ctx context.Context) (*Result, *models.Session, error) {
	// Seed the corpus from initial seeds.
	if e.cfg.Seeds != nil {
		for _, s := range e.cfg.Seeds {
			seed, added := e.cfg.Corpus.AddBytes(s, "seed")
			if added && e.cfg.Safety != nil {
				e.cfg.Safety.AddCorpusBytes(seed.Size)
			}
		}
	}

	e.start = time.Now()
	e.sess = &models.Session{
		ID:       models.NewSessionID(),
		TargetID: targetID(e.cfg.Target),
		Start:    e.start,
		State:    models.SessionRunning,
	}

	// Safety gate: the campaign must be authorized and within declared limits
	// before starting.
	if e.cfg.Safety != nil {
		auth := e.cfg.Target != nil && (e.cfg.Target.Sim || e.cfg.Target.Authorized())
		if err := e.cfg.Safety.CheckAuthorized(auth); err != nil {
			return nil, nil, err
		}
		e.cfg.Safety.Start()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < e.cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.worker(ctx)
		}()
	}
	// Monitor hard runtime limit.
	if e.cfg.HardRuntime > 0 {
		go func() {
			select {
			case <-time.After(e.cfg.HardRuntime):
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	// Monitor loop finishes workers when context is done.
	wg.Wait()

	e.processDeepAnalysis(ctx)

	res := e.finalize()
	return res, e.sess, nil
}

func (e *Engine) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if e.limitReached() {
			return
		}
		if e.cfg.Safety != nil {
			if err := e.cfg.Safety.CheckExec(); err != nil {
				return
			}
		}
		seed := e.sched.Next(e.cfg.Corpus)
		if seed == nil {
			// No corpus input; generate a trivial seed.
			e.cfg.Corpus.AddBytes([]byte{'A'}, "generated")
			continue
		}
		e.processSeed(ctx, seed)
	}
}

func (e *Engine) processSeed(ctx context.Context, seed *models.Seed) {
	for i := 0; i < 128; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if e.limitReached() {
			return
		}
		op, mutated := e.engine.Mutate(seed.Data)
		if e.cfg.Safety != nil {
			if err := e.cfg.Safety.CheckInput(len(mutated)); err != nil {
				return
			}
		}
		timeout := timeoutFor(e.cfg.Timeout)
		res, err := e.cfg.Runner.Exec(ctx, mutated, timeout)
		if err != nil {
			return
		}
		e.countExec.Add(1)
		if e.cfg.Safety != nil {
			e.cfg.Safety.Executed()
		}

		class := detection.Classify(res, e.cfg.Base)

		// Feedback / novelty.
		feats := feedback.BehaviorSignature(res)
		novel := e.track.Observe(feats)
		if novel > 0 {
			n := e.countBehavior.Add(int64(novel))
			_ = n
			e.countCoverage.Add(int64(novel))
			e.cfg.Corpus.AddBytes(mutated, "behavior")
			e.cfg.OnNewBehavior(feats, op)
			e.engine.Record(op, true, true, true, false, false, false)
			e.emit6(true, true, true)
			continue
		}

		// Classification of anomalies.
		switch class {
		case models.ClassCrash:
			e.countCrash.Add(1)
			e.handleCrash(ctx, seed, op, mutated, res)
			e.emit6(true, false, true)
		case models.ClassHang:
			e.countHang.Add(1)
			e.countTimeout.Add(1)
			e.cfg.Corpus.AddBytes(mutated, "hang")
			e.emit6(false, false, true)
		case models.ClassUnexpected, models.ClassNormalFailure:
			// Might be expected-from-baseline; only flag anomaly if it
			// deviates from baseline behavior.
			if e.isAnomaly(res) {
				e.countAnomaly.Add(1)
				e.cfg.Corpus.AddBytes(mutated, "anomaly")
				e.emit6(false, false, true)
			}
		}
		e.cfg.Corpus.MarkExecuted(seed.ID, novel, novel, 0)
	}
}

func (e *Engine) handleCrash(_ context.Context, seed *models.Seed, op mutation.Operator, mutated []byte, res *models.ExecutionResult) {
	sig := detection.Signature(res, &models.ExecutionDetail{
		ExitCode: res.ExitCode, Signal: res.Signal, Stderr: res.Stderr,
	})
	e.crashMu.Lock()
	newCrash := !e.crashSet[sig]
	e.crashSet[sig] = true
	e.crashMu.Unlock()

	if !newCrash {
		return
	}

	f := e.buildCrashFinding(sig, op, seed, mutated, res)
	e.sess.AddCrash(f)
	e.emitCrash(f)
}

func (e *Engine) buildCrashFinding(sig string, op mutation.Operator, seed *models.Seed, mutated []byte, res *models.ExecutionResult) *models.Finding {
	f := &models.Finding{
		Title:          "target crashed during fuzzing",
		Category:       "fuzzing",
		Classification: models.ClassificationCrashDetected,
		Description:    "A fuzzed input caused the target to terminate with a crash signal during the campaign.",
		Severity:       models.SeverityHigh,
		Confidence:     models.ConfidenceHigh,
		Status:         models.StatusValidated,
		RuleID:         "FUZ-001",
		CrashSign:      sig,
		Timestamp:      time.Now().UTC(),
	}
	if e.cfg.Target != nil {
		f.TargetID = e.cfg.Target.ID
	}
	f.ID = shortFingerprint(f.Fingerprint())
	if seed != nil {
		f.InputRef = seed.ID
	}
	if f.Attributes == nil {
		f.Attributes = map[string]string{}
	}
	f.Attributes["signal"] = res.Signal
	f.Attributes["operator"] = string(op)
	f.Attributes["exit_code"] = fmt.Sprintf("%d", res.ExitCode)
	f.AddEvidence(models.Evidence{
		Kind:    models.KindTriggeringInput,
		Content: string(mutated),
		Data:    mutated,
		Hash:    execution.FingerprintInput(mutated),
	})
	return f
}

// isAnomaly decides whether an unexpected/error result is a genuine anomaly
// relative to the baseline rather than an expected error we already know about.
func (e *Engine) isAnomaly(res *models.ExecutionResult) bool {
	if e.cfg.Base == nil || e.cfg.Base.ExitProfile == nil {
		return false
	}
	for _, code := range e.cfg.Base.ExitProfile.ExitCodes {
		if code == res.ExitCode {
			return false // expected from baseline
		}
	}
	return true
}

func (e *Engine) limitReached() bool {
	if e.cfg.ExecutionLimit > 0 && e.countExec.Load() >= e.cfg.ExecutionLimit {
		return true
	}
	return false
}

func timeoutFor(t time.Duration) time.Duration {
	if t <= 0 {
		return 2 * time.Second
	}
	return t
}

// processDeepAnalysis is a hook for the async deep path. In this build crash
// findings are created inline (they are cheap); minimization is not run inline
// on the hot path.
func (e *Engine) processDeepAnalysis(_ context.Context) {}

func (e *Engine) finalize() *Result {
	dur := time.Since(e.start)
	e.resultMu.Lock()
	defer e.resultMu.Unlock()
	r := e.result
	r.Executions = e.countExec.Load()
	r.CorpusSize = e.cfg.Corpus.Size()
	r.CoverageGained = int(e.countCoverage.Load())
	r.NewBehaviors = int(e.countBehavior.Load())
	r.UniqueCrashes = len(e.crashSet)
	r.Crashes = int(e.countCrash.Load())
	r.Hangs = int(e.countHang.Load())
	r.Anomalies = int(e.countAnomaly.Load())
	r.Timeouts = int(e.countTimeout.Load())
	r.Duration = dur
	if dur > 0 {
		r.ExecPerSec = float64(r.Executions) / dur.Seconds()
	}
	return r
}

func (e *Engine) emit6(_, _, _ bool) {
	// Reserved: use the configured emit hook when needed (hot-path minimal).
}

func (e *Engine) emitCrash(f *models.Finding) {
	if e.cfg.Emit != nil {
		e.cfg.Emit("error", events.CrashDetected, map[string]any{
			"finding_id": f.ID,
			"signal":     f.Attributes["signal"],
			"crash_sig":  f.CrashSign,
		})
	}
}

func targetID(t *models.Target) string {
	if t != nil {
		return t.ID
	}
	return ""
}

// LoadSeeds reads seed files deterministically into the corpus.
func LoadSeeds(c *corpus.Store, paths []string) (int, error) {
	count := 0
	for _, p := range paths {
		data, err := readFile(p)
		if err != nil {
			return count, err
		}
		if _, added := c.AddBytes(data, "file:"+p); added {
			count++
		}
	}
	return count, nil
}

func readFile(path string) ([]byte, error) {
	return readFileImpl(path)
}
