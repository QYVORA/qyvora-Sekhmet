package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/events"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/exitcode"
	"github.com/QYVORA/qyvora-sekhmet/internal/fuzz"
	"github.com/QYVORA/qyvora-sekhmet/internal/mutation"
	"github.com/QYVORA/qyvora-sekhmet/internal/output"
	"github.com/QYVORA/qyvora-sekhmet/internal/safety"
	"github.com/QYVORA/qyvora-sekhmet/internal/scheduler"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func newFuzzCmd() *cobra.Command {
	var (
		workers       int
		executions    int64
		runtime       time.Duration
		timeout       time.Duration
		seedFiles     []string
		strategy      string
		deterministic bool
		seedRNG       int64
		retention     int
	)
	cmd := &cobra.Command{
		Use:   "fuzz",
		Short: "Run a fuzzing campaign against the current target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			t, err := app.requireTarget()
			if err != nil {
				return err
			}
			if app.dryRun {
				printDryRunPlan(t, workers, executions, runtime, strategy, deterministic)
				return nil
			}
			runner, err := execution.New(t, &execution.Options{
				UseStdin:    true,
				Args:        t.Args,
				HTTPTimeout: timeoutOr(timeout, 5*time.Second),
				MaxOutput:   int(safety.DefaultLimits().MaxOutputBytes),
				InsecureTLS: app.insecureTLS,
			})
			if err != nil {
				return errs.WrapExitError(2, "building target runner", err)
			}
			defer func() { _ = runner.Close() }()

			// Baseline first: never fuzz blindly.
			built, err := buildBaseline(cmd.Context(), runner)
			if err != nil {
				return errs.WrapExitError(1, "baseline profiling before fuzz", err)
			}

			corp := corpus.New(corpusDir())
			var seeds [][]byte
			if len(seedFiles) > 0 {
				seeds, err = loadSeedBytes(seedFiles)
				if err != nil {
					return errs.WrapExitError(2, "loading seeds", err)
				}
			}

			campaign := fuzz.Campaign{
				Target:         t,
				Base:           built,
				Runner:         runner,
				Corpus:         corp,
				Seeds:          seeds,
				Workers:        workers,
				Timeout:        timeoutOr(timeout, 2*time.Second),
				HardRuntime:    runtime,
				ExecutionLimit: executions,
				Strategy:       scheduler.ParseStrategy(strategy),
				Deterministic:  deterministic,
				SeedRNG:        seedRNG,
				Retention:      retention,
				Emit:           appEmitEvent,
				OnNewBehavior:  func(_ []string, _ mutation.Operator) {},
				Safety:         buildGuardian(workers, executions, runtime, len(seeds)),
			}

			app.emitf("campaign: %d workers, max %d executions, runtime %s",
				workers, executions, runtimeString(runtime))
			app.emitEvent(events.ScanStarted, map[string]any{
				"target": t.DisplayName(), "strategy": strategy, "workers": workers,
			})
			app.emitEvent(events.CampaignStarted, map[string]any{
				"target": t.DisplayName(), "strategy": strategy, "workers": workers,
			})

			res, sess, err := fuzz.New(campaign).Run(cmd.Context())
			if err != nil {
				if cmd.Context().Err() != nil {
					return errs.NewExitError(exitcode.Interrupted, "campaign interrupted")
				}
				return errs.WrapExitError(1, "campaign failed", err)
			}
			sess.Result = buildSessionResult(res, sess.ID, t.ID, workers)
			sess.Finish()
			path, err := app.persistSession(sess)
			if err != nil {
				return errs.WrapExitError(1, "persisting session", err)
			}
			app.emitf("session saved: %s", path)
			printCampaignSummary(res)
			printFindingsSummary(sess)
			app.emitEvent(events.CampaignCompleted, map[string]any{
				"executions": res.Executions, "unique_crashes": res.UniqueCrashes,
				"new_behaviors": res.NewBehaviors, "corpus_size": res.CorpusSize,
			})
			app.emitEvent(events.ScanCompleted, map[string]any{
				"session": sess.ID, "crashes": len(sess.Crashes), "findings": len(sess.Findings),
			})
			if cmd.Context().Err() != nil {
				return errs.NewExitError(exitcode.Interrupted, "campaign interrupted; partial session saved")
			}
			// Machine formats render the full session on stdout (the fixed
			// contract: `fuzz -o json` must never emit zero bytes). Human
			// summaries above already routed to stderr in machine mode.
			if app.printer.Format() != output.FormatTerminal {
				body := renderSessionFormatted(sess, app.printer.Format())
				out := app.printer.Writer()
				if app.stdoutOwned {
					out = os.Stderr
				}
				_, _ = out.Write([]byte(body))
				if !strings.HasSuffix(body, "\n") {
					_, _ = fmt.Fprintln(out)
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&workers, "workers", 4, "number of parallel workers")
	cmd.Flags().Int64Var(&executions, "executions", 0, "max executions (0 = unlimited)")
	cmd.Flags().DurationVar(&runtime, "runtime", 0, "hard runtime limit (e.g. 30s, 5m)")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Second, "per-execution timeout")
	cmd.Flags().StringSliceVar(&seedFiles, "seed", nil, "seed input file (repeatable)")
	cmd.Flags().StringVar(&strategy, "strategy", "adaptive", "scheduling strategy: fast, explore, exploit, rare, balanced, adaptive")
	cmd.Flags().BoolVar(&deterministic, "deterministic", false, "use a fixed RNG for reproducible runs")
	cmd.Flags().Int64Var(&seedRNG, "rng-seed", 1337, "RNG seed when --deterministic")
	cmd.Flags().IntVar(&retention, "retention", 100000, "corpus retention cap")
	return cmd
}

func timeoutOr(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}

func runtimeString(d time.Duration) string {
	if d <= 0 {
		return "unlimited"
	}
	return d.String()
}

// buildGuardian constructs the safety guard for a campaign from CLI flags.
func buildGuardian(workers int, executions int64, runtime time.Duration, seedCount int) *safety.Guardian {
	l := safety.DefaultLimits()
	l.MaxConcurrent = workers
	l.MaxExecutions = executions
	l.MaxRuntime = runtime
	l.RequireAuthorized = true
	_ = seedCount
	guard := safety.NewGuardian(l)
	return guard
}

func printDryRunPlan(t interface{ DisplayName() string }, workers int, executions int64, runtime time.Duration, strategy string, det bool) {
	app.emitf("[dry-run] target:         %s", t.DisplayName())
	app.emitf("[dry-run] workers:        %d", workers)
	app.emitf("[dry-run] max executions: %d", executions)
	app.emitf("[dry-run] runtime limit:  %s", runtimeString(runtime))
	app.emitf("[dry-run] strategy:       %s", strategy)
	app.emitf("[dry-run] deterministic:  %v", det)
	app.emitf("[dry-run] plan resolved, not executing.")
}

func buildSessionResult(res *fuzz.Result, id, targetID string, workers int) *models.SessionResult {
	return &models.SessionResult{
		SessionID:      id,
		TargetID:       targetID,
		Runtime:        res.Duration.String(),
		Executions:     res.Executions,
		ExecPerSec:     res.ExecPerSec,
		CorpusSize:     res.CorpusSize,
		CoverageGained: res.CoverageGained,
		NewBehaviors:   res.NewBehaviors,
		UniqueCrashes:  res.UniqueCrashes,
		Hangs:          res.Hangs,
		Anomalies:      res.Anomalies,
		Timeouts:       res.Timeouts,
		Strategy:       res.Strategy,
		Workers:        workers,
		State:          models.SessionFinished,
	}
}

func printCampaignSummary(res *fuzz.Result) {
	app.emitf("")
	app.emitf("campaign complete in %s", res.Duration)
	app.emitf("  executions:       %d (%.1f/s)", res.Executions, res.ExecPerSec)
	app.emitf("  corpus size:      %d", res.CorpusSize)
	app.emitf("  new behaviors:    %d", res.NewBehaviors)
	app.emitf("  unique crashes:   %d (total %d)", res.UniqueCrashes, res.Crashes)
	app.emitf("  hangs/timeouts:   %d / %d", res.Hangs, res.Timeouts)
	app.emitf("  anomalies:        %d", res.Anomalies)
}

func printFindingsSummary(sess *models.Session) {
	if len(sess.Crashes) == 0 && len(sess.Findings) == 0 {
		app.emitf("  no findings this run")
		return
	}
	app.emitf("  findings:         %d crash, %d total", len(sess.Crashes), len(sess.Findings))
}

func corpusDir() string {
	return app.store.Dir() + "/corpus"
}

func appEmitEvent(level, name string, data map[string]any) {
	if app.eventStream != nil {
		app.eventStream.Emit(level, name, data)
	}
}

// emitEvent emits an informational event on the active JSONL stream.
func (a *appState) emitEvent(name string, data map[string]any) {
	if a.eventStream != nil {
		a.eventStream.Emit(events.LevelInfo, name, data)
	}
}

func loadSeedBytes(paths []string) ([][]byte, error) {
	var out [][]byte
	for _, p := range paths {
		b, err := readSeedFile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}
