package cli

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/minimization"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func newMinimizeCmd() *cobra.Command {
	var (
		input   string
		out     string
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "minimize",
		Short: "Minimize an interesting input against the current target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			t, err := app.requireTarget()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(input)
			if err != nil {
				return errs.WrapExitError(2, "reading input", err)
			}
			runner, err := execution.New(t, &execution.Options{
				UseStdin:    true,
				Args:        t.Args,
				HTTPTimeout: timeoutOr(timeout, 5*time.Second),
			})
			if err != nil {
				return errs.WrapExitError(2, "building runner", err)
			}
			defer func() { _ = runner.Close() }()

			base, err := buildBaseline(runner)
			if err != nil {
				return errs.WrapExitError(1, "profiling baseline", err)
			}
			expected := make(map[int]bool)
			if base != nil && base.ExitProfile != nil {
				for _, c := range base.ExitProfile.ExitCodes {
					expected[c] = true
				}
			}

			perTimeout := timeoutOr(timeout, 2*time.Second)
			reduced := minimization.Minimize(cmd.Context(), runner, data, perTimeout, func(res *models.ExecutionResult) bool {
				if res.Signal != "" || res.TimedOut {
					return true
				}
				if res.ExitCode != 0 && !expected[res.ExitCode] {
					return true
				}
				return false
			})
			if out == "" {
				out = input + ".min"
			}
			if err := os.WriteFile(out, reduced, 0o600); err != nil {
				return errs.WrapExitError(1, "writing minimized input", err)
			}
			app.emitf("minimized %d -> %d bytes (%s)", len(data), len(reduced), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "input file to minimize")
	cmd.Flags().StringVar(&out, "out", "", "output file (default <input>.min)")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Second, "per-execution timeout")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
