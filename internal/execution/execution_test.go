package execution

import (
	"net/http"
	"strings"
	"testing"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

func TestCappedBufferBoundsCapturedBytes(t *testing.T) {
	var b cappedBuffer
	b.max = 8
	in := []byte("0123456789")
	if n, err := b.Write(in); err != nil || n != len(in) {
		t.Fatalf("write returned (%d, %v), want (%d, nil)", n, err, len(in))
	}
	if b.Len() != 8 {
		t.Fatalf("captured %d bytes, want capped at 8", b.Len())
	}
	if got := string(b.Bytes()); got != "01234567" {
		t.Fatalf("captured content %q, want %q", got, "01234567")
	}
	if !b.Capped() {
		t.Fatal("expected truncation flag after overage")
	}
}

func TestCappedBufferExactFitNotTruncated(t *testing.T) {
	var b cappedBuffer
	b.max = 4
	if _, err := b.Write([]byte("abcd")); err != nil {
		t.Fatal(err)
	}
	if b.Capped() {
		t.Fatal("exact fit must not be flagged as truncated")
	}
}

func TestTLSVerifyOffByDefault(t *testing.T) {
	opts := (&Options{}).withDefaults()
	r, err := newHTTPRunner(&models.Target{Type: models.TargetHTTP, Endpoint: "http://example.com"}, opts)
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := r.(*httpRunner).client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	cfg := tr.TLSClientConfig
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify must default to false")
	}
}

func TestTLSVerifyEnabledWithFlag(t *testing.T) {
	opts := (&Options{InsecureTLS: true}).withDefaults()
	r, err := newHTTPRunner(&models.Target{Type: models.TargetHTTP, Endpoint: "http://example.com"}, opts)
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := r.(*httpRunner).client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	cfg := tr.TLSClientConfig
	if !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should be on when InsecureTLS is set")
	}
}

func TestOptionsDefaults(t *testing.T) {
	o := (&Options{}).withDefaults()
	if o.MaxOutput != 4<<20 {
		t.Fatalf("MaxOutput default = %d, want %d", o.MaxOutput, 4<<20)
	}
	if o.MaxInputSize != 4<<20 {
		t.Fatalf("MaxInputSize default = %d, want %d", o.MaxInputSize, 4<<20)
	}
}

func TestNewRejectsNilTarget(t *testing.T) {
	if _, err := New(nil, nil); !strings.Contains(err.Error(), "nil target") {
		t.Fatalf("expected nil-target error, got %v", err)
	}
}
