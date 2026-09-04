package feedback

import (
	"testing"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func TestBehaviorSignatureStableAcrossIdenticalRuns(t *testing.T) {
	a := BehaviorSignature(&models.ExecutionResult{
		ExitCode: 0, ExitClass: models.ClassNormalSuccess, Stdout: []byte("output A"), StdoutSize: 8,
	})
	b := BehaviorSignature(&models.ExecutionResult{
		ExitCode: 0, ExitClass: models.ClassNormalSuccess, Stdout: []byte("output A"), StdoutSize: 8,
	})
	if len(a) == 0 {
		t.Fatal("expected non-empty signature")
	}
	if !equalStrings(a, b) {
		t.Fatalf("expected identical signatures, got %v vs %v", a, b)
	}
}

func TestBehaviorSignatureDistinguishesClasses(t *testing.T) {
	normal := BehaviorSignature(&models.ExecutionResult{ExitClass: models.ClassNormalSuccess})
	crash := BehaviorSignature(&models.ExecutionResult{ExitClass: models.ClassCrash, Signal: "SIGSEGV"})
	if equalStrings(normal, crash) {
		t.Fatal("expected distinct signatures for different exit classes")
	}
}

func TestTrackerNoveltyAgainstBaseline(t *testing.T) {
	// Baseline already contains "edge-a": it must not count as novel.
	tr := NewTracker(KindBehavioral, []string{"edge-a"})
	if got := tr.Observe([]string{"edge-a"}); got != 0 {
		t.Fatalf("baseline feature counted as novel: %d", got)
	}
	if got := tr.Observe([]string{"edge-b", "edge-c"}); got != 2 {
		t.Fatalf("expected 2 novel features, got %d", got)
	}
	// Re-observing the same features must not add more.
	if got := tr.Observe([]string{"edge-b"}); got != 0 {
		t.Fatalf("duplicate feature counted as novel: %d", got)
	}
	if tr.NewCount() != 2 {
		t.Fatalf("expected NewCount 2, got %d", tr.NewCount())
	}
}

func TestTrackerNoneKindNeverNovel(t *testing.T) {
	tr := NewTracker(KindNone, nil)
	if got := tr.Observe([]string{"x"}); got != 0 {
		t.Fatalf("none kind should not report novelty, got %d", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
