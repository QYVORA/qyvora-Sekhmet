// Package capabilities exposes sekhmet's capabilities as machine-readable tool
// metadata. Each major capability (target.discover, baseline.profile,
// campaign.fuzz, input.minimize, result.replay, crash.list, report.generate)
// is representable as a tool with input/output schema, risk, target type,
// authorization and confirmation metadata. The future QYVORA AI orchestrator
// lives above the frameworks and consumes this metadata; sekhmet itself does
// not embed an LLM.
package capabilities

import (
	"sort"
)

// RiskLevel is the closed safety risk set (S1..S4) shared with the ecosystem.
// S1 = reversible/read-only, S2 = needs confirmation, S3/S4 = higher risk.
type RiskLevel string

const (
	RiskS1 RiskLevel = "S1"
	RiskS2 RiskLevel = "S2"
	RiskS3 RiskLevel = "S3"
	RiskS4 RiskLevel = "S4"
)

// NoiseLevel defines the OPSEC footprint of an operation
type NoiseLevel string

const (
	NoiseLevelPassive    NoiseLevel = "passive"    // No active probing, analysis only
	NoiseLevelLow        NoiseLevel = "low"        // Minimal interaction, basic enumeration
	NoiseLevelModerate   NoiseLevel = "moderate"   // Active testing, noticeable
	NoiseLevelAggressive NoiseLevel = "aggressive" // Exploitation attempts, highly visible
)

// Tool is one machine-readable capability open to orchestration.
type Tool struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Framework    string     `json:"framework"`
	Category     string     `json:"category"`
	Output       []string   `json:"output_schema"`
	Risk         RiskLevel  `json:"risk"`
	NoiseLevel   NoiseLevel `json:"noise_level,omitempty"`
	AuthRequired bool       `json:"authorization_required"`
	TargetType   string     `json:"target_type"`
	Confirm      bool       `json:"confirmation_required"`
	Reversible   bool       `json:"reversible"`
	Duration     string     `json:"expected_duration,omitempty"`
}

// Catalog returns the sorted set of machine-readable capabilities.
func Catalog() []Tool {
	tools := []Tool{
		{
			ID: "sekhmet.target.discover", Name: "Discover and validate a target",
			Description: "Define and validate a fuzz target (process, file, http, cli, network, simulation).",
			Framework:   "sekhmet", Category: "targeting", Output: []string{"Target"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: true, TargetType: "any", Reversible: true,
		},
		{
			ID: "sekhmet.baseline.profile", Name: "Profile target baseline",
			Description: "Measure the target's normal behaviour (timing, response, exit profiles) before fuzzing.",
			Framework:   "sekhmet", Category: "baseline", Output: []string{"Baseline"},
			Risk: RiskS1, NoiseLevel: NoiseLevelLow, AuthRequired: true, TargetType: "any", Reversible: true,
		},
		{
			ID: "sekhmet.campaign.fuzz", Name: "Run a fuzzing campaign",
			Description: "Run a mutation/generation campaign against a target with adaptive scheduling.",
			Framework:   "sekhmet", Category: "fuzzing", Output: []string{"SessionResult", "Finding", "Crash"},
			Risk: RiskS2, NoiseLevel: NoiseLevelAggressive, AuthRequired: true, TargetType: "any", Reversible: true,
		},
		{
			ID: "sekhmet.corpus.manage", Name: "Manage the seed corpus",
			Description: "Import, export, deduplicate and prioritize corpus seeds.",
			Framework:   "sekhmet", Category: "corpus", Output: []string{"Seed"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: false, TargetType: "session", Reversible: true,
		},
		{
			ID: "sekhmet.input.minimize", Name: "Minimize an input",
			Description: "Reduce an interesting input while preserving triggering behaviour.",
			Framework:   "sekhmet", Category: "triage", Output: []string{"Evidence"},
			Risk: RiskS2, NoiseLevel: NoiseLevelModerate, AuthRequired: true, TargetType: "any", Reversible: true,
		},
		{
			ID: "sekhmet.result.replay", Name: "Replay an input",
			Description: "Reproduce an input against a target to confirm a result.",
			Framework:   "sekhmet", Category: "triage", Output: []string{"ExecutionResult"},
			Risk: RiskS2, NoiseLevel: NoiseLevelModerate, AuthRequired: true, TargetType: "any", Reversible: true,
		},
		{
			ID: "sekhmet.crash.list", Name: "List crashes",
			Description: "List deduplicated crashes from a session.",
			Framework:   "sekhmet", Category: "reporting", Output: []string{"Finding"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: false, TargetType: "session", Reversible: true,
		},
		{
			ID: "sekhmet.findings.list", Name: "List findings",
			Description: "List current campaign findings.",
			Framework:   "sekhmet", Category: "reporting", Output: []string{"Finding"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: false, TargetType: "session", Reversible: true,
		},
		{
			ID: "sekhmet.report.generate", Name: "Generate a report",
			Description: "Generate a campaign report in a chosen format.",
			Framework:   "sekhmet", Category: "reporting", Output: []string{"Report"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: false, TargetType: "session", Reversible: true,
		},
		{
			ID: "sekhmet.wordlists.discover", Name: "Discover SecLists wordlists",
			Description: "Locate and index a local SecLists installation for payload providers.",
			Framework:   "sekhmet", Category: "payload", Output: []string{"Wordlist"},
			Risk: RiskS1, NoiseLevel: NoiseLevelPassive, AuthRequired: false, TargetType: "session", Reversible: true,
		},
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].ID < tools[j].ID })
	return tools
}
