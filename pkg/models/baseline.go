package models

// Stat is a single statistical summary of a measured metric (e.g. execution
// time). It carries the full descriptive set so consumers can reason about
// variability without recomputing from raw samples.
type Stat struct {
	Count    int     `json:"count"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	Median   float64 `json:"median"` // p50
	P90      float64 `json:"p90"`
	P95      float64 `json:"p95"`
	P99      float64 `json:"p99"`
	Variance float64 `json:"variance"`
}

// ExitClass classifies a single process execution into a baseline-relative
// bucket. This is the conceptual backbone of "do not fuzz blindly": the
// baseline determines what is NORMAL before anomalies are reported.
type ExitClass string

const (
	ClassNormalSuccess   ExitClass = "normal_success"
	ClassNormalFailure   ExitClass = "normal_failure"
	ClassExpectedError   ExitClass = "expected_error"
	ClassUnexpected      ExitClass = "unexpected_behavior"
	ClassCrash           ExitClass = "crash"
	ClassHang            ExitClass = "hang"
	ClassResourceAnomaly ExitClass = "resource_anomaly"
)

// TimingProfile summarizes the observed duration behaviour of the target
// during baseline profiling and, later, the live campaign.
type TimingProfile struct {
	Normal    Stat    `json:"normal"`
	SlowFloor float64 `json:"slow_floor"` // p95 observed
	Timeout   float64 `json:"timeout_ms"` // adaptive timeout (ms)
	Warmup    int     `json:"warmup"`
	Samples   int     `json:"samples"`
}

// ResponseProfile describes the normal output/response surface of the target.
// For process targets this is stdout/stderr signatures; for HTTP targets it is
// status codes, header sets and body structure; for file targets it is output
// sizes.
type ResponseProfile struct {
	StdoutSize Stat     `json:"stdout_size"`
	StderrSize Stat     `json:"stderr_size"`
	ExitCodes  []int    `json:"exit_codes,omitempty"`
	Signals    []string `json:"signals,omitempty"`
	StatusCode []int    `json:"status_codes,omitempty"`
	BodySizes  Stat     `json:"body_sizes"`
	Headers    []string `json:"headers,omitempty"`
}

// Baseline is the complete structured description of normal target behaviour.
// The fuzzing engine compares live executions against it to classify
// anomalies instead of treating every nonzero exit as a vulnerability.
type Baseline struct {
	TargetID    string           `json:"target_id"`
	TargetType  TargetType       `json:"target_type"`
	ExitProfile *ResponseProfile `json:"exit_profile,omitempty"`
	Timing      *TimingProfile   `json:"timing,omitempty"`
	// DynamicFields lists field names the engine observed varying naturally
	// (request ids, timestamps, tokens) so it does not report them as
	// anomalies.
	DynamicFields []string        `json:"dynamic_fields,omitempty"`
	Behaviors     []BehaviorSig   `json:"behaviors,omitempty"`
	Coverage      CoverageProfile `json:"coverage,omitempty"`
}

// BehaviorSig is a normalized signature of one class of observable behavior.
type BehaviorSig struct {
	Name   string            `json:"name"`
	Label  string            `json:"label"`
	Normal bool              `json:"normal"` // whether baseline says this is expected
	Count  int               `json:"count"`
	Fields map[string]string `json:"fields,omitempty"`
}

// CoverageProfile describes the coverage state seen during baseline. It is
// intentionally generic so sekhmet works for black-box targets (behavioral
// feedback) as well as instrumented grey-box targets (edge coverage).
type CoverageProfile struct {
	Kind            string `json:"kind"` // "none" | "behavioral" | "edges" | "blocks"
	BaselineEdges   int    `json:"baseline_edges,omitempty"`
	TrackingEnabled bool   `json:"tracking_enabled"`
}
