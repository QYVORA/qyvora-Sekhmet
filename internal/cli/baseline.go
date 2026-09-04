package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/baseline"
	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func newBaselineCmd() *cobra.Command {
	var (
		warmup  int
		samples int
	)
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Profile the target's normal behaviour before fuzzing",
		Long: `Profile the target's normal behaviour by running a warm-up phase
followed by sampled executions. sekhmet uses this baseline to classify crashes,
hangs and anomalies rather than fuzzing blindly.`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			t, err := app.requireTarget()
			if err != nil {
				return err
			}
			runner, err := execution.New(t, &execution.Options{
				UseStdin:    true,
				Args:        t.Args,
				HTTPTimeout: parseDurationOpt(app.timeout, 5*time.Second),
			})
			if err != nil {
				return errs.WrapExitError(2, "building target runner", err)
			}
			defer func() { _ = runner.Close() }()

			app.emitf("profiling target %s (%d samples, %d warmup)...",
				t.DisplayName(), samples, warmupEqual(warmup))

			base, err := baseline.Profile(context.Background(), runner, &baseline.Options{
				Warmup:  warmup,
				Samples: samples,
			})
			if err != nil {
				return errs.WrapExitError(1, "baseline profiling failed", err)
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(base)
				return nil
			}
			printBaseline(base)
			return nil
		},
	}
	cmd.Flags().IntVar(&warmup, "warmup", 5, "number of warm-up executions")
	cmd.Flags().IntVar(&samples, "samples", 30, "number of sampled executions")
	return cmd
}

func warmupEqual(w int) int { return w }

func printBaseline(base *models.Baseline) {
	samplesStr := "-"
	if base.Timing != nil {
		samplesStr = itoaPre(base.Timing.Samples)
	}
	rows := [][]string{
		{"executions/samples", samplesStr},
	}
	if base.ExitProfile != nil {
		rows = append(rows, []string{"stdout bytes", f64str(base.ExitProfile.StdoutSize.Mean)})
		rows = append(rows, []string{"stderr bytes", f64str(base.ExitProfile.StderrSize.Mean)})
		rows = append(rows, []string{"exit codes", joinInts(base.ExitProfile.ExitCodes)})
		rows = append(rows, []string{"signals", joinStrs(base.ExitProfile.Signals)})
	}
	if base.Timing != nil {
		rows = append(rows, []string{"mean duration", statMs(base.Timing.Normal.Mean)})
		rows = append(rows, []string{"slow floor (p95)", statMs(base.Timing.SlowFloor)})
		rows = append(rows, []string{"adaptive timeout", statMs(base.Timing.Timeout)})
	}
	rows = append(rows, []string{"coverage kind", base.Coverage.Kind})
	app.printer.PrintTable([]string{"metric", "value"}, rows)
}

func statMs(f float64) string { return f64str(f) + " ms" }

func joinInts(v []int) string {
	if len(v) == 0 {
		return "-"
	}
	out := ""
	for i, x := range v {
		if i > 0 {
			out += ","
		}
		out += itoaPre(x)
	}
	return out
}

func joinStrs(v []string) string {
	if len(v) == 0 {
		return "-"
	}
	out := ""
	for i, s := range v {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func parseDurationOpt(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
