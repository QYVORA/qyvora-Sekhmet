// Package console implements the interactive REPL for sekhmet. When run with
// no subcommand, sekhmet drops into an interactive console that can drive the
// baseline and fuzz commands, query sessions, and set the current target, all
// while surfacing live events. The console is thin: it dispatches to the same
// command handlers the CLI uses, so behaviour stays identical across both
// entry points.
package console

import (
	"context"
	"fmt"
	"strings"

	"github.com/chzyer/readline"
)

// Cmd is a dispatchable console command.
type Cmd struct {
	Name    string
	Aliases []string
	Help    string
	Run     func(ctx context.Context, args []string) error
}

// Host is the callback interface the CLI host must implement so the console
// can delegate to real framework commands.
type Host interface {
	// RunCommand executes a framework command line and returns an error
	// message ("" on success).
	RunCommand(ctx context.Context, argv []string) string
}

// Console is the interactive REPL session.
type Console struct {
	cmds  map[string]*Cmd
	order []string
	host  Host
}

// New returns a console backed by the given host.
func New(host Host) *Console {
	c := &Console{cmds: map[string]*Cmd{}, order: []string{}, host: host}
	c.registerBuiltins()
	return c
}

// Register adds an application command.
func (c *Console) Register(cmd *Cmd) {
	if cmd == nil || cmd.Name == "" {
		return
	}
	c.cmds[cmd.Name] = cmd
	c.order = append(c.order, cmd.Name)
}

func (c *Console) registerBuiltins() {
	c.Register(&Cmd{Name: "help", Help: "list console commands", Run: c.runHelp})
	c.Register(&Cmd{Name: "quit", Aliases: []string{"exit"}, Help: "exit the console", Run: func(_ context.Context, _ []string) error {
		return errQuit{}
	}})
}

type errQuit struct{}

func (errQuit) Error() string { return "quit" }

func (c *Console) runHelp(_ context.Context, _ []string) error {
	fmt.Println("commands:")
	for _, n := range c.order {
		cmd := c.cmds[n]
		if len(cmd.Aliases) > 0 {
			fmt.Printf("  %-8s (alias: %s) %s\n", cmd.Name, strings.Join(cmd.Aliases, ","), cmd.Help)
		} else {
			fmt.Printf("  %-8s %s\n", cmd.Name, cmd.Help)
		}
	}
	fmt.Println("  Any 'sekhmet ...' command line also works, e.g. 'fuzz --workers 8'.")
	return nil
}

// Run starts the interactive loop until EOF or quit.
func (c *Console) Run(ctx context.Context) error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "sekhmet> ",
		HistoryFile:       "/dev/null",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = rl.Close() }()
	fmt.Println("sekhmet console. Type 'help' for commands, 'quit' to exit.")

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := c.run(ctx, line); err != nil {
			if _, ok := err.(errQuit); ok {
				break
			}
			fmt.Fprintf(rl.Stderr(), "Error: %v\n", err)
		}
	}
	return nil
}

func (c *Console) run(ctx context.Context, line string) error {
	fields := strings.Fields(line)
	name := fields[0]
	args := fields[1:]
	if cmd, ok := c.find(name); ok {
		return cmd.Run(ctx, args)
	}
	// Fall back to the host (framework command) so any sekhmet subcommand
	// works from the REPL, e.g. "fuzz --workers 8".
	if c.host != nil {
		if msg := c.host.RunCommand(ctx, fields); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return nil
	}
	return fmt.Errorf("unknown command %q (type 'help')", name)
}

func (c *Console) find(name string) (*Cmd, bool) {
	if cmd, ok := c.cmds[name]; ok {
		return cmd, true
	}
	for _, cmd := range c.cmds {
		for _, a := range cmd.Aliases {
			if a == name {
				return cmd, true
			}
		}
	}
	return nil, false
}
