package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
)

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage fuzzing sessions",
	}
	cmd.AddCommand(newSessionListCmd())
	cmd.AddCommand(newSessionShowCmd())
	return cmd
}

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored sessions",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			ids, err := app.store.List()
			if err != nil {
				return errs.WrapExitError(1, "listing sessions", err)
			}
			rows := make([][]string, 0, len(ids))
			for _, id := range ids {
				rows = append(rows, []string{id, filepath.Base(id)})
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(ids)
				return nil
			}
			app.printer.PrintTable([]string{"id", "file"}, rows)
			return nil
		},
	}
}

func newSessionShowCmd() *cobra.Command {
	var sessionID string
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a stored session",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if sessionID == "" {
				return errs.NewExitError(2, "a --session id is required")
			}
			sess, err := app.store.Load(sessionID)
			if err != nil {
				return errs.WrapExitError(2, "loading session", err)
			}
			app.printer.Print(sess)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "session id to show")
	return cmd
}
