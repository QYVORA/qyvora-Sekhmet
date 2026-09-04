package corpus

import (
	"testing"
)

func TestAddDeduplicatesByIdentity(t *testing.T) {
	s := New(t.TempDir())
	first, added1 := s.AddBytes([]byte("hello world"), "a")
	if !added1 {
		t.Fatal("expected first add to report added")
	}
	second, added2 := s.AddBytes([]byte("hello world"), "b")
	if added2 {
		t.Fatal("expected duplicate add to be rejected")
	}
	if first.ID != second.ID {
		t.Fatal("expected identical payload to map to same seed")
	}
	if s.Size() != 1 {
		t.Fatalf("expected size 1, got %d", s.Size())
	}
}

func TestAddDistinctPayloads(t *testing.T) {
	s := New(t.TempDir())
	s.AddBytes([]byte("aaa"), "a")
	s.AddBytes([]byte("bbb"), "b")
	if s.Size() != 2 {
		t.Fatalf("expected size 2, got %d", s.Size())
	}
}

func TestMarkExecutedTracksNovelty(t *testing.T) {
	s := New(t.TempDir())
	seed, _ := s.AddBytes([]byte("data"), "a")
	s.MarkExecuted(seed.ID, 5, 3, 0.5)
	got, _ := s.Get(seed.ID)
	if got.Executions != 1 {
		t.Fatalf("expected 1 execution, got %d", got.Executions)
	}
	if got.Novelty != 5 {
		t.Fatalf("expected novelty 5, got %d", got.Novelty)
	}
}

func TestTrimToKeepsBest(t *testing.T) {
	s := New(t.TempDir())
	high, _ := s.AddBytes([]byte("high-value"), "a")
	s.MarkExecuted(high.ID, 100, 100, 1.0)
	low, _ := s.AddBytes([]byte("low-value"), "b")
	s.MarkExecuted(low.ID, 1, 1, 0.0)
	removed := s.TrimTo(1)
	if s.Size() != 1 {
		t.Fatalf("expected 1 seed after trim, got %d", s.Size())
	}
	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}
}

func TestPrioritizedOrdersByPriority(t *testing.T) {
	s := New(t.TempDir())
	a, _ := s.AddBytes([]byte("a"), "x")
	b, _ := s.AddBytes([]byte("b"), "x")
	s.MarkExecuted(a.ID, 50, 50, 0.5)
	s.MarkExecuted(b.ID, 10, 10, 0.1)
	prio := s.Prioritized()
	if len(prio) != 2 {
		t.Fatalf("expected 2 seeds, got %d", len(prio))
	}
	if prio[0].Data[0] != 'a' {
		t.Fatalf("expected higher-priority seed first")
	}
}
