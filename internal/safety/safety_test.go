package safety

import (
	"testing"
	"time"
)

func TestExecutionLimit(t *testing.T) {
	g := NewGuardian(Limits{MaxExecutions: 10})
	g.Start()
	for i := 0; i < 10; i++ {
		if err := g.CheckExec(); err != nil {
			t.Fatalf("execution %d should be allowed", i)
		}
		g.Executed()
	}
	if err := g.CheckExec(); err == nil {
		t.Fatal("expected limit error after 10 executions")
	}
}

func TestInputSizeCap(t *testing.T) {
	g := NewGuardian(Limits{MaxInputBytes: 10})
	if err := g.CheckInput(9); err != nil {
		t.Fatalf("input within limit rejected: %v", err)
	}
	if err := g.CheckInput(11); err == nil {
		t.Fatal("expected input size error")
	}
}

func TestAuthorizationGate(t *testing.T) {
	g := NewGuardian(Limits{RequireAuthorized: true})
	if err := g.CheckAuthorized(false); err == nil {
		t.Fatal("expected authorization error when required")
	}
	if err := g.CheckAuthorized(true); err != nil {
		t.Fatalf("authorized target rejected: %v", err)
	}
	// Gate disabled
	g2 := NewGuardian(Limits{RequireAuthorized: false})
	if err := g2.CheckAuthorized(false); err != nil {
		t.Fatalf("gate disabled should not error: %v", err)
	}
}

func TestRuntimeLimit(t *testing.T) {
	g := NewGuardian(Limits{MaxRuntime: 20 * time.Millisecond})
	g.Start()
	time.Sleep(30 * time.Millisecond)
	if err := g.CheckExec(); err == nil {
		t.Fatal("expected runtime limit error")
	}
}

func TestTrackedLimitNotification(t *testing.T) {
	notified := make(chan string, 1)
	g := NewGuardian(Limits{MaxExecutions: 1, NotifyOnExceed: func(k string, _ int64) {
		notified <- k
	}})
	g.Start()
	_ = g.CheckExec()
	g.Executed()
	_ = g.CheckExec()
	select {
	case k := <-notified:
		if k != "executions" {
			t.Fatalf("expected executions notification, got %s", k)
		}
	default:
		t.Fatal("expected notification to fire")
	}
	if len(g.Exceeded()) == 0 {
		t.Fatal("expected exceeded kinds to be recorded")
	}
}
