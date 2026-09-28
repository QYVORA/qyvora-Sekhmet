package cli

import (
	"context"
	"fmt"
	"os"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/version"
	"github.com/QYVORA/qyvora-tui"
)

// runTUI starts the interactive terminal application.
//
// The TUI is a presentation layer over the same command tree the one-shot CLI
// exposes. It runs commands in-process through ExecuteArgsContext and reads
// the structured JSONL event stream, so it never parses human-readable output
// and a command behaves identically whether it was typed at a prompt or in a
// script.
//
// The root command is passed in rather than referenced directly: the root's own
// RunE calls this function, so naming the package-level root here would be an
// initialisation cycle.
func runTUI(root *cobra.Command, ctx context.Context) error {
	// A TUI needs a terminal. When stdout is redirected, or when the binary is
	// driven by something that is not a human, fall through to ordinary
	// behaviour: printing the command list is useful, and it is what a piped
	// or CI invocation needs. Launching a full-screen interface into a pipe
	// would fill it with escape codes and destroy the machine-readable output
	// the tool exists to produce.
	if !tui.IsInteractive(os.Stdout) {
		return root.Help()
	}

	// A machine event destination and the interface are contradictory: one
	// screen cannot hand the same bytes to a renderer and to a file. The
	// destination used to be ignored in silence, so a bare
	// `--events out.jsonl` opened the session and wrote no file.
	//
	// The flag has to have been *asked for*, not merely be set. This tool's
	// --events may default to a real destination so the stream is always on, and
	// testing the value alone would refuse every ordinary interactive run.
	if root.Flags().Changed("events") && !eventsDisabled(app.eventsF) {
		return errs.NewExitError(2, "cannot open the interactive session with a machine event destination (--events); the session transcript is already its event stream. Run a command for machine output, or drop --events to use the session.")
	}

	runner := &tui.InProcessRunner{
		ToolName: "sekhmet",
		Execute:  ExecuteArgsContext,
		Meta:     tuiCommands(root),
	}

	code, err := tui.Run(tui.Config{
		Title:   "QYVORA / SEKHMET",
		Version: version.String(),
		Runner:  runner,
		Out:     os.Stdout,
		// The tool's own progress output is discarded rather than shown: it
		// would be redrawn under the TUI's own frames and read as noise. The
		// transcript carries the event stream, which is the same information
		// in a form the interface can lay out.
		Err: nil,
	})
	if err != nil {
		if tui.IsNotInteractive(err) {
			return root.Help()
		}
		return err
	}
	if code != 0 {
		return &exitStatusError{code: code}
	}
	return nil
}

// commandTUI returns the explicit form of the interactive command, so the TUI
// can be started without relying on the bare invocation.
func commandTUI() *cobra.Command {
	return &cobra.Command{
		Use:     "tui",
		Aliases: []string{"console"},
		Short:   "start the interactive terminal application",
		Long: "Start the QYVORA interactive terminal application.\n\n" +
			"Commands are entered at the prompt and run through the same engine as the\n" +
			"one-shot CLI, with the structured event stream rendered as execution blocks.\n" +
			"Ctrl+C stops the running command; Ctrl+D leaves.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd.Root(), cmd.Context())
		},
	}
}

// exitStatusError carries a non-zero exit code out of the TUI so the process
// still reports the last command's status.
type exitStatusError struct{ code int }

func (e *exitStatusError) Error() string {
	return fmt.Sprintf("last command exited with status %d", e.code)
}

// tuiCommands derives completion metadata from the live command tree.
//
// Reading it from Cobra rather than from a hand-written list means completion
// cannot drift from the commands that actually exist: adding a command makes it
// completable with no second edit. The adapter exists because the shared TUI
// deliberately does not depend on Cobra -- which command framework a tool uses
// is the tool's choice, not the interface's.
func tuiCommands(root *cobra.Command) []tui.Command {
	return tui.CollectCommands(cobraNode{root})
}

// cobraNode adapts a Cobra command to the TUI's command-tree interface.
type cobraNode struct{ c *cobra.Command }

func (n cobraNode) Name() string      { return n.c.Name() }
func (n cobraNode) Short() string     { return n.c.Short }
func (n cobraNode) Hidden() bool      { return n.c.Hidden }
func (n cobraNode) Aliases() []string { return n.c.Aliases }

func (n cobraNode) Children() []tui.CommandNode {
	out := make([]tui.CommandNode, 0, len(n.c.Commands()))
	for _, c := range n.c.Commands() {
		out = append(out, cobraNode{c})
	}
	return out
}
