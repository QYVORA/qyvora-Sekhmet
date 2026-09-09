package cli

import (
	"bytes"
	"os"
	"strings"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/output"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func newReportCmd() *cobra.Command {
	var (
		sessionID string
		out       string
	)
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render a campaign report for a session",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if sessionID == "" {
				return errs.NewExitError(2, "a --session id is required")
			}
			sess, err := app.store.Load(sessionID)
			if err != nil {
				return errs.WrapExitError(2, "loading session", err)
			}
			body := renderSessionFormatted(sess, app.printer.Format())
			if out != "" {
				if err := os.WriteFile(out, []byte(body), 0o600); err != nil {
					return errs.WrapExitError(1, "writing report", err)
				}
				app.emitf("report written to %s", out)
				return nil
			}
			w := app.printer.Writer()
			_, _ = w.Write([]byte(body))
			if !strings.HasSuffix(body, "\n") {
				_, _ = w.Write([]byte("\n"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "session id to report")
	cmd.Flags().StringVar(&out, "out", "", "write report to file")
	return cmd
}

// renderSessionFormatted renders a session in the active output format. The
// human report is used only for the terminal; machine formats serialize the
// structured session model through the printer so the output stays valid.
func renderSessionFormatted(sess *models.Session, f output.Format) string {
	if f == output.FormatTerminal {
		return renderReport(sess)
	}
	var buf bytes.Buffer
	p := output.New()
	p.SetFormat(f)
	p.SetWriter(&buf)
	p.Print(sess)
	return buf.String()
}

func renderReport(sess *models.Session) string {
	var b strings.Builder
	b.WriteString("QYVORA Sekhmet Campaign Report\n")
	b.WriteString("Session: " + sess.ID + "  Target: " + sess.TargetID + "  State: " + string(sess.State) + "\n\n")

	if sess.Result != nil {
		b.WriteString("Summary\n-------\n")
		b.WriteString("  Executions:     " + itoaPre(int(sess.Result.Executions)) + "\n")
		b.WriteString("  Exec/sec:       " + f64str(sess.Result.ExecPerSec) + "\n")
		b.WriteString("  Corpus size:    " + itoaPre(sess.Result.CorpusSize) + "\n")
		b.WriteString("  New behaviors:  " + itoaPre(sess.Result.NewBehaviors) + "\n")
		b.WriteString("  Unique crashes: " + itoaPre(sess.Result.UniqueCrashes) + "\n")
		b.WriteString("  Hangs:          " + itoaPre(sess.Result.Hangs) + "\n")
		b.WriteString("  Strategy:       " + sess.Result.Strategy + "\n\n")
	}

	b.WriteString("Crashes\n-------\n")
	if len(sess.Crashes) == 0 {
		b.WriteString("  none\n")
	} else {
		for _, f := range sess.Crashes {
			b.WriteString("  - " + crashSigLabel(f.CrashSign) + "\n")
		}
	}

	b.WriteString("\nFindings\n--------\n")
	if len(sess.Findings) == 0 {
		b.WriteString("  none\n")
	} else {
		for _, f := range sess.Findings {
			b.WriteString("  - [" + string(f.Severity) + "] " + f.Title + "\n")
		}
	}
	return b.String()
}

func f64str(f float64) string {
	whole := int(f)
	frac := int((f - float64(whole)) * 1000)
	return itoaPre(whole) + "." + padStart(itoaPre(frac), 3)
}

func padStart(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}
