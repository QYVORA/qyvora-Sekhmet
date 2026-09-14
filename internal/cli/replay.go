package cli

import (
	"os"
	"time"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/exitcode"
)

func newReplayCmd() *cobra.Command {
	var (
		input   string
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay a single input against the current target",
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
				InsecureTLS: app.insecureTLS,
			})
			if err != nil {
				return errs.WrapExitError(2, "building runner", err)
			}
			defer func() { _ = runner.Close() }()

			res, err := runner.Exec(cmd.Context(), data, timeoutOr(timeout, 5*time.Second))
			if err != nil {
				return errs.WrapExitError(1, "execution failed", err)
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(res)
				return nil
			}
			app.emitf("exit:     %d", res.ExitCode)
			app.emitf("class:    %s", res.ExitClass)
			app.emitf("signal:   %s", res.Signal)
			app.emitf("status:   %d", res.StatusCode)
			app.emitf("duration: %s", res.Duration)
			app.emitf("timeout:  %v", res.TimedOut)
			app.emitf("stdout:   %d bytes", res.StdoutSize)
			app.emitf("stderr:   %d bytes", res.StderrSize)
			if cmd.Context().Err() != nil {
				return errs.NewExitError(exitcode.Interrupted, "replay interrupted")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "input file to replay")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "execution timeout")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
