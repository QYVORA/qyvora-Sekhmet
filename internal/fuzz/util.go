package fuzz

import (
	"math/rand"
	"os"
)

// newSeededRand returns a deterministic RNG for reproducible campaigns.
func newSeededRand(seed int64) *rand.Rand {
	if seed == 0 {
		seed = 1337
	}
	return rand.New(rand.NewSource(seed))
}

func readFileImpl(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// shortFingerprint truncates a hex fingerprint to a stable short id.
func shortFingerprint(fp string) string {
	if len(fp) > 12 {
		return fp[:12]
	}
	return fp
}
