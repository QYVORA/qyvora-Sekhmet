package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func TestSimRunnerNormal(t *testing.T) {
	r := &simRunner{}
	res, err := r.Exec(context.Background(), []byte("hello world"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitClass != models.ClassNormalSuccess {
		t.Fatalf("expected normal success, got %s", res.ExitClass)
	}
}

func TestSimRunnerCrashToken(t *testing.T) {
	r := &simRunner{}
	res, _ := r.Exec(context.Background(), []byte("prefix SECLISTS_CRASH"), time.Second)
	_ = res
	res, _ = r.Exec(context.Background(), []byte("SEKHMET_CRASH"), time.Second)
	if res.Signal != "SIGSEGV" {
		t.Fatalf("expected SIGSEGV, got %q", res.Signal)
	}
	if res.ExitClass != models.ClassCrash {
		t.Fatalf("expected crash class, got %s", res.ExitClass)
	}
}

func TestSimRunnerBehaviorToken(t *testing.T) {
	r := &simRunner{}
	res, _ := r.Exec(context.Background(), []byte("SEKHMET_BEHAVIOR"), time.Second)
	if !strings.Contains(string(res.Stdout), "novel behavior") {
		t.Fatalf("expected novel behavior output, got %q", res.Stdout)
	}
}

func TestFingerprintInputDeterministic(t *testing.T) {
	a := FingerprintInput([]byte("abc"))
	b := FingerprintInput([]byte("abc"))
	if a != b {
		t.Fatalf("expected deterministic fingerprint, got %s vs %s", a, b)
	}
	if a == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}
