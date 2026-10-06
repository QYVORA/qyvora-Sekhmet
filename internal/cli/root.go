// Package cli implements the sekhmet command-line interface. The same binary
// is also the interactive console. The package wires configuration, logging,
// output formatting, the target manager and the fuzzing pipeline together and
// exposes the baseline/fuzz/corpus/report commands.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/config"
	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/logger"
	"github.com/QYVORA/qyvora-sekhmet/internal/output"
	"github.com/QYVORA/qyvora-sekhmet/internal/session"
	"github.com/QYVORA/qyvora-sekhmet/internal/target"
	"github.com/QYVORA/qyvora-sekhmet/internal/version"
)
var updateFlag bool



var app = newAppState()

const appDescription = `sekhmet is a terminal-first CLI for high-performance, baseline-aware,
feedback-driven fuzzing and vulnerability discovery. It first understands a
target's normal behaviour, then mutates and executes inputs against it,
classifying crashes, hangs and anomalies against that baseline rather than
fuzzing blindly.

Usage modes:
  sekhmet                                start the interactive console
  sekhmet baseline --target ./target     profile normal target behaviour
  sekhmet fuzz --target ./target         run a fuzzing campaign
  sekhmet analyze --session <id>         classify results and findings
  sekhmet corpus import|crops|list       manage the seed corpus
  sekhmet crashes --session <id>         list deduplicated crashes
  sekhmet minimize --input <file>        minimize an interesting input
  sekhmet replay --input <file>          reproduce an input against a target
  sekhmet session list|show              manage fuzzing sessions
  sekhmet report --session <id>          render a campaign report
  sekhmet target set|list|show           manage targets
  sekhmet wordlists list|search          SecLists integration
  sekhmet capabilities                   list machine-readable capabilities
  sekhmet version

Fuzzing targets are explicit. Remote targets (HTTP/network) require explicit
authorization acknowledgement; local targets are scoped to the declared path.
sekhmet is intended for use only on systems you are authorized to test.`

var rootCmd = &cobra.Command{
	Use:           "sekhmet",
	Short:         "High-performance fuzzing and vulnerability-discovery framework",
	Long:          appDescription,
	Version:       version.String(),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if app.initErr != nil {
			return errs.NewExitError(2, app.initErr.Error())
		}
		if app.stdoutOwned && app.printer.Format() != output.FormatTerminal {
			// stdout must carry exactly one machine stream. With the event
			// JSONL stream owning stdout, a machine report format cannot share
			// it: use --events stderr or --events <file>.
			return errs.NewExitError(2, "cannot combine --events stdout with a machine report format; use --events stderr or --events <file>")
		}
		return nil
	},
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errs.NewExitError(2, fmt.Sprintf("unknown command %q (try 'sekhmet --help')", args[0]))
		}
		return nil
	},
}

// Execute runs the root command against os.Args and returns the process exit
// code. It never calls os.Exit itself so callers control termination.
func Execute() int {
	return ExecuteArgs(os.Args[1:])
}

// ExecuteArgs runs the root command with an explicit argument vector and
// returns the process exit code.
func ExecuteArgs(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return ExecuteArgsContext(ctx, args)
}

// ExecuteArgsContext runs the root command with an explicit argument vector
// under a caller-supplied context.
//
// The interactive TUI needs this form: it runs commands in-process on its own
// goroutine and must be able to cancel one execution without tearing down the
// process. Driving the work from a context it owns, rather than from
// process-wide signal handling, is what makes Ctrl+C cancel the operation
// instead of the interface.
func ExecuteArgsContext(ctx context.Context, args []string) int {
	rootCmd.SetContext(ctx)
	rootCmd.SetArgs(args)

	// If --update is passed, route to the update subcommand regardless of
	// other positional arguments. This preserves compatibility with "tool --update".
	for _, a := range args {
		if a == "--update" || a == "-update" || a == "--update=true" {
			rootCmd.SetArgs([]string{"update"})
			break
		}
	}

	if err := rootCmd.Execute(); err != nil {
		var exitErr *errs.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, wrapErr(exitErr.Message))
			if exitErr.Cause != nil {
				fmt.Fprintln(os.Stderr, "  "+exitErr.Cause.Error())
			}
			return exitErr.Code
		}
		fmt.Fprintln(os.Stderr, wrapErr(err.Error()))
		return 1
	}
	if app.initErr != nil {
		fmt.Fprintln(os.Stderr, wrapErr(app.initErr.Error()))
		return 2
	}
	return 0
}

func init() {
	// The default action opens the interactive TUI. It is assigned here rather
	// than in the rootCmd literal because Go's initialization dependency
	// analysis follows references through function bodies: runTUI reaches
	// rootCmd, so naming it inside rootCmd's own initializer is a cycle.
	// Assigning in init is exempt from that analysis.
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runTUI(cmd.Root(), cmd.Context())
	}

	cobra.OnInitialize(initConfig)

	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return errs.NewExitError(2, err.Error())
	})

	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&app.cfgFile, "config", "c", "", "config file (default $HOME/.config/qyvora/sekhmet/config.yaml)")
	pf.BoolVarP(&app.verbose, "verbose", "v", false, "verbose output")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress non-error output")
	pf.StringVarP(&app.outputFmt, "output", "o", "", "output format: terminal, json, markdown, html, yaml")
	pf.BoolVar(&app.jsonOut, "json", false, "output in JSON format (shorthand for --output json)")
	pf.StringVar(&app.eventsF, "events", "", "emit a JSONL event stream to stdout, stderr, or a file path")
	pf.BoolVar(&app.dryRun, "dry-run", false, "resolve and print the fuzzing plan without executing")
	pf.StringVar(&app.timeout, "timeout", "", "default timeout for executions (e.g. 1s)")
	pf.BoolVar(&app.insecureTLS, "insecure-tls", false, "disable TLS certificate verification for HTTP endpoints (authorized testing only)")
	pf.BoolVar(&updateFlag, "update", false, "update the CLI to the latest official release")

	rootCmd.PersistentFlags().BoolP("authorized", "y", false, "confirm authorization scope non-interactively")

	rootCmd.AddCommand(commandTUI())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCapabilitiesCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newUpdatesCmd())
	rootCmd.AddCommand(newTargetCmd())
	rootCmd.AddCommand(newBaselineCmd())
	rootCmd.AddCommand(newFuzzCmd())
	rootCmd.AddCommand(newAnalyzeCmd())
	rootCmd.AddCommand(newCorpusCmd())
	rootCmd.AddCommand(newCrashesCmd())
	rootCmd.AddCommand(newMinimizeCmd())
	rootCmd.AddCommand(newReplayCmd())
	rootCmd.AddCommand(newSessionCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newWordlistsCmd())

	rootCmd.SetVersionTemplate(fmt.Sprintf("sekhmet %s\n", version.String()))
}

// initConfig loads configuration and initializes the shared logger, printer,
// target manager and session store. Failures are recorded as init errors so
// ExecuteArgs can report them without calling os.Exit.
func initConfig() {
	// The interactive TUI runs this command tree repeatedly in-process, so
	// every invocation starts from clean state. Without this, a flag set by one
	// run would silently apply to the next: stdoutOwned in particular decides
	// whether stdout is allowed to carry a machine report, and a stale true
	// would make an unrelated later command fail.
	app.stdoutOwned = false

	v, err := config.Load(app.cfgFile)
	if err != nil {
		app.initErr = errs.WrapExitError(2, "loading config", err)
		return
	}
	app.cfg = v
	initLogger()
	initPrinter()
	app.targets = target.NewManager(targetsDir(v))
	app.store = session.NewStore(v.GetString("session.dir"))
	if app.eventsF != "" {
		if err := app.resolveEvents(rootCmd.Context()); err != nil {
			app.initErr = errs.WrapExitError(2, "configuring events", err)
			return
		}
	}
}

func initLogger() {
	app.log = logger.New()
	app.log.SetLevel(logger.ParseLevel(app.cfg.GetString("log.level")))
	if app.verbose || app.cfg.GetBool("verbose") {
		app.log.SetVerbose(true)
	}
	if app.quiet || app.cfg.GetBool("quiet") {
		app.log.SetQuiet(true)
	}
}

func initPrinter() {
	app.printer = output.New()
	format := "terminal"
	switch {
	case app.outputFmt != "":
		format = app.outputFmt
	case app.jsonOut:
		format = "json"
	case app.cfg.GetBool("json"):
		format = "json"
	case app.cfg.IsSet("output"):
		if v, ok := app.cfg.Get("output").(string); ok && v != "" {
			format = v
		}
	}
	parsed, err := output.ParseFormat(format)
	if err != nil {
		app.initErr = errs.WrapExitError(2, "invalid --output format", err)
		return
	}
	app.printer.SetFormat(parsed)
	// ANSI color is a terminal-only nicety: disable it when stdout is not an
	// interactive device or when the caller opts out via NO_COLOR, so no
	// escape sequences leak into redirected or piped output.
	color.NoColor = !stdoutIsTerminal() || os.Getenv("NO_COLOR") != ""
}

// stdoutIsTerminal reports whether standard output is an interactive
// character device.
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func wrapErr(msg string) string {
	return color.New(color.FgRed, color.Bold).Sprint("Error: ") + msg
}