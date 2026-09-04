package models

import (
	"time"
)

// TargetType is the closed set of fuzzable target classes that sekhmet can
// reason about. Not every type is fully implemented in every build; the
// capability system reports what a specific build supports honestly.
type TargetType string

const (
	TargetAuto       TargetType = "auto"
	TargetProcess    TargetType = "process"
	TargetFileFormat TargetType = "file"
	TargetHTTP       TargetType = "http"
	TargetCLI        TargetType = "cli"
	TargetNetwork    TargetType = "network"
	TargetSimulation TargetType = "simulation"
)

// ParseTargetType normalizes an arbitrary string to a known TargetType.
// Unknown/empty values normalize to TargetAuto ("auto") which defers to
// automated detection.
func ParseTargetType(s string) TargetType {
	switch TargetType(s) {
	case TargetProcess, TargetFileFormat, TargetHTTP, TargetCLI,
		TargetNetwork, TargetSimulation:
		return TargetType(s)
	default:
		return TargetAuto
	}
}

// Authorization records the consent granting an assessment of a target.
type Authorization struct {
	Granted   bool      `json:"granted"`
	GrantedAt time.Time `json:"granted_at,omitempty"`
	Scope     string    `json:"scope,omitempty"`
	GrantedBy string    `json:"granted_by,omitempty"`
	Method    string    `json:"method,omitempty"`
}

// Target identifies and describes the system being fuzzed. It is deliberately
// transport-agnostic at the model layer: adapters (process, http, file, ...)
// interpret these fields.
type Target struct {
	ID        string        `json:"id"`
	Name      string        `json:"name,omitempty"`
	Type      TargetType    `json:"type"`
	Path      string        `json:"path,omitempty"`     // executable or file path
	Endpoint  string        `json:"endpoint,omitempty"` // http URL or host:port
	Args      []string      `json:"args,omitempty"`     // argv template for cli/process
	InputFile string        `json:"input_file,omitempty"`
	Auth      Authorization `json:"authorization"`
	CreatedAt time.Time     `json:"created_at"`
	Profile   string        `json:"profile,omitempty"`
	Sim       bool          `json:"sim,omitempty"`
}

// Authorized reports whether explicit authorization has been granted.
func (t *Target) Authorized() bool {
	return t != nil && t.Auth.Granted
}

// DisplayName returns the best human-readable label for the target.
func (t *Target) DisplayName() string {
	if t == nil {
		return "<nil>"
	}
	switch {
	case t.Name != "":
		return t.Name
	case t.Path != "":
		return t.Path
	case t.Endpoint != "":
		return t.Endpoint
	case t.Type != "":
		return string(t.Type)
	default:
		return "unknown target"
	}
}
