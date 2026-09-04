package cli

import (
	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
)

func newCrashesCmd() *cobra.Command {
	var sessionID string
	cmd := &cobra.Command{
		Use:   "crashes",
		Short: "List deduplicated crashes from a session",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if sessionID == "" {
				return errs.NewExitError(2, "a --session id is required")
			}
			sess, err := app.store.Load(sessionID)
			if err != nil {
				return errs.WrapExitError(2, "loading session", err)
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(sess.Crashes)
				return nil
			}
			rows := make([][]string, 0, len(sess.Crashes))
			for _, f := range sess.Crashes {
				rows = append(rows, []string{
					f.ID, crashSigLabel(f.CrashSign), string(f.Classification), f.Attributes["signal"],
				})
			}
			app.printer.PrintTable([]string{"id", "signature", "classification", "signal"}, rows)
			app.emitf("")
			app.emitf("%d unique crash(es)", len(sess.Crashes))
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "session id to read crashes from")
	return cmd
}

func crashSigLabel(s string) string {
	if len(s) > 24 {
		return s[:24]
	}
	return s
}
