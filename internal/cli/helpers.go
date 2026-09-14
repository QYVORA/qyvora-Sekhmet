package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/QYVORA/qyvora-sekhmet/internal/baseline"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// buildBaseline runs a compact baseline profile against the runner. It is used
// to seed classification state before a campaign so fuzzing is never blind.
func buildBaseline(ctx context.Context, runner execution.Runner) (*models.Baseline, error) {
	return baseline.Profile(ctx, runner, &baseline.Options{
		Warmup:  2,
		Samples: 8,
	})
}

func readSeedFile(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > maxSeedFileSize {
		return nil, fmt.Errorf("seed file %s is %d bytes; exceeds the %d-byte cap",
			path, fi.Size(), maxSeedFileSize)
	}
	return os.ReadFile(path)
}

const maxSeedFileSize = 64 << 20

// targetsDir returns the directory used to persist target definitions. It is
// derived from session.dir so target state and session state live together.
func targetsDir(v interface {
	GetString(string) string
}) string {
	base := v.GetString("session.dir")
	if base == "" {
		base = "sessions"
	}
	return filepath.Join(base, "targets")
}
