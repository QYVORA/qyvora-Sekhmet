// Package execution implements the high-speed target execution hot path.
// Its job is to launch a target with an input and produce a cheap
// ExecutionResult with minimal overhead. Expensive analysis is kept out of
// this path; it belongs to the detection/triage packages.
package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Runner executes a single input against a target and returns a cheap result.
// It is the narrow seam separating target adapters from the fuzz scheduler.
type Runner interface {
	// Exec runs input against the target honoring the timeout. It must never
	// panic on malformed input; it returns an ExecutionResult with Failed=true
	// and a plan-level error via the returned error if the launch itself fails.
	Exec(ctx context.Context, input []byte, timeout time.Duration) (*models.ExecutionResult, error)
	// Kind returns a short label for the adapter ("process", "http", "file").
	Kind() string
	// Close releases any long-lived resources (e.g. HTTP transport pools).
	Close() error
}

// Options configures the execution layer.
type Options struct {
	// MaxInputSize hard-caps input bytes fed to a target to prevent resource
	// exhaustion on the hot path.
	MaxInputSize int
	// Stdin to feed a CLI/process target. If false, argv or file is used per
	// the target config.
	UseStdin bool
	// Args template for argv-based targets. {fuzz} is substituted with a temp
	// file path; {stdin} is substituted with "-".
	Args []string
	// HTTP timeout/rate defaults are handled by the http runner.
	HTTPTimeout time.Duration
	// HTTPMaxBody caps response bodies captured.
	HTTPMaxBody int
}

func (o *Options) withDefaults() *Options {
	if o == nil {
		o = &Options{}
	}
	if o.MaxInputSize == 0 {
		o.MaxInputSize = 4 << 20 // 4 MiB
	}
	if o.HTTPTimeout == 0 {
		o.HTTPTimeout = 5 * time.Second
	}
	if o.HTTPMaxBody == 0 {
		o.HTTPMaxBody = 4 << 20
	}
	return o
}

// New builds a Runner for the given target and options. It returns a pointed
// error for target classes that this build cannot serve (honest capability
// boundary rather than a fake adapter).
func New(t *models.Target, opts *Options) (Runner, error) {
	if t == nil {
		return nil, fmt.Errorf("execution: nil target")
	}
	opts = opts.withDefaults()

	switch t.Type {
	case models.TargetProcess, models.TargetCLI:
		if t.Path == "" {
			return nil, fmt.Errorf("execution: process target requires a valid path")
		}
		return &processRunner{target: t, opts: opts, inputFile: ""}, nil
	case models.TargetFileFormat:
		return &processRunner{target: t, opts: opts, inputFile: t.InputFile}, nil
	case models.TargetHTTP:
		return newHTTPRunner(t, opts)
	case models.TargetSimulation:
		return &simRunner{}, nil
	default:
		return nil, fmt.Errorf("execution: unsupported target type %q for this build", t.Type)
	}
}

// processRunner executes a local process, feeding the input as stdin, argv, or
// a temp file. It avoids shell concatenation entirely: inputs are passed via
// os/exec process APIs, never through a shell.
type processRunner struct {
	target    *models.Target
	opts      *Options
	inputFile string
}

func (r *processRunner) Kind() string { return "process" }

func (r *processRunner) Close() error { return nil }

func (r *processRunner) Exec(ctx context.Context, input []byte, timeout time.Duration) (*models.ExecutionResult, error) {
	if len(input) > r.opts.MaxInputSize {
		input = input[:r.opts.MaxInputSize]
	}

	var stdin io.Reader
	var argv []string

	switch {
	case r.inputFile != "":
		argv = r.argvFor(r.inputFile)
	case r.opts.UseStdin || len(r.opts.Args) == 0:
		stderr := input // pass via stdin
		_ = stderr
		stdin = newBytesReader(input)
		argv = r.target.Args
	default:
		tmp, clean, err := writeTempInput(input)
		if err != nil {
			return failResult(err), nil
		}
		defer clean()
		argv = r.argvFor(tmp)
	}

	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.target.Path, argv...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stdout, stderr bytesBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	res := &models.ExecutionResult{
		Stdout:     stdout.Bytes(),
		Stderr:     stderr.Bytes(),
		StdoutSize: stdout.Len(),
		StderrSize: stderr.Len(),
		Duration:   elapsed,
		ExitClass:  models.ClassNormalSuccess,
	}

	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitClass = models.ClassUnexpected
		_ = cmd.Process.Kill()
		return res, nil
	}

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			res.Signal = signalName(ee)
			if res.Signal != "" {
				res.ExitClass = models.ClassCrash
			} else if res.ExitCode != 0 {
				res.ExitClass = models.ClassNormalFailure
			}
			return res, nil
		}
		// Launch failure (binary missing, permission, etc).
		res.Failed = true
		res.Error = err.Error()
		return res, nil
	}

	res.ExitCode = 0
	res.ExitClass = models.ClassNormalSuccess
	return res, nil
}

func (r *processRunner) argvFor(input string) []string {
	argv := r.opts.Args
	if len(argv) == 0 {
		argv = r.target.Args
	}
	out := make([]string, 0, len(argv)+1)
	for _, a := range argv {
		switch a {
		case "{fuzz}":
			out = append(out, input)
		case "{stdin}":
			out = append(out, "-")
		default:
			out = append(out, a)
		}
	}
	return out
}

func writeTempInput(data []byte) (string, func(), error) {
	f, err := os.CreateTemp("", "sekhmet-in-*")
	if err != nil {
		return "", nil, err
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		_ = os.Remove(name)
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", nil, err
	}
	return name, func() { _ = os.Remove(name) }, nil
}

func signalName(ee *exec.ExitError) string {
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return ws.Signal().String()
	}
	return ""
}

func failResult(err error) *models.ExecutionResult {
	return &models.ExecutionResult{Failed: true, Error: err.Error()}
}

// bytesBuffer is a concurrency-safe-free minimal []byte sink optimized for a
// single writer (the child process).
type bytesBuffer struct {
	b []byte
}

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}

func (b *bytesBuffer) Bytes() []byte { return b.b }
func (b *bytesBuffer) Len() int      { return len(b.b) }

func newBytesReader(p []byte) io.Reader { return &sliceReader{p: p} }

type sliceReader struct {
	p []byte
	i int
}

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.i >= len(r.p) {
		return 0, io.EOF
	}
	n := copy(p, r.p[r.i:])
	r.i += n
	return n, nil
}

// FingerprintInput returns a stable content hash for deduplication of inputs.
func FingerprintInput(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ResolveTargetPath validates that a path exists and returns the cleaned path.
func ResolveTargetPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("target path is empty")
	}
	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("target executable '%s' does not exist", path)
	}
	return p, nil
}

// IsPlainName reports whether a string contains no shell metacharacters (used
// only for informational display, never for execution).
func IsPlainName(s string) bool {
	return !strings.ContainsAny(s, ";&|`$\\")
}
