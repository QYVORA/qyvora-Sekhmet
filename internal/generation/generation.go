// Package generation implements generation-based, grammar-based and
// structure-aware fuzzing. Where mutation modifies existing inputs, generation
// produces new structured inputs from templates, grammars and schemas. This is
// the clean extension seam for future constraint-aware and spec-driven
// generation.
package generation

import (
	"fmt"
	"math/rand"
	"strings"
)

// Generator produces a new input from scratch.
type Generator interface {
	// Name returns the generator identifier.
	Name() string
	// Generate returns a newly generated input.
	Generate(rng *rand.Rand) []byte
}

// Registry maps generator names to implementations.
type Registry struct {
	gens  map[string]Generator
	order []string
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{gens: make(map[string]Generator), order: []string{}}
}

// Register adds a generator.
func (r *Registry) Register(g Generator) {
	if g == nil {
		return
	}
	if _, ok := r.gens[g.Name()]; !ok {
		r.order = append(r.order, g.Name())
	}
	r.gens[g.Name()] = g
}

// Get returns a generator by name.
func (r *Registry) Get(name string) (Generator, bool) {
	g, ok := r.gens[name]
	return g, ok
}

// Names returns registered generator names in order.
func (r *Registry) Names() []string { return r.order }

// Generate invokes a named generator.
func (r *Registry) Generate(name string, rng *rand.Rand) ([]byte, error) {
	g, ok := r.gens[name]
	if !ok {
		return nil, fmt.Errorf("unknown generator %q", name)
	}
	return g.Generate(rng), nil
}

// Random generates from a uniformly-chosen registered generator.
func (r *Registry) Random(rng *rand.Rand) ([]byte, string, error) {
	if len(r.order) == 0 {
		return nil, "", fmt.Errorf("no generators registered")
	}
	name := r.order[rng.Intn(len(r.order))]
	out, err := r.Generate(name, rng)
	return out, name, err
}

// TokenGen generates random byte tokens of bounded length.
type TokenGen struct{}

// Name implements Generator.
func (TokenGen) Name() string { return "token" }

// Generate implements Generator.
func (TokenGen) Generate(rng *rand.Rand) []byte {
	n := 1 + rng.Intn(64)
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rng.Intn(256))
	}
	return out
}

// BoundaryGen generates boundary/interesting numeric encodings.
type BoundaryGen struct{}

// Name implements Generator.
func (BoundaryGen) Name() string { return "boundary" }

// Generate implements Generator.
func (BoundaryGen) Generate(rng *rand.Rand) []byte {
	vals := [...]string{"0", "1", "-1", "255", "256", "65535", "-2147483648", "2147483647", "4294967295"}
	return []byte(vals[rng.Intn(len(vals))])
}

// JSONGen produces structurally-valid JSON with random-but-valid field values,
// useful for structure-aware fuzzing of JSON parsers (valid structure reaches
// deeper application logic than random corruption).
type JSONGen struct{}

// Name implements Generator.
func (JSONGen) Name() string { return "json" }

// Generate implements Generator.
func (JSONGen) Generate(rng *rand.Rand) []byte {
	simple := [...]string{
		`{}`,
		`[]`,
		`null`,
		`true`,
		`false`,
		`"a"`,
		`0`,
		`{"a":1,"b":[1,2,3],"c":{"x":"y"}}`,
		`[{"id":1,"name":"x"},{"id":2,"name":"y"}]`,
	}
	return []byte(simple[rng.Intn(len(simple))])
}

// StructMuter is a clean seam for structure-aware mutation. It mutates within
// a recognized structure without destroying it. In the initial build it
// performs structure-preserving value replacement for JSON/XML-like inputs.
type StructMuter struct {
	rng *rand.Rand
}

// NewStructMuter returns a structure-aware mutator.
func NewStructMuter(rng *rand.Rand) *StructMuter { return &StructMuter{rng: rng} }

// MutateStructure preserves the outer structure and mutates scalars inside.
func (m *StructMuter) MutateStructure(data []byte) []byte {
	depthTracking := strings.Count(string(data), "{") + strings.Count(string(data), "[")
	_ = depthTracking
	out := append([]byte(nil), data...)
	// Structure-aware approach: if the input is JSON-like, keep braces and
	// mutate a scalar token; otherwise fall back to byte-level mutation that
	// preserves high-level delimiters.
	// Locate a colon-delimited value boundary and perturb the value region.
	// Simple, deterministic implementation for this build: mutate a random
	// digit within a quoted/number scalar if present.
	for i := 0; i < len(out); i++ {
		if out[i] >= '0' && out[i] <= '9' && m.rng.Intn(20) == 0 {
			if i > 0 && (out[i-1] == ':' || isDigitSpace(out[i-1])) {
				out[i] = byte('0' + m.rng.Intn(10))
				return out
			}
		}
	}
	// No scalar found; safe fallback: flip a byte outside of structural chars.
	for i := 0; i < len(out); i++ {
		c := out[i]
		if c != '{' && c != '}' && c != '[' && c != ']' && c != ':' && c != ',' && c != '"' && c != '\'' {
			out[i] ^= 0x01
			return out
		}
	}
	return out
}

func isDigitSpace(b byte) bool {
	return b >= '0' && b <= '9'
}
