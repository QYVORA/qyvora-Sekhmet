// Package console implements the interactive REPL for sekhmet. When run with
// no subcommand, sekhmet drops into an interactive console that can drive the
// baseline and fuzz commands, query sessions, and set the current target, all
// while surfacing live events. The console is thin: it dispatches to the same
// command handlers the CLI uses, so behaviour stays identical across both
// entry points.
package console

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"

	"github.com/QYVORA/qyvora-sekhmet/internal/banner"
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
	c.Register(&Cmd{Name: "banner", Help: "show the brand ASCII banner", Run: func(_ context.Context, _ []string) error {
		printBanner()
		return nil
	}})
	c.Register(&Cmd{Name: "quit", Aliases: []string{"exit"}, Help: "exit the console", Run: func(_ context.Context, _ []string) error {
		return errQuit{}
	}})
}

// printBanner writes the canonical startup banner followed by a short footer.
//
// The banner is drawn through banner.Render, which resolves the QYVORA accent
// from the environment once. This console writes straight to stdout rather than
// through a colour-aware writer, so the environment is the only signal available
// here: a NO_COLOR request, or a terminal that cannot show the accent, gets the
// plain art.
func printBanner() {
	fmt.Print(banner.Render())
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

// Run starts the interactive loop until EOF or quit. When stdin is not an
// interactive terminal the console falls back to a plain line reader so piped
// input works without line-editing escapes on stdout.
func (c *Console) Run(ctx context.Context) error {
	if !stdinIsTerminal() {
		return c.runPlain(ctx)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "sekhmet> ",
		HistoryFile:       c.historyPath(),
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
		AutoComplete:      readline.NewPrefixCompleter(c.completer()...),
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "line editing unavailable (%v); continuing in plain mode\n", err)
		return c.runPlain(ctx)
	}
	defer func() { _ = rl.Close() }()
	printBanner()
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

// runPlain executes console lines from a non-interactive stdin, writing only
// command output and errors (never banners or control sequences) to stdout.
func (c *Console) runPlain(ctx context.Context) error {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if err := c.run(ctx, line); err != nil {
			if _, ok := err.(errQuit); ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
	return sc.Err()
}

// historyPath returns the console history file, creating its parent directory.
func (c *Console) historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sekhmet_history"
	}
	dir := filepath.Join(home, ".qyvora")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ".sekhmet_history"
	}
	return filepath.Join(dir, "sekhmet_history")
}

// stdinIsTerminal reports whether standard input is an interactive device.
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func (c *Console) completer() []readline.PrefixCompleterInterface {
	return []readline.PrefixCompleterInterface{
		readline.PcItem("help"),
		readline.PcItem("banner"),
		readline.PcItem("quit", readline.PcItem("exit")),
		readline.PcItem("baseline"),
		readline.PcItem("fuzz"),
		readline.PcItem("analyze"),
		readline.PcItem("corpus"),
		readline.PcItem("crashes"),
		readline.PcItem("minimize"),
		readline.PcItem("replay"),
		readline.PcItem("session"),
		readline.PcItem("report"),
		readline.PcItem("target"),
		readline.PcItem("wordlists"),
		readline.PcItem("capabilities"),
		readline.PcItem("updates"),
		readline.PcItem("version"),
	}
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
