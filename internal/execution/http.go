package execution

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// httpRunner executes inputs as HTTP requests against an authorized endpoint.
// It understands that dynamic web applications vary naturally: the baseline
// engine normalizes request ids/timestamps so they are not reported as
// anomalies. Response bodies and headers are captured for the deep path.
type httpRunner struct {
	mu        sync.Mutex
	client    *http.Client
	target    *models.Target
	opts      *Options
	seq       uint64
	rateLimit time.Duration
	lastSend  time.Time
}

func newHTTPRunner(t *models.Target, opts *Options) (Runner, error) {
	if t.Endpoint == "" {
		return nil, fmt.Errorf("execution: http target requires an endpoint URL")
	}
	if !strings.HasPrefix(t.Endpoint, "http://") && !strings.HasPrefix(t.Endpoint, "https://") {
		return nil, fmt.Errorf("execution: endpoint must be http:// or https://: %s", t.Endpoint)
	}
	client := &http.Client{
		Timeout: opts.HTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // user-scoped authorized testing
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 16,
		},
	}
	return &httpRunner{
		client: client,
		target: t,
		opts:   opts,
	}, nil
}

func (r *httpRunner) Kind() string { return "http" }

func (r *httpRunner) Close() error {
	if tr, ok := r.client.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
	return nil
}

func (r *httpRunner) Exec(ctx context.Context, input []byte, _ time.Duration) (*models.ExecutionResult, error) {
	r.mu.Lock()
	if r.rateLimit > 0 {
		if d := r.rateLimit - time.Since(r.lastSend); d > 0 {
			select {
			case <-time.After(d):
			case <-ctx.Done():
				r.mu.Unlock()
				return failResult(ctx.Err()), nil
			}
		}
	}
	r.lastSend = time.Now()
	r.seq++
	n := r.seq
	client := r.client
	r.mu.Unlock()

	body := bytes.NewReader(input)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.target.Endpoint, body)
	if err != nil {
		return failResult(err), nil
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Sekhmet-Seq", fmt.Sprintf("%d", n))

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	out := &models.ExecutionResult{Duration: elapsed}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			out.TimedOut = true
			out.ExitClass = models.ClassUnexpected
			return out, nil
		}
		out.Failed = true
		out.Error = err.Error()
		return out, nil
	}
	defer resp.Body.Close()

	// Capture status/headers cheaply, body up to the cap.
	var headerMap map[string][]string
	if len(resp.Header) > 0 {
		headerMap = make(map[string][]string, len(resp.Header))
		for k, v := range resp.Header {
			headerMap[k] = v
		}
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, int64(r.opts.HTTPMaxBody)))

	out.ExitCode = resp.StatusCode
	out.StatusCode = resp.StatusCode
	out.Headers = headerMap
	out.Body = bodyBytes
	out.BodySize = len(bodyBytes)

	// Cheap classification: 5xx / 4xx are "errors" but are often expected in
	// baseline; the classifier compares against the baseline to decide.
	switch {
	case resp.StatusCode >= 500:
		out.ExitClass = models.ClassNormalFailure
	case resp.StatusCode >= 400:
		out.ExitClass = models.ClassExpectedError
	default:
		out.ExitClass = models.ClassNormalSuccess
	}
	return out, nil
}
