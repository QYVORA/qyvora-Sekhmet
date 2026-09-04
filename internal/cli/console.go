package cli

import (
	"context"
	"strings"

	"github.com/QYVORA/qyvora-sekhmet/internal/console"
)

// hostAdapter lets the interactive console dispatch framework commands back
// through the same cobra root command the CLI uses, so both surfaces share
// identical behaviour.
type hostAdapter struct{}

func (hostAdapter) RunCommand(ctx context.Context, argv []string) string {
	exec := rootCmd.Root()
	exec.SetArgs(argv)
	exec.SetContext(ctx)
	if err := exec.Execute(); err != nil {
		return err.Error()
	}
	return ""
}

// runConsole starts the interactive REPL.
func runConsole(ctx context.Context) error {
	con := console.New(hostAdapter{})
	con.Register(&console.Cmd{
		Name: "target",
		Help: "work with the current target (" + strings.Join([]string{"set", "list", "show"}, ", ") + ")",
		Run:  func(_ context.Context, _ []string) error { return nil },
	})
	return con.Run(ctx)
}
