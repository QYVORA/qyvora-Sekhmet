package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/viper"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/events"
	"github.com/QYVORA/qyvora-sekhmet/internal/exitcode"
	"github.com/QYVORA/qyvora-sekhmet/internal/logger"
	"github.com/QYVORA/qyvora-sekhmet/internal/output"
	"github.com/QYVORA/qyvora-sekhmet/internal/session"
	"github.com/QYVORA/qyvora-sekhmet/internal/target"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

type appState struct {
	cfg     *viper.Viper
	log     *logger.Logger
	printer *output.Printer
	targets *target.Manager
	store   *session.Store

	eventStream *events.Stream
	eventSink   io.Writer

	// stdoutOwned is set when --events stdout is active: stdout then carries
	// only the JSONL event stream, so report rendering routes to stderr.
	stdoutOwned bool

	cfgFile     string
	verbose     bool
	quiet       bool
	jsonOut     bool
	outputFmt   string
	eventsF     string
	dryRun      bool
	timeout     string
	insecureTLS bool

	initErr error
}

func newAppState() *appState {
	return &appState{}
}

func (a *appState) requireTarget() (*models.Target, error) {
	t := a.targets.Current()
	if t == nil {
		return nil, errs.NewExitError(2, "no target selected; run 'sekhmet target set' first")
	}
	if !t.Authorized() && !t.Sim {
		return nil, errs.NewExitError(exitcode.AuthorizationRefused, "current target is not authorized: "+t.DisplayName())
	}
	return t, nil
}

func (a *appState) persistSession(sess *models.Session) (string, error) {
	path, err := a.store.Save(sess)
	if err != nil {
		return "", err
	}
	sess.OutputDir = a.store.Dir()
	return path, nil
}

// emitf writes an informational line. In terminal mode it goes to the output
// writer; in machine-readable formats it goes to stderr so stdout stays pure.
func (a *appState) emitf(format string, args ...any) {
	if a.printer.Format() == output.FormatTerminal {
		fmt.Fprintf(a.printer.Writer(), format+"\n", args...)
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (a *appState) resolveEvents(_ context.Context) error {
	var w io.Writer
	switch strings.ToLower(a.eventsF) {
	case "", "off", "none", "disable", "disabled":
		return nil
	case "stdout":
		w = os.Stdout
		a.stdoutOwned = true
		// stdout carries only the JSONL event stream; route every human and
		// report line off it.
		a.printer.SetWriter(os.Stderr)
	case "stderr":
		w = os.Stderr
	default:
		// Truncated, not appended, so one file holds exactly one run's
		// events. Appending left no run boundary in the file, which matters
		// to anything tailing it.
		f, err := os.OpenFile(a.eventsF, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("opening events file: %w", err)
		}
		w = f
	}
	a.eventStream = events.NewStream(w)
	a.eventSink = w
	return nil
}
