package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// EvidenceKind is the closed set of evidence artifact types sekhmet preserves.
type EvidenceKind string

const (
	KindTriggeringInput EvidenceKind = "triggering_input"
	KindMinimizedInput  EvidenceKind = "minimized_input"
	KindStdout          EvidenceKind = "stdout"
	KindStderr          EvidenceKind = "stderr"
	KindExitCode        EvidenceKind = "exit_code"
	KindSignal          EvidenceKind = "signal"
	KindTiming          EvidenceKind = "timing"
	KindCoverage        EvidenceKind = "coverage"
	KindSanitizer       EvidenceKind = "sanitizer"
	KindCrashSignature  EvidenceKind = "crash_signature"
	KindBaselineCompare EvidenceKind = "baseline_comparison"
	KindHTTPResponse    EvidenceKind = "http_response"
	KindEnvironment     EvidenceKind = "environment"
	KindReproduction    EvidenceKind = "reproduction"
)

// Evidence is a reproducible artifact that backs a finding. Triggering and
// minimized inputs are preserved as content (base64/hash) plus a reference so
// results can be replayed without guessing.
type Evidence struct {
	ID        string       `json:"id"`
	Kind      EvidenceKind `json:"kind"`
	Source    string       `json:"source"`
	Content   string       `json:"content,omitempty"`
	Data      []byte       `json:"-"`
	MimeType  string       `json:"mime_type,omitempty"`
	Hash      string       `json:"hash"`
	Reference string       `json:"reference,omitempty"` // on-disk artifact path
	Timestamp time.Time    `json:"timestamp"`
}

// EvidenceID returns a sequence-stable identifier for an evidence record.
func EvidenceID(prefix string, n int) string {
	return fmt.Sprintf("%s-%d", prefix, n)
}

// HashContent returns the hex SHA-256 of arbitrary bytes.
func HashContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Fingerprint returns a content-derived identifier for an input-based evidence
// record so that identical artifacts dedupe regardless of provenance.
func (e *Evidence) Fingerprint() string {
	var b strings.Builder
	b.WriteString(string(e.Kind))
	b.WriteString("\x00")
	if e.Hash != "" {
		b.WriteString(e.Hash)
	} else {
		b.WriteString(HashContent(e.Data))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
