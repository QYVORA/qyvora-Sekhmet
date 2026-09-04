// Package models defines the core structured data model shared across the
// sekhmet framework. The model package is the source of truth: reports render
// it directly, never a parallel hand-maintained representation. This keeps
// machine-readable output honest and deterministic.
package models

import "strings"

// Severity is a closed vocabulary matching the QYVORA ecosystem (lowercase).
type Severity string

const (
	SeverityCritical      Severity = "critical"
	SeverityHigh          Severity = "high"
	SeverityMedium        Severity = "medium"
	SeverityLow           Severity = "low"
	SeverityInformational Severity = "informational"
)

// ParseSeverity normalizes an arbitrary string to a valid Severity.
func ParseSeverity(s string) Severity {
	norm := Severity(strings.ToLower(s))
	switch norm {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInformational:
		return norm
	default:
		return SeverityInformational
	}
}

// Weights returns a numeric rank used for ordering and summary. informational
// is rank 0; critical is rank 4.
func (s Severity) Weights() int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// Valid reports whether s is a member of the closed severity set.
func (s Severity) Valid() bool {
	return s.Weights() != 0 || s == SeverityInformational
}
