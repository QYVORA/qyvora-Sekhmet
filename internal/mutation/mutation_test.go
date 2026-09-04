package mutation

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestMutateChangesInput(t *testing.T) {
	e := New(Config{})
	data := []byte("the quick brown fox jumps over the lazy dog 0123456789")
	for i := 0; i < 200; i++ {
		_, out := e.Mutate(data)
		if len(out) == 0 {
			t.Fatalf("iteration %d: empty output", i)
		}
	}
	// Ensure at least some outputs differ from the original (a real mutation
	// engine should rarely leave the input untouched).
	changed := 0
	for i := 0; i < 500; i++ {
		_, out := e.Mutate(data)
		if !bytes.Equal(out, data) {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("expected at least some mutated outputs to differ from input")
	}
}

func TestMutateEmptyInserts(t *testing.T) {
	e := New(Config{})
	for i := 0; i < 50; i++ {
		op, out := e.Mutate(nil)
		switch op {
		case OpBlockInsert, OpByteInsert, OpBlockDuplicate, OpDictionary:
		default:
			// non-insert operators fall back to a single byte
		}
		if len(out) == 0 {
			t.Fatalf("empty input produced empty output for op %s", op)
		}
	}
}

func TestDeterministicRNGReproducible(t *testing.T) {
	var first []Operator
	for round := 0; round < 2; round++ {
		e := New(Config{RNG: rand.New(rand.NewSource(1234))})
		var ops []Operator
		const n = 40
		for i := 0; i < n; i++ {
			op, _ := e.Mutate([]byte("seed-data"))
			ops = append(ops, op)
		}
		if round == 0 {
			first = ops
		} else if !equalOps(first, ops) {
			t.Fatalf("deterministic run diverged")
		}
	}
}

func TestRecordAdaptsWeights(t *testing.T) {
	e := New(Config{})
	// Boost a specific operator with a positive signal.
	e.Record(OpByteFlip, true, true, true, false, false, false)
	if e.weights[OpByteFlip] <= 1 {
		t.Fatalf("expected byte_flip weight to be boosted, got %d", e.weights[OpByteFlip])
	}
}

func equalOps(a, b []Operator) bool {
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
