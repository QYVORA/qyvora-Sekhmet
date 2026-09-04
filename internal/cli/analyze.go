package cli

import (
	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func newAnalyzeCmd() *cobra.Command {
	var (
		sessionID string
		status    string
	)
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Classify and summarize the findings of a session",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if sessionID == "" {
				return errs.NewExitError(2, "a --session id is required")
			}
			sess, err := app.store.Load(sessionID)
			if err != nil {
				return errs.WrapExitError(2, "loading session", err)
			}
			findings := sess.Findings
			if status != "" {
				var filtered []*models.Finding
				for _, f := range findings {
					if string(f.Status) == status {
						filtered = append(filtered, f)
					}
				}
				findings = filtered
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(findings)
				return nil
			}
			sum := summarizeFindings(sess)
			app.emitf("session %-8s %s", sess.ID, sess.State)
			app.emitf("  total findings: %d", sum.Total)
			app.emitf("  high:           %d", sum.High)
			app.emitf("  medium:         %d", sum.Medium)
			app.emitf("  low:            %d", sum.Low)
			app.emitf("  info:           %d", sum.Info)
			app.emitf("  crashes:        %d", sum.Crashes)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionID, "session", "", "session id to analyze")
	cmd.Flags().StringVar(&status, "status", "", "filter by finding status")
	return cmd
}

type findingSummary struct {
	Total   int
	High    int
	Medium  int
	Low     int
	Info    int
	Crashes int
}

func summarizeFindings(sess *models.Session) findingSummary {
	s := findingSummary{
		Total:   len(sess.Findings) + len(sess.Crashes),
		Crashes: len(sess.Crashes),
	}
	for _, f := range sess.Findings {
		switch string(f.Severity) {
		case "high":
			s.High++
		case "medium":
			s.Medium++
		case "low":
			s.Low++
		default:
			s.Info++
		}
	}
	return s
}
