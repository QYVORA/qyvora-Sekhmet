package cli

import (
	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/version"
)

// newUpdatesCmd reports the current build version. sekhmet intentionally does
// not phone home for updates; it prints local build info and how to update.
func newUpdatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "updates",
		Aliases: []string{"update", "upgrade"},
		Short:   "Report version and update channel",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			info := version.GetInfo()
			app.emitf("sekhmet %s (framework %s, built %s)", version.String(), info.Framework, emptyDash(info.Date))
			app.emitf("version check:        offline (no automatic update checks)")
			app.emitf("update channel:       github.com/QYVORA/qyvora-Sekhmet")
			return nil
		},
	}
	return cmd
}
