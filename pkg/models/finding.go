package models

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// FindingStatus is the lifecycle state of a finding, following the shared
// QYVORA progression where relevant.
type FindingStatus string

const (
	StatusSuspected     FindingStatus = "suspected"
	StatusValidated     FindingStatus = "validated"
	StatusExploitable   FindingStatus = "exploitable"
	StatusExploited     FindingStatus = "exploited"
	StatusFalsePositive FindingStatus = "false-positive"
	StatusInformational FindingStatus = "informational"
)

// Tier represents the capability tier that generated a finding
type Tier string

const (
	TierRecon        Tier = "recon"        // Tier 1: Discovery and enumeration
	TierTechnique    Tier = "technique"    // Tier 2: Vulnerability identification
	TierExploitation Tier = "exploitation" // Tier 3: Active exploitation
)

// ParseFindingStatus normalizes an arbitrary string to a known status.
func ParseFindingStatus(s string) FindingStatus {
	switch FindingStatus(s) {
	case StatusSuspected, StatusValidated, StatusExploitable, StatusExploited,
		StatusFalsePositive, StatusInformational:
		return FindingStatus(s)
	default:
		return StatusSuspected
	}
}

// FindingClassification distinguishes honest reporting categories. sekhmet
// never upgrades an observation to a vulnerability claim without justification.
type FindingClassification string

const (
	ClassificationCrashDetected          FindingClassification = "crash_detected"
	ClassificationPotentialVulnerability FindingClassification = "potential_vulnerability"
	ClassificationConfirmedVulnerability FindingClassification = "confirmed_vulnerability"
	ClassificationRobustnessIssue        FindingClassification = "robustness_issue"
	ClassificationPerformanceIssue       FindingClassification = "performance_issue"
	ClassificationHang                   FindingClassification = "hang"
	ClassificationBehaviorAnomaly        FindingClassification = "behavior_anomaly"
)

// Finding is a structured, evidence-backed result from a fuzzing campaign.
// It carries the shared QYVORA finding fields (snake_case JSON) plus
// fuzzing-specific evidence: the triggering input reference, crash signature,
// and coverage information.
type Finding struct {
	ID             string                `json:"id"`
	TargetID       string                `json:"target_id"`
	SessionID      string                `json:"session_id"`
	RuleID         string                `json:"rule_id,omitempty"`
	Title          string                `json:"title"`
	Category       string                `json:"category"`
	Classification FindingClassification `json:"classification"`
	Description    string                `json:"description"`
	Impact         string                `json:"impact,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
	Severity       Severity              `json:"severity"`
	Confidence     Confidence            `json:"confidence"`
	Status         FindingStatus         `json:"status"`
	Tier           Tier                  `json:"tier,omitempty"` // Capability tier that generated this finding
	InputRef       string                `json:"input_ref,omitempty"`
	CrashSign      string                `json:"crash_signature,omitempty"`
	Coverage       string                `json:"coverage,omitempty"`
	Evidence       []Evidence            `json:"evidence,omitempty"`
	Attributes     map[string]string     `json:"attributes,omitempty"`
	References     []string              `json:"references,omitempty"`
	Timestamp      time.Time             `json:"timestamp"`
}

// TierPrefix returns a display prefix for the finding's tier
func (f *Finding) TierPrefix() string {
	switch f.Tier {
	case TierRecon:
		return "[RECON]"
	case TierTechnique:
		return "[TECHNIQUE]"
	case TierExploitation:
		return "[EXPLOIT]"
	default:
		return ""
	}
}

// AddEvidence appends an evidence record, deduplicating by content hash.
func (f *Finding) AddEvidence(ev Evidence) *Finding {
	seen := make(map[string]bool, len(f.Evidence))
	for _, e := range f.Evidence {
		seen[e.Hash] = true
	}
	if !seen[ev.Hash] {
		f.Evidence = append(f.Evidence, ev)
	}
	return f
}

// Fingerprint returns a deterministic identifier for deduplication. Two
// findings with the same rule, category, classification, title, target and
// attributes are considered duplicates; evidence merges into the survivor.
func (f *Finding) Fingerprint() string {
	var b strings.Builder
	b.WriteString(f.RuleID)
	b.WriteString("\x00")
	b.WriteString(f.Category)
	b.WriteString("\x00")
	b.WriteString(string(f.Classification))
	b.WriteString("\x00")
	b.WriteString(f.Title)
	b.WriteString("\x00")
	b.WriteString(f.TargetID)
	keys := make([]string, 0, len(f.Attributes))
	for k := range f.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("\x00")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(f.Attributes[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// SeverityCounts summarizes findings by closed severity set.
type SeverityCounts struct {
	Critical      int `json:"critical"`
	High          int `json:"high"`
	Medium        int `json:"medium"`
	Low           int `json:"low"`
	Informational int `json:"informational"`
}

// Tally returns the total number of findings represented.
func (c SeverityCounts) Tally() int {
	return c.Critical + c.High + c.Medium + c.Low + c.Informational
}

// mergeConfidence upgrades dst only upward (confirmed > high > medium > low).
func mergeConfidence(dst, src Confidence) Confidence {
	if src.Rank() > dst.Rank() {
		return src
	}
	return dst
}
