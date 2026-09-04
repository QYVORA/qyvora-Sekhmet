// Package feedback implements the coverage/novelty feedback architecture. It
// supports behavioral feedback for black-box targets (where instrumentation is
// absent) and edge/block tracking for instrumented targets. Novelty scoring
// decides which inputs are "interesting enough" to retain in the corpus.
package feedback

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Kind identifies the feedback mode.
type Kind string

const (
	KindNone       Kind = "none"
	KindBehavioral Kind = "behavioral"
	KindEdges      Kind = "edges"
	KindBlocks     Kind = "blocks"
)

// Tracker accumulates observed coverage/behavior and reports novelty.
type Tracker struct {
	mu       sync.Mutex
	kind     Kind
	seen     map[string]bool
	baseline map[string]bool
	total    int
}

// NewTracker returns a tracker of the given kind. baselineSeen pre-adds the
// coverage/behavior observed during baseline profiling so those do not count
// as novel.
func NewTracker(kind Kind, baselineSeen []string) *Tracker {
	t := &Tracker{kind: kind, seen: make(map[string]bool), baseline: make(map[string]bool)}
	for _, s := range baselineSeen {
		t.baseline[s] = true
	}
	return t
}

// Observe records a raw feature (edge id, behavior label) seen in a single
// execution and returns the number of brand-new (non-baseline) features it
// contributed. Cheap enough for the hot path.
func (t *Tracker) Observe(features []string) int {
	if t == nil || t.kind == KindNone {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	newCount := 0
	for _, f := range features {
		if t.baseline[f] || t.seen[f] {
			continue
		}
		t.seen[f] = true
		t.total++
		newCount++
	}
	return newCount
}

// NewCount returns the number of novel features discovered so far.
func (t *Tracker) NewCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total
}

// Reset clears the non-baseline tracking (used between phases).
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seen = make(map[string]bool)
	t.total = 0
}

// BehaviorSignature derives a deterministic, normalized feature set from an
// execution result. For black-box targets this is based on exit class, exit
// code, output size bucket and a normalized stdout fingerprint — fields that
// vary meaningfully between behaviors but are invariant across identical
// runs.
func BehaviorSignature(res *models.ExecutionResult) []string {
	fields := []string{
		"exit=" + string(res.ExitClass),
	}
	if res.ExitCode != 0 {
		fields = append(fields, "code="+itoa(res.ExitCode))
	}
	if res.Signal != "" {
		fields = append(fields, "sig="+res.Signal)
	}
	if res.StatusCode != 0 {
		fields = append(fields, "status="+itoa(res.StatusCode))
	}
	fields = append(fields, "out="+bucket(res.StdoutSize))
	fields = append(fields, "err="+bucket(res.StderrSize))
	// Include a normalized content fingerprint so that genuinely different
	// outputs register as distinct behaviors, while identical outputs dedupe.
	fp := hashShort(normalizeOutput(res.Stdout))
	if fp != "" {
		fields = append(fields, "outfp="+fp)
	}
	sort.Strings(fields)
	return fields
}

func bucket(n int) string {
	switch {
	case n == 0:
		return "k0"
	case n < 64:
		return "k64"
	case n < 1024:
		return "k1k"
	case n < 65536:
		return "k64k"
	case n < 1<<20:
		return "m1"
	default:
		return "m1p"
	}
}

// normalizeOutput extracts a coarse stable fingerprint from output bytes so
// timestamps/request ids do not cause false "new behavior". It keeps only the
// first few normalized lines.
func normalizeOutput(b []byte) string {
	lines := splitLines(b)
	if len(lines) > 4 {
		lines = lines[:4]
	}
	return join(lines)
}

func splitLines(b []byte) []string {
	var out []string
	cur := ""
	for _, c := range b {
		if c == '\n' || c == '\r' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func join(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func hashShort(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
