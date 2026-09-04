package detection

import (
	"testing"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func TestClassifyCrashSignalWins(t *testing.T) {
	res := &models.ExecutionResult{Signal: "SIGSEGV", ExitClass: models.ClassNormalSuccess}
	if got := Classify(res, nil); got != models.ClassCrash {
		t.Fatalf("expected crash, got %s", got)
	}
}

func TestClassifyTimeoutIsHang(t *testing.T) {
	res := &models.ExecutionResult{TimedOut: true}
	if got := Classify(res, nil); got != models.ClassHang {
		t.Fatalf("expected hang, got %s", got)
	}
}

func TestSignatureDedupesNormalizedStderr(t *testing.T) {
	a := &models.ExecutionResult{
		ExitCode: -11, Signal: "SIGSEGV",
		Stderr: []byte("fatal error: SEGV\naddress 0x7ffc1234\ncorrupt 99999\n"),
	}
	b := &models.ExecutionResult{
		ExitCode: -11, Signal: "SIGSEGV",
		Stderr: []byte("fatal error: SEGV\naddress 0x7ffc5678\ncorrupt 12345\n"),
	}
	sa := Signature(a, &models.ExecutionDetail{Stderr: a.Stderr})
	sb := Signature(b, &models.ExecutionDetail{Stderr: b.Stderr})
	if sa != sb {
		t.Fatalf("expected identical normalized signatures, got %q vs %q", sa, sb)
	}
}

func TestSignatureDistinguishesDifferentFaults(t *testing.T) {
	a := &models.ExecutionResult{Signal: "SIGSEGV", Stderr: []byte("heap-buffer-overflow")}
	b := &models.ExecutionResult{Signal: "SIGABRT", Stderr: []byte("assertion failed")}
	sa := Signature(a, &models.ExecutionDetail{Stderr: a.Stderr})
	sb := Signature(b, &models.ExecutionDetail{Stderr: b.Stderr})
	if sa == sb {
		t.Fatalf("expected distinct signatures, both %q", sa)
	}
}

func TestSanitizerFinding(t *testing.T) {
	stderr := []byte("log line\n==ERROR: AddressSanitizer: heap-buffer-overflow\nbacktrace")
	msg, ok := SanitizerFinding(stderr)
	if !ok {
		t.Fatal("expected sanitizer finding")
	}
	if msg == "" {
		t.Fatal("expected non-empty sanitizer message")
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 5: "5", -11: "-11", 2147483647: "2147483647", -2147483648: "-2147483648"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Fatalf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}
