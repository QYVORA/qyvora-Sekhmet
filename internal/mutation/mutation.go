// Package mutation implements the mutation engine: independent, testable
// mutation operators plus an adaptive operator selector that learns which
// strategies are productive for a given target. Deterministic operators use
// seeded randomness so reproducible campaigns are possible.
package mutation

import (
	"math/rand"
	"sync"
)

// Operator identifies a single mutation primitive. Each operator must be
// independently reproducible (given the same RNG state) and independently
// testable.
type Operator string

const (
	OpBitFlip           Operator = "bit_flip"
	OpByteFlip          Operator = "byte_flip"
	OpArithmetic        Operator = "arithmetic"
	OpIncDec            Operator = "inc_dec"
	OpInterestingValue  Operator = "interesting_value"
	OpBoundary          Operator = "boundary"
	OpBlockInsert       Operator = "block_insert"
	OpBlockDelete       Operator = "block_delete"
	OpBlockDuplicate    Operator = "block_duplicate"
	OpBlockReplace      Operator = "block_replace"
	OpByteInsert        Operator = "byte_insert"
	OpByteDelete        Operator = "byte_delete"
	OpChunkShuffle      Operator = "chunk_shuffle"
	OpSplice            Operator = "splice"
	OpLengthMutation    Operator = "length_mutation"
	OpDelimiterMutation Operator = "delimiter_mutation"
	OpDictionary        Operator = "dictionary"
)

// AllOperators is the stable enumeration in a deterministic order.
var AllOperators = []Operator{
	OpBitFlip, OpByteFlip, OpArithmetic, OpIncDec, OpInterestingValue,
	OpBoundary, OpBlockInsert, OpBlockDelete, OpBlockDuplicate,
	OpBlockReplace, OpByteInsert, OpByteDelete, OpChunkShuffle,
	OpSplice, OpLengthMutation, OpDelimiterMutation, OpDictionary,
}

// Interesting values for integers of various sizes.
var interestingInts = []uint64{
	0, 1, 2, 3, 4, 8, 16, 32, 64, 128, 255, 256, 512, 1024, 2048, 4096,
	0x7f, 0x80, 0xff, 0x100, 0x7fff, 0x8000, 0xffff, 0x10000,
	0x7fffffff, 0x80000000, 0xffffffff, 0x100000000,
	0x7fffffffffffffff, 0x8000000000000000, 0xffffffffffffffff,
	0x41414141, 0xdeadbeef, 0xcafebabe, 0xbeefcafe,
}

// Mutator applies a single operator to data and returns the result.
type Mutator interface {
	// Name returns the operator identifier.
	Name() Operator
	// Mutate applies the operator, returning a new byte slice.
	Mutate(data []byte) []byte
}

// Config groups RNG plus optional dictionary tokens for the engine.
type Config struct {
	RNG        *rand.Rand
	Dictionary [][]byte
}

// Engine selects and applies mutation operators, tracking per-operator
// effectiveness for adaptive scheduling.
type Engine struct {
	mu         sync.Mutex
	rng        *rand.Rand
	dictionary [][]byte
	stats      map[Operator]*Stat
	weights    map[Operator]int
	total      int
}

// Stat tracks adaptive bookkeeping for one operator.
type Stat struct {
	Operator    Operator `json:"operator"`
	Executions  int      `json:"executions"`
	Interesting int      `json:"interesting"`
	NewCoverage int      `json:"new_coverage"`
	NewBehavior int      `json:"new_behavior"`
	Crashes     int      `json:"crashes"`
	Timeouts    int      `json:"timeouts"`
	Anomalies   int      `json:"anomalies"`
}

// New returns an engine. If rng is nil a fresh one is created (non-seeded,
// non-deterministic); pass a seeded rand for reproducibility.
func New(cfg Config) *Engine {
	rng := cfg.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}
	e := &Engine{
		rng:        rng,
		dictionary: cfg.Dictionary,
		stats:      make(map[Operator]*Stat),
		weights:    make(map[Operator]int),
	}
	for _, op := range AllOperators {
		e.stats[op] = &Stat{Operator: op}
		e.weights[op] = 1
	}
	return e
}

// Mutate applies a single weighted-random operator, returning the op used and
// the mutated bytes. If data is empty only insert/duplicate operators apply.
// It is safe for concurrent use by multiple workers.
func (e *Engine) Mutate(data []byte) (Operator, []byte) {
	e.mu.Lock()
	defer e.mu.Unlock()
	op := e.pick()
	e.stats[op].Executions++
	e.total++
	return op, e.apply(op, data)
}

// MutateN applies n stacked mutations (splicing/compound mutations). Safe for
// concurrent use.
func (e *Engine) MutateN(data []byte, n int) []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := append([]byte(nil), data...)
	for i := 0; i < n; i++ {
		op := e.pick()
		e.stats[op].Executions++
		e.total++
		out = e.apply(op, out)
	}
	return out
}

// Record updates adaptive statistics for the last-used operator so the engine
// learns which mutations are productive. There is no false "learning": only
// observed outcomes move weights. Safe for concurrent use.
func (e *Engine) Record(op Operator, interesting, newCoverage, newBehavior, crash, timeout, anomaly bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.stats[op]
	if s == nil {
		return
	}
	if interesting {
		s.Interesting++
	}
	if newCoverage {
		s.NewCoverage++
	}
	if newBehavior {
		s.NewBehavior++
	}
	if crash {
		s.Crashes++
	}
	if timeout {
		s.Timeouts++
	}
	if anomaly {
		s.Anomalies++
	}
	e.adjust(op)
}

// Stats returns a copy of per-operator statistics for reporting.
func (e *Engine) Stats() []Stat {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Stat, 0, len(e.stats))
	for _, op := range AllOperators {
		s := e.stats[op]
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// adjust boosts operators that produce coverage/behavior, and slightly decays
// those that only burn executions. Bounded so any single operator can never
// fully monopolize the schedule.
func (e *Engine) adjust(op Operator) {
	s := e.stats[op]
	base := 1
	if s.NewCoverage > 0 || s.NewBehavior > 0 || s.Crashes > 0 {
		base = 3
	} else if s.Executions > 64 && s.Interesting == 0 && s.NewCoverage == 0 {
		base = 1 // no positive signal; keep at floor
	}
	w := base
	if s.Interesting > 0 {
		w += 2
	}
	if w > 10 {
		w = 10
	}
	e.weights[op] = w
}

func (e *Engine) pick() Operator {
	total := 0
	for _, w := range e.weights {
		total += w
	}
	if total <= 0 {
		return AllOperators[e.rng.Intn(len(AllOperators))]
	}
	r := e.rng.Intn(total)
	for _, op := range AllOperators {
		r -= e.weights[op]
		if r < 0 {
			return op
		}
	}
	return AllOperators[e.rng.Intn(len(AllOperators))]
}

func (e *Engine) apply(op Operator, data []byte) []byte {
	if len(data) == 0 {
		switch op {
		case OpBlockInsert, OpByteInsert, OpBlockDuplicate, OpDictionary:
			return e.insertEmpty(op)
		default:
			// Cannot mutate empty meaningfully; fall back to inserting a byte.
			return append([]byte{byte(e.rng.Intn(256))}, data...)
		}
	}
	switch op {
	case OpBitFlip:
		return bitFlip(data, e.rng)
	case OpByteFlip:
		return byteFlip(data, e.rng)
	case OpArithmetic:
		return arithmetic(data, e.rng)
	case OpIncDec:
		return incDec(data, e.rng)
	case OpInterestingValue:
		return interestingValue(data, e.rng)
	case OpBoundary:
		return boundary(data, e.rng)
	case OpBlockInsert:
		return blockInsert(data, e.rng)
	case OpBlockDelete:
		return blockDelete(data, e.rng)
	case OpBlockDuplicate:
		return blockDuplicate(data, e.rng)
	case OpBlockReplace:
		return blockReplace(data, e.rng)
	case OpByteInsert:
		return byteInsert(data, e.rng)
	case OpByteDelete:
		return byteDelete(data, e.rng)
	case OpChunkShuffle:
		return chunkShuffle(data, e.rng)
	case OpSplice:
		return splice(data, e)
	case OpLengthMutation:
		return lengthMutation(data, e.rng)
	case OpDelimiterMutation:
		return delimiterMutation(data, e.rng)
	case OpDictionary:
		return dictionaryInsert(data, e)
	default:
		return byteFlip(data, e.rng)
	}
}

func (e *Engine) insertEmpty(op Operator) []byte {
	switch op {
	case OpDictionary:
		return dictionaryInsert([]byte{}, e)
	default:
		return []byte{byte(e.rng.Intn(256))}
	}
}
