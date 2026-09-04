package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
)

func newCorpusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "corpus",
		Aliases: []string{"seeds"},
		Short:   "Manage the seed corpus",
	}
	cmd.AddCommand(newCorpusImportCmd())
	cmd.AddCommand(newCorpusListCmd())
	cmd.AddCommand(newCorpusCropCmd())
	return cmd
}

func newCorpusImportCmd() *cobra.Command {
	var source string
	cmd := &cobra.Command{
		Use:   "import <paths...>",
		Short: "Import seed files or directories into the corpus",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			c := corpus.New(corpusDir())
			added := 0
			for _, p := range args {
				if fi, err := os.Stat(p); err == nil && fi.IsDir() {
					n, err := c.ImportDir(p, source)
					if err != nil {
						return errs.WrapExitError(2, "importing dir", err)
					}
					added += n
				} else {
					if _, ok := c.AddBytes(mustRead(p), source); ok {
						added++
					}
				}
			}
			if err := c.Persist(); err != nil {
				return errs.WrapExitError(1, "persisting corpus", err)
			}
			app.emitf("imported %d seeds into %s", added, c.Dir())
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "import", "source label for imported seeds")
	return cmd
}

func newCorpusListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List seeds in the corpus",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := corpus.New(corpusDir())
			seeds := c.Prioritized()
			if app.printer.Format() == outputFormatJSON || app.printer.Format() == outputFormatYAML {
				app.printer.Print(seeds)
				return nil
			}
			rows := make([][]string, 0, len(seeds))
			for _, s := range seeds {
				rows = append(rows, []string{s.ID, itoaPre(s.Novelty), s.Source, itoaPre(len(s.Data))})
			}
			app.printer.PrintTable([]string{"id", "novelty", "source", "size"}, rows)
			return nil
		},
	}
}

func newCorpusCropCmd() *cobra.Command {
	var (
		max  int
		path string
	)
	cmd := &cobra.Command{
		Use:   "crop",
		Short: "Trim the corpus to the best N seeds",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := corpus.New(corpusDir())
			removed := c.TrimTo(max)
			if path != "" {
				if err := c.Persist(); err != nil {
					return errs.WrapExitError(1, "persisting corpus", err)
				}
			}
			app.emitf("cropped corpus to %d seeds (removed %d)", c.Size(), len(removed))
			return nil
		},
	}
	cmd.Flags().IntVar(&max, "max", 1000, "maximum number of seeds to keep")
	cmd.Flags().StringVar(&path, "dir", "", "corpus directory (default app corpus dir)")
	return cmd
}

func mustRead(p string) []byte {
	b, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return b
}

func itoaPre(n int) string {
	if n <= 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

var _ = errors.New
