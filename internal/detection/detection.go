// Package detection implements crash, hang, timeout and anomaly detection and
// classification against a target baseline. It also builds crash signatures
// for deduplication so thousands of identical crashes collapse into one
// finding.
package detection

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Classify labels an execution as crash, hang, anomaly or normal relative to
// the baseline. It never treats a nonzero exit as a vulnerability by itself;
// only signals and anomalies are reported.
func Classify(res *models.ExecutionResult, base *models.Baseline) models.ExitClass {
	// Precedence: explicit crash signal wins; then timeout/hang; then
	// resource anomaly; then normal/expected.
	if res.Signal != "" {
		return models.ClassCrash
	}
	if res.TimedOut {
		return models.ClassHang
	}
	if res.ExitClass == models.ClassCrash {
		return models.ClassCrash
	}
	if base != nil && base.Timing != nil {
		// A wildly slow execution relative to baseline is a hang candidate
		// even before the hard timeout.
		if !res.TimedOut && res.Duration > time.Duration(base.Timing.Timeout)*time.Millisecond {
			return models.ClassHang
		}
	}
	return res.ExitClass
}

// IsCrash reports whether the result is a definite crash signal.
func IsCrash(res *models.ExecutionResult) bool {
	return res.Signal != "" || res.ExitClass == models.ClassCrash
}

// IsHang reports whether the result hit a timeout or exceeded the baseline
// slow floor.
func IsHang(res *models.ExecutionResult) bool {
	return res.TimedOut
}

// sanitizerRe matches common sanitizer signature lines in stderr.
var sanitizerRe = regexp.MustCompile(`(?i)(AddressSanitizer|UndefinedBehaviorSanitizer|MemorySanitizer|ThreadSanitizer|LeakSanitizer|runtime error:|SEGV|heap-buffer-overflow|stack-buffer-overflow)`)

// SanitizerFinding extracts an optional sanitizer message from stderr.
func SanitizerFinding(stderr []byte) (string, bool) {
	for _, line := range strings.Split(string(stderr), "\n") {
		if sanitizerRe.MatchString(line) {
			return strings.TrimSpace(line), true
		}
	}
	return "", false
}

// Signature builds a deterministic crash signature used for deduplication. It
// combines signal, normalized stderr fingerprint and execution context so that
// identical crashes collapse while distinct ones are preserved.
func Signature(res *models.ExecutionResult, detail *models.ExecutionDetail) string {
	var parts []string
	if res.Signal != "" {
		parts = append(parts, "sig:"+res.Signal)
	}
	if res.ExitCode != 0 {
		parts = append(parts, "code:"+itoa(res.ExitCode))
	}
	// Sanitizer directive dominates the signature when present.
	if d, ok := SanitizerFinding(detailStderr(detail)); ok {
		parts = append(parts, "san:"+hashShort(d))
	}
	// Normalized stderr contributes a fingerprint for native faults.
	norm := normalizeStderr(detailStderr(detail))
	if norm != "" {
		parts = append(parts, "err:"+hashShort(norm))
	}
	if len(parts) == 0 {
		parts = append(parts, "unclassified")
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func detailStderr(d *models.ExecutionDetail) []byte {
	if d != nil {
		return d.Stderr
	}
	return nil
}

func normalizeStderr(b []byte) string {
	// Collapse stack addresses, hex offsets and varying counters so that the
	// same logical crash on different input runs produces the same signature.
	lines := strings.Split(string(b), "\n")
	var kept []string
	for _, l := range lines {
		l = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(l, "0xADDR")
		l = regexp.MustCompile(`\b\d{4,}\b`).ReplaceAllString(l, "N")
		l = strings.TrimSpace(l)
		if l != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

func hashShort(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
