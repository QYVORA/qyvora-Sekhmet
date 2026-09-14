package models

import (
	"time"
)

// ExecutionResult is the cheap observation produced by one hot-path execution.
// It is intentionally small: the hot path should not carry expensive analysis.
type ExecutionResult struct {
	ExitCode   int           `json:"exit_code"`
	Signal     string        `json:"signal,omitempty"` // e.g. SIGSEGV
	Stdout     []byte        `json:"-"`
	Stderr     []byte        `json:"-"`
	StdoutSize int           `json:"stdout_size"`
	StderrSize int           `json:"stderr_size"`
	Duration   time.Duration `json:"duration_ns"`
	TimedOut   bool          `json:"timed_out"`
	Killed     bool          `json:"killed"`
	Failed     bool          `json:"failed"` // launch/runtime failure, not target crash
	Error      string        `json:"error,omitempty"`
	// OutputTruncated marks captured stdio that was cut at the output cap.
	OutputTruncated bool `json:"output_truncated,omitempty"`

	// HTTP fields populated for http targets.
	StatusCode int                 `json:"status_code,omitempty"`
	Headers    map[string][]string `json:"-"` // large; held out of cheap JSON
	Body       []byte              `json:"-"`
	BodySize   int                 `json:"body_size,omitempty"`

	// Behavior signature derived from the observation (populated during
	// classification).
	ExitClass ExitClass `json:"exit_class,omitempty"`
}

// DirReport is returned by the deep-analysis path (stdout/stderr/body are
// heavy and should not be re-serialized on every hot-path execution).
func (r *ExecutionResult) DirReport() ExecutionDetail {
	return ExecutionDetail{
		ExitCode: r.ExitCode,
		Signal:   r.Signal,
		Stdout:   r.Stdout,
		Stderr:   r.Stderr,
		Body:     r.Body,
		Duration: r.Duration,
		TimedOut: r.TimedOut,
	}
}

// ExecutionDetail is the full observation record used by the deep analysis
// path: crash analysis, deduplication, minimization and evidence generation.
type ExecutionDetail struct {
	ExitCode int                 `json:"exit_code"`
	Signal   string              `json:"signal,omitempty"`
	Stdout   []byte              `json:"stdout,omitempty"`
	Stderr   []byte              `json:"stderr,omitempty"`
	Body     []byte              `json:"body,omitempty"`
	Headers  map[string][]string `json:"headers,omitempty"`
	Duration time.Duration       `json:"duration_ns"`
	TimedOut bool                `json:"timed_out"`
}
