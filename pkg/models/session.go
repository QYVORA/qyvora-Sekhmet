package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// SessionState is the coarse lifecycle of a fuzzing campaign session.
type SessionState string

const (
	SessionRunning  SessionState = "running"
	SessionPaused   SessionState = "paused"
	SessionStopped  SessionState = "stopped"
	SessionFinished SessionState = "finished"
)

// SessionResult is a summary metric snapshot of a campaign. It is the
// machine-readable contract the CLI HUD and reports both render from.
type SessionResult struct {
	SessionID      string       `json:"session_id"`
	TargetID       string       `json:"target_id,omitempty"`
	Profile        string       `json:"profile,omitempty"`
	Start          time.Time    `json:"start"`
	End            time.Time    `json:"end,omitempty"`
	Runtime        string       `json:"runtime,omitempty"`
	Executions     int64        `json:"executions"`
	ExecPerSec     float64      `json:"exec_per_sec"`
	CorpusSize     int          `json:"corpus_size"`
	CoverageGained int          `json:"coverage_gained"`
	NewBehaviors   int          `json:"new_behaviors"`
	UniqueCrashes  int          `json:"unique_crashes"`
	Hangs          int          `json:"hangs"`
	Anomalies      int          `json:"anomalies"`
	Timeouts       int          `json:"timeouts"`
	CurrentSeed    string       `json:"current_seed,omitempty"`
	Strategy       string       `json:"strategy,omitempty"`
	Workers        int          `json:"workers"`
	State          SessionState `json:"state"`
}

// Session is the persisted, reproducible record of a fuzzing campaign.
type Session struct {
	ID        string         `json:"id"`
	TargetID  string         `json:"target_id"`
	Profile   string         `json:"profile"`
	Start     time.Time      `json:"start"`
	End       time.Time      `json:"end,omitempty"`
	State     SessionState   `json:"state"`
	Findings  []*Finding     `json:"findings,omitempty"`
	Crashes   []*Finding     `json:"crashes,omitempty"`
	Result    *SessionResult `json:"result,omitempty"`
	Errors    []string       `json:"errors,omitempty"`
	OutputDir string         `json:"output_dir,omitempty"`
}

// NewSessionID returns a crypto-random session identifier.
func NewSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "sess-" + hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return "sess-" + hex.EncodeToString(b)
}

// AddFinding adds a finding, deduplicating by fingerprint and merging
// evidence upward. This is the shared QYVORA dedup contract.
func (s *Session) AddFinding(f *Finding) {
	f.SessionID = s.ID
	fp := f.Fingerprint()
	for _, existing := range s.Findings {
		if existing.Fingerprint() == fp {
			mergeEvidence(existing, f)
			return
		}
	}
	s.Findings = append(s.Findings, f)
}

// AddCrash records a crash finding, deduplicated against the crash list.
func (s *Session) AddCrash(f *Finding) {
	f.SessionID = s.ID
	fp := f.Fingerprint()
	for _, existing := range s.Crashes {
		if existing.Fingerprint() == fp {
			mergeEvidence(existing, f)
			return
		}
	}
	s.Crashes = append(s.Crashes, f)
}

// Finish marks a session complete.
func (s *Session) Finish() {
	s.End = time.Now().UTC()
	s.State = SessionFinished
}

func mergeEvidence(dst, src *Finding) {
	seen := make(map[string]bool, len(dst.Evidence))
	for _, ev := range dst.Evidence {
		seen[ev.Hash] = true
	}
	for _, ev := range src.Evidence {
		if ev.Hash == "" {
			ev.Hash = HashContent(ev.Data)
		}
		if !seen[ev.Hash] {
			dst.Evidence = append(dst.Evidence, ev)
			seen[ev.Hash] = true
		}
	}
	dst.Confidence = mergeConfidence(dst.Confidence, src.Confidence)
}
