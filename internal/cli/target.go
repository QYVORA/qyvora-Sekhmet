package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// newTargetCmd builds the `sekhmet target` command tree (set/list/show).
func newTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Manage fuzzing targets",
	}
	cmd.AddCommand(newTargetSetCmd())
	cmd.AddCommand(newTargetListCmd())
	cmd.AddCommand(newTargetShowCmd())
	return cmd
}

func newTargetSetCmd() *cobra.Command {
	var (
		tName string
		tType string
		path  string
		end   string
		args  []string
		input string
		auth  bool
		sim   bool
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the current fuzzing target",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if tName == "" {
				tName = displayDefaultName(path, end, sim)
			}
			tt := models.ParseTargetType(tType)
			if sim {
				tt = models.TargetSimulation
			}
			now := time.Now().UTC()
			authz := models.Authorization{Granted: auth, GrantedAt: now}
			t := &models.Target{
				Name:      tName,
				Type:      tt,
				Path:      path,
				Endpoint:  end,
				Args:      args,
				InputFile: input,
				Auth:      authz,
				Sim:       sim,
				CreatedAt: now,
			}
			if sim {
				t.Auth = models.Authorization{Granted: true, GrantedAt: now}
			}
			if err := app.targets.Set(t); err != nil {
				return errs.WrapExitError(2, "setting target", err)
			}
			app.emitf("target set: %s", t.DisplayName())
			return nil
		},
	}
	cmd.Flags().StringVar(&tName, "name", "", "target name")
	cmd.Flags().StringVar(&tType, "type", "cli", "target type: cli, http, network, simulation")
	cmd.Flags().StringVar(&path, "path", "", "executable or file path for cli targets")
	cmd.Flags().StringVar(&end, "endpoint", "", "http URL or host:port for http/network targets")
	cmd.Flags().StringSliceVar(&args, "arg", nil, "argv template (use {fuzz}, {stdin})")
	cmd.Flags().StringVar(&input, "input-file", "", "write input to this file instead of stdin")
	cmd.Flags().BoolVar(&auth, "authorize", false, "confirm authorization scope for this target")
	cmd.Flags().BoolVar(&sim, "sim", false, "use the deterministic simulation target")
	return cmd
}

func newTargetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured targets",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			list := app.targets.List()
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(list)
				return nil
			}
			rows := make([][]string, 0, len(list))
			for _, t := range list {
				rows = append(rows, []string{
					t.Name, string(t.Type), t.DisplayName(),
					yes(t.Authorized()),
				})
			}
			app.printer.PrintTable([]string{"name", "type", "target", "authorized"}, rows)
			return nil
		},
	}
}

func newTargetShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current target",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			t := app.targets.Current()
			if t == nil {
				return errs.NewExitError(2, "no target selected")
			}
			app.printer.Print(t)
			return nil
		},
	}
}

func displayDefaultName(path, end string, sim bool) string {
	if sim {
		return "simulation"
	}
	if path != "" {
		return pathBase(path)
	}
	if end != "" {
		return end
	}
	return "target"
}

func pathBase(p string) string {
	idx := strings.LastIndexAny(p, "/\\")
	if idx >= 0 && idx < len(p)-1 {
		return p[idx+1:]
	}
	return p
}
