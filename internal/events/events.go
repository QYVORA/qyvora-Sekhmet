// Package events implements the shared QYVORA JSONL event envelope used by
// anansi, toha3ee, jabari, aksum, nzinga and shaka. Every emitted line is one
// JSON object:
//
//	{"schema_version":"1.0","timestamp":"...","execution_id":"...",
//	 "framework":"sekhmet","level":"info","event":"seed.executed","data":{}}
//
// Consumers key on event names, never on terminal output. See the QYVORA
// output spec for the frozen contract.
//
// sekhmet emits the common lifecycle verbs (scan.started/completed,
// campaign.*) plus a fuzzing-specific vocabulary (target.discovered,
// baseline.completed, seed.executed, coverage.updated, behavior.discovered,
// crash.detected, hang.detected, anomaly.detected, input.minimized). New verb
// additions are backward compatible; the envelope fields themselves are
// frozen at schema version 1.0.
package events

import (
	"io"

	shared "github.com/QYVORA/qyvora-events"
)

// SchemaVersion is the event schema version every emitted event carries.
const SchemaVersion = shared.SchemaVersion

// Shared, ecosystem-wide verbs plus sekhmet's fuzzing-specific verbs with the
// "category.thing.state" shape so consumers can route on prefix without a
// lookup table.
const (
	ScanStarted       = "scan.started"
	ScanCompleted     = "scan.completed"
	CampaignStarted   = "campaign.started"
	CampaignCompleted = "campaign.completed"
	StageStarted      = "stage.started"
	StageCompleted    = "stage.completed"
	FindingDiscovered = "finding.discovered"
	EvidenceCollected = "evidence.collected"
	ReportGenerated   = "report.generated"
	Warning           = "warning"
	Error             = "error"

	TargetDiscovered   = "target.discovered"
	TargetValidated    = "target.validated"
	BaselineStarted    = "baseline.started"
	BaselineCompleted  = "baseline.completed"
	SeedCreated        = "seed.created"
	SeedExecuted       = "seed.executed"
	CoverageUpdated    = "coverage.updated"
	BehaviorDiscovered = "behavior.discovered"
	CrashDetected      = "crash.detected"
	HangDetected       = "hang.detected"
	AnomalyDetected    = "anomaly.detected"
	InputMinimized     = "input.minimized"
	CorpusAdded        = "corpus.added"
	CorpusTrimmed      = "corpus.trimmed"
	SessionStopped     = "session.stopped"
)

// Level values for the envelope's level field.
const (
	LevelInfo    = shared.LevelInfo
	LevelWarning = shared.LevelWarning
	LevelError   = shared.LevelError
)

type Event = shared.Event
type Stream = shared.Stream

// NewStream returns a stream bound to sekhmet.
func NewStream(w io.Writer) *Stream { return shared.NewStream(w, "sekhmet") }
