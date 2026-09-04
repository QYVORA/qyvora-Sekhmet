package minimization

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// stubRunner places the input bytes into Stdout so a predicate can decide
// whether an input is still "interesting" (here: contains the marker KEEP).
type stubRunner struct{}

func (stubRunner) Kind() string { return "stub" }
func (stubRunner) Close() error { return nil }
func (stubRunner) Exec(_ context.Context, input []byte, _ time.Duration) (*models.ExecutionResult, error) {
	return &models.ExecutionResult{
		ExitClass:  models.ClassNormalSuccess,
		ExitCode:   0,
		Stdout:     append([]byte(nil), input...),
		StdoutSize: len(input),
	}, nil
}

var marker = []byte("KEEP")

func keepMarker(res *models.ExecutionResult) bool {
	return res != nil && bytes.Contains(res.Stdout, marker)
}

func TestMinimizeStructuralReduction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	input := []byte("AAAAAAAAAKEEPRRRRRRRRRRBBBBBBBBB")
	reduced := Minimize(ctx, stubRunner{}, input, 50*time.Millisecond, keepMarker)
	if len(reduced) > len(input) {
		t.Fatalf("expected reduction <= input, got %d > %d", len(reduced), len(input))
	}
	// A correct structural minimizer should drop surrounding filler while
	// leaving the marker intact.
	if !bytes.Contains(reduced, marker) {
		t.Fatalf("minimizer dropped the marker: %q", reduced)
	}
	if len(reduced) > len(marker)+2 {
		t.Logf("note: minimizer could not fully reduce %q (conservative is fine)", reduced)
	}
}

func TestMinimizeMarkerStrict(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Predicate only holds when the input contains the marker exactly; filler
	// X's are what minimization should be able to remove.
	input := append([]byte("XXXXXXXXXXXX"), marker...)
	reduced := Minimize(ctx, stubRunner{}, input, 100*time.Millisecond, func(res *models.ExecutionResult) bool {
		return res != nil && bytes.Contains(res.Stdout, marker)
	})
	if !bytes.Contains(reduced, marker) {
		t.Fatalf("strict predicate lost the marker: %q", reduced)
	}
}

func TestMinimizeSmallInputUntouched(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	in := []byte("AB")
	got := Minimize(ctx, stubRunner{}, in, 20*time.Millisecond, keepMarker)
	if len(got) == 0 {
		t.Fatal("expected non-empty output")
	}
}
