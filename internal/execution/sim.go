package execution

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// simRunner is the deterministic simulation target used for CI, tests and
// development without a real-world target. It implements a small fake parser
// that:
//
//   - returns normal output for most inputs,
//   - returns a distinct "new behavior" signature when the input contains a
//     registered magic token,
//   - "crashes" (signal SIGSEGV) when the input contains a crash token,
//   - "hangs" (times out) when the input contains a hang token.
//
// The magic tokens are the fixed constants below; because the behavior is
// fully specified, this is the canonical way to test baseline, mutation,
// coverage, crash detection, dedup, minimization and reporting end to end.
type simRunner struct{}

const (
	simDefaultCrash    = "SEKHMET_CRASH"
	simDefaultHang     = "SEKHMET_HANG"
	simDefaultBehavior = "SEKHMET_BEHAVIOR"
)

func (s *simRunner) Kind() string { return "simulation" }

func (s *simRunner) Close() error { return nil }

func (s *simRunner) Exec(ctx context.Context, input []byte, _ time.Duration) (*models.ExecutionResult, error) {
	start := time.Now()
	res := &models.ExecutionResult{Duration: 0, ExitClass: models.ClassNormalSuccess}

	if bytes.Contains(input, []byte(simDefaultCrash)) {
		res.ExitClass = models.ClassCrash
		res.Signal = "SIGSEGV"
		res.ExitCode = -11
		res.Stderr = []byte("simulation: SIGSEGV (address 0x0)\n")
		res.StderrSize = len(res.Stderr)
		res.Duration = time.Since(start)
		return res, nil
	}

	if bytes.Contains(input, []byte(simDefaultHang)) {
		select {
		case <-ctx.Done():
			res.TimedOut = true
			res.ExitClass = models.ClassUnexpected
			res.Duration = time.Since(start)
			return res, nil
		case <-time.After(10 * time.Second):
			// Unreachable under a normal timeout; guarded for safety.
		}
	}

	if bytes.Contains(input, []byte(simDefaultBehavior)) {
		res.ExitCode = 0
		res.Stdout = []byte("simulation: novel behavior reached\n")
		res.StdoutSize = len(res.Stdout)
		res.Duration = time.Since(start)
		return res, nil
	}

	// Normal deterministic parser output: echoes input length in a way that
	// depends only on the content, so coverage/behavior feedback is stable.
	res.ExitCode = 0
	label := fmt.Sprintf("sim:ok:%d", hashSeed(input))
	res.Stdout = []byte(label)
	res.StdoutSize = len(res.Stdout)
	res.Duration = time.Since(start)
	return res, nil
}

func hashSeed(b []byte) uint32 {
	var h uint32 = 2166136261
	for _, c := range b {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}
