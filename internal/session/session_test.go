package session

import (
	"testing"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir())
	sess := Begin("tgt-1")
	sess.AddCrash(&models.Finding{
		Title: "crash", RuleID: "R", Category: "fuzz",
		Classification: models.ClassificationCrashDetected,
		Attributes:     map[string]string{"k": "v"},
	})
	sess.Finish()

	path, err := s.Save(sess)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != sess.ID {
		t.Fatalf("id mismatch: %s vs %s", loaded.ID, sess.ID)
	}
	if len(loaded.Crashes) != 1 {
		t.Fatalf("expected 1 crash after load, got %d", len(loaded.Crashes))
	}
	if path == "" {
		t.Fatal("expected a save path")
	}
}

func TestListIDs(t *testing.T) {
	s := NewStore(t.TempDir())
	a := Begin("t1")
	b := Begin("t2")
	_, _ = s.Save(a)
	_, _ = s.Save(b)
	ids, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 session ids, got %d", len(ids))
	}
}

func TestSessionCrashDedup(t *testing.T) {
	sess := Begin("t")
	mk := func() *models.Finding {
		return &models.Finding{
			Title: "x", RuleID: "R", Category: "fuzz",
			Classification: models.ClassificationCrashDetected,
			Attributes:     map[string]string{"k": "v"},
		}
	}
	sess.AddCrash(mk())
	sess.AddCrash(mk()) // identical fingerprint -> dedup
	if len(sess.Crashes) != 1 {
		t.Fatalf("expected dedup to 1 crash, got %d", len(sess.Crashes))
	}
}
