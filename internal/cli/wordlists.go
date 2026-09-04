package cli

import (
	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/wordlists"
)

func newWordlistsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wordlists",
		Aliases: []string{"wl"},
		Short:   "SecLists integration: list categories and search wordlists",
	}
	cmd.AddCommand(newWordlistsListCmd())
	cmd.AddCommand(newWordlistsSearchCmd())
	cmd.AddCommand(newWordlistsPathCmd())
	return cmd
}

func seclistsRoot() (string, error) {
	dir := app.cfg.GetString("seclists.dir")
	root := wordlists.SecListsDir(dir)
	if root == "" {
		return "", errs.NewExitError(2, "SecLists not found; set QYVORA_SEKHMET_SECLISTS_DIR or seclists.dir")
	}
	return root, nil
}

func newWordlistsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SecLists categories",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := seclistsRoot()
			if err != nil {
				return err
			}
			cats, err := wordlists.Categories(root)
			if err != nil {
				return errs.WrapExitError(2, "listing categories", err)
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(cats)
				return nil
			}
			rows := make([][]string, 0, len(cats))
			for _, c := range cats {
				rows = append(rows, []string{c.RelPath, itoaPre(c.Files)})
			}
			app.printer.PrintTable([]string{"category", "files"}, rows)
			return nil
		},
	}
}

func newWordlistsSearchCmd() *cobra.Command {
	var kw []string
	cmd := &cobra.Command{
		Use:   "search <keywords...>",
		Short: "Search SecLists for matching wordlist files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			root, err := seclistsRoot()
			if err != nil {
				return err
			}
			kws := append(append([]string{}, kw...), args...)
			results, err := wordlists.Search(root, kws, 3)
			if err != nil {
				return errs.WrapExitError(2, "searching wordlists", err)
			}
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(results)
				return nil
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{r.Name, r.Path, itoaPre(r.Estimate)})
			}
			app.printer.PrintTable([]string{"name", "path", "est. entries"}, rows)
			if len(results) == 0 {
				app.emitf("no matches")
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&kw, "kw", nil, "additional keywords (repeatable)")
	return cmd
}

func newWordlistsPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved SecLists directory",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := seclistsRoot()
			if err != nil {
				return err
			}
			app.emitf("%s", root)
			return nil
		},
	}
}
