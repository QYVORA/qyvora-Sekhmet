package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Seed is a single corpus entry with rich metadata. The corpus uses this
// metadata to prioritize, schedule and never randomly execute seeds.
type Seed struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"` // imported, generated, mutation, seed
	Size      int       `json:"size"`
	TargetID  string    `json:"target_id"`
	CreatedBy string    `json:"creation_method,omitempty"`
	ParentID  string    `json:"parent_seed,omitempty"`
	Operator  string    `json:"mutation_operator,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Execution bookkeeping (updated by the engine, kept in memory on the
	// in-memory copy).
	Executions         int     `json:"executions"`
	CoverageNovelty    int     `json:"coverage_novelty"`
	Novelty            int     `json:"novelty"`
	Interestingness    float64 `json:"interestingness"`
	FailureAssociation string  `json:"failure_association,omitempty"`

	// Data is held outside the wire model so JSON serialization of the
	// metadata does not dump raw bytes; the corpus stores content on disk.
	Data []byte `json:"-"`
}

// NewSeedID returns a crypto-random seed identifier.
func NewSeedID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "seed-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return "seed-" + hex.EncodeToString(b)
}

// Priority ranks a seed for scheduling. Higher rank = execute sooner.
func (s *Seed) Priority(weightNovelty float64) float64 {
	base := float64(s.Novelty)*weightNovelty + s.Interestingness
	// Prefer small inputs that reach the same behavior to keep the hot path
	// cheap: a size cost term rewards small seeds.
	if s.Size > 0 {
		base += 64.0 / float64(s.Size)
	}
	return base
}
