package models

// Confidence is a closed vocabulary matching the QYVORA ecosystem. It is
// independent of severity: a fuzz finding can be critical in impact yet only
// low confidence until reproduced or minimized.
type Confidence string

const (
	ConfidenceConfirmed Confidence = "confirmed"
	ConfidenceHigh      Confidence = "high"
	ConfidenceMedium    Confidence = "medium"
	ConfidenceLow       Confidence = "low"
)

// ParseConfidence normalizes an arbitrary string to a valid Confidence.
func ParseConfidence(s string) Confidence {
	switch Confidence(s) {
	case ConfidenceConfirmed, ConfidenceHigh, ConfidenceMedium, ConfidenceLow:
		return Confidence(s)
	default:
		return ConfidenceLow
	}
}

// Valid reports whether c is a member of the closed confidence set.
func (c Confidence) Valid() bool {
	return c == ConfidenceConfirmed || c == ConfidenceHigh ||
		c == ConfidenceMedium || c == ConfidenceLow
}

// Rank returns a numeric ordering used to merge confidence upward only:
// confirmed > high > medium > low.
func (c Confidence) Rank() int {
	switch c {
	case ConfidenceConfirmed:
		return 4
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	default:
		return 0
	}
}
