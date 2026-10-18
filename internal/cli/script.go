package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-script/ast"
	"github.com/QYVORA/qyvora-script/binding"
	"github.com/QYVORA/qyvora-script/interpreter"
	"github.com/QYVORA/qyvora-script/parser"

	"github.com/QYVORA/qyvora-sekhmet/internal/capabilities"
	"github.com/QYVORA/qyvora-sekhmet/internal/corpus"
	errs "github.com/QYVORA/qyvora-sekhmet/internal/errors"
	"github.com/QYVORA/qyvora-sekhmet/internal/events"
	"github.com/QYVORA/qyvora-sekhmet/internal/execution"
	"github.com/QYVORA/qyvora-sekhmet/internal/fuzz"
	"github.com/QYVORA/qyvora-sekhmet/internal/mutation"
	"github.com/QYVORA/qyvora-sekhmet/internal/safety"
	"github.com/QYVORA/qyvora-sekhmet/internal/scheduler"
	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// newScriptCmd adds the flat `script` utility that runs the shared
// qyvora-script engine (ARCH-004 Part C). `script run <file>` executes a
// script; `script build <file>` prints a dry-run plan. A bare `script <file>`
// runs the file for convenience.
func newScriptCmd() *cobra.Command {
	run := func(dryRun bool) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			return runScriptFile(cmd, args[0], dryRun)
		}
	}
	cmd := &cobra.Command{
		Use:   "script <file>",
		Short: "run a .qs script with the shared scripting engine",
		Args:  cobra.ExactArgs(1),
		RunE:  run(false),
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "run <file>",
			Short: "execute a script file",
			Args:  cobra.ExactArgs(1),
			RunE:  run(false),
		},
		&cobra.Command{
			Use:   "build <file>",
			Short: "validate a script file and print a dry-run plan",
			Args:  cobra.ExactArgs(1),
			RunE:  run(true),
		},
	)
	return cmd
}

// runScriptFile parses (build) or executes (run) a script against sekhmet's
// binding.
func runScriptFile(cmd *cobra.Command, path string, dryRun bool) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading script: %w", err)
	}
	prog, err := parser.Parse(string(src))
	if err != nil {
		return errs.NewExitError(2, err.Error())
	}
	if dryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "script plan "+path)
		for _, line := range ast.Describe(prog) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  -", line)
		}
		return nil
	}
	b := &scriptBinding{
		ctx:   cmd.Context(),
		allow: allowShellOut,
		cfg:   map[string]string{},
	}
	engine := interpreter.NewEngine(b)
	engine.Events = os.Stderr
	if err := engine.RunProgram(prog); err != nil {
		return errs.NewExitError(2, err.Error())
	}
	return nil
}

// scriptBinding implements binding.Binding against sekhmet's fuzzing
// assessment. Module ids map to the tool's operations: "assess" sets a target
// and runs a bounded fuzzing campaign; the specific capability ids run their
// own operation. The campaign uses conservative defaults (one worker, a small
// execution cap) so a script run stays quick and deterministic against the
// simulation target.
type scriptBinding struct {
	ctx      context.Context
	allow    bool
	cfg      map[string]string
	last     *models.Session
	lastRes  *fuzz.Result
	lastBase *models.Baseline
}

var _ binding.Binding = (*scriptBinding)(nil)

// Capabilities lists the whole-assessment "assess" entry plus every
// capability in sekhmet's machine contract.
func (b *scriptBinding) Capabilities() []string {
	catalog := capabilities.Catalog()
	ids := make([]string, 0, 1+len(catalog))
	ids = append(ids, "assess")
	for _, t := range catalog {
		ids = append(ids, t.ID)
	}
	return ids
}

// AllowShellOut reports whether exec/run.cmd may run: only with the explicit
// flag or the shared opt-in environment variable.
func (b *scriptBinding) AllowShellOut() bool { return b.allow || shellOutEnv() }

func shellOutEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(interpreter.AllowShellOutEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Call dispatches one engine capability invocation.
func (b *scriptBinding) Call(capabilityID string, args []binding.Value) (binding.Value, error) {
	switch capabilityID {
	case binding.CapEcho:
		_, _ = fmt.Fprintln(os.Stdout, asStr(args, 0))
		return nil, nil

	case binding.CapSetConfig:
		b.cfg[asStr(args, 0)] = asStr(args, 1)
		return nil, nil

	case binding.CapGetConfig:
		return b.cfg[asStr(args, 0)], nil

	case binding.CapStart:
		return nil, b.start(asStr(args, 0))

	case binding.CapStop, binding.CapIsRunning:
		if capabilityID == binding.CapIsRunning {
			return false, nil
		}
		return nil, nil

	case binding.CapShow:
		return nil, b.show(asStr(args, 0))

	case binding.CapReport:
		return nil, b.report(asStr(args, 0))

	case binding.CapProp:
		return b.prop(asStr(args, 0)), nil

	case binding.CapExec:
		return nil, b.exec(asStr(args, 0))
	}
	return nil, fmt.Errorf("unknown capability %q", capabilityID)
}

// start dispatches a module id to the matching sekhmet operation.
func (b *scriptBinding) start(id string) error {
	switch id {
	case "assess", "":
		if err := b.setTarget(); err != nil {
			return err
		}
		return b.runFuzz()
	case "sekhmet.target.discover":
		return b.setTarget()
	case "sekhmet.baseline.profile":
		return b.runBaseline()
	case "sekhmet.campaign.fuzz":
		return b.runFuzz()
	case "sekhmet.findings.list":
		return b.listFindings()
	case "sekhmet.crash.list":
		return b.listCrashes()
	case "sekhmet.report.generate":
		return b.report(firstNonEmpty(b.cfg["report.out"], "sekhmet-report.json"))
	case "sekhmet.input.minimize", "sekhmet.result.replay", "sekhmet.corpus.manage", "sekhmet.wordlists.discover":
		return fmt.Errorf("module %q needs an explicit input; run it via the CLI", id)
	}
	if !hasCapability(id) {
		return fmt.Errorf("unknown module %q", id)
	}
	return fmt.Errorf("module %q is not scriptable; run it via the CLI", id)
}

// setTarget records the fuzzing target from the script config (simulation by
// default when assess.sim is on).
func (b *scriptBinding) setTarget() error {
	now := time.Now().UTC()
	var t *models.Target
	if isTrue(b.cfg["assess.sim"]) || isTrue(b.cfg["sim"]) {
		t = &models.Target{
			Name: "simulation", Type: models.TargetSimulation, Sim: true,
			Auth: models.Authorization{Granted: true, GrantedAt: now}, CreatedAt: now,
		}
	} else {
		t = &models.Target{
			Name: "script", Type: models.TargetCLI,
			Path:      firstNonEmpty(b.cfg["assess.path"], b.cfg["path"]),
			Auth:      models.Authorization{Granted: isTrue(b.cfg["assess.authorized"]), GrantedAt: now},
			CreatedAt: now,
		}
	}
	if err := app.targets.Set(t); err != nil {
		return errs.WrapExitError(2, "setting target", err)
	}
	app.emitf("target set: %s", t.DisplayName())
	return nil
}

// targetRunner builds an execution runner for the current target.
func (b *scriptBinding) targetRunner() (execution.Runner, *models.Target, error) {
	t, err := app.requireTarget()
	if err != nil {
		return nil, nil, err
	}
	runner, err := execution.New(t, &execution.Options{
		UseStdin:    true,
		Args:        t.Args,
		HTTPTimeout: 5 * time.Second,
		MaxOutput:   int(safety.DefaultLimits().MaxOutputBytes),
		InsecureTLS: app.insecureTLS,
	})
	if err != nil {
		return nil, nil, errs.WrapExitError(2, "building target runner", err)
	}
	return runner, t, nil
}

// runBaseline profiles the current target's normal behaviour.
func (b *scriptBinding) runBaseline() error {
	runner, t, err := b.targetRunner()
	if err != nil {
		return err
	}
	defer func() { _ = runner.Close() }()
	base, err := buildBaseline(b.ctx, runner)
	if err != nil {
		return errs.WrapExitError(1, "baseline profiling before fuzz", err)
	}
	b.lastBase = base
	app.emitf("baseline profiled for %s", t.DisplayName())
	printBaseline(base)
	return nil
}

// runFuzz runs a bounded fuzzing campaign against the current target and
// captures the resulting session.
func (b *scriptBinding) runFuzz() error {
	runner, t, err := b.targetRunner()
	if err != nil {
		return err
	}
	defer func() { _ = runner.Close() }()

	base, err := buildBaseline(b.ctx, runner)
	if err != nil {
		return errs.WrapExitError(1, "baseline profiling before fuzz", err)
	}
	workers := 1
	executions := int64(25)
	campaign := fuzz.Campaign{
		Target:         t,
		Base:           base,
		Runner:         runner,
		Corpus:         corpus.New(corpusDir()),
		Workers:        workers,
		Timeout:        2 * time.Second,
		ExecutionLimit: executions,
		Strategy:       scheduler.ParseStrategy("adaptive"),
		SeedRNG:        1337,
		Retention:      100000,
		Emit:           appEmitEvent,
		OnNewBehavior:  func(_ []string, _ mutation.Operator) {},
		Safety:         buildGuardian(workers, executions, 0, 0),
	}
	app.emitEvent(events.ScanStarted, map[string]any{"target": t.DisplayName()})
	app.emitEvent(events.CampaignStarted, map[string]any{"target": t.DisplayName(), "workers": workers})

	res, sess, err := fuzz.New(campaign).Run(b.ctx)
	if err != nil {
		return errs.WrapExitError(1, "campaign failed", err)
	}
	sess.Result = buildSessionResult(res, sess.ID, t.ID, workers)
	sess.Finish()
	if _, err := app.persistSession(sess); err != nil {
		return errs.WrapExitError(1, "persisting session", err)
	}
	b.last = sess
	b.lastRes = res
	b.lastBase = base
	app.emitEvent(events.CampaignCompleted, map[string]any{
		"executions": res.Executions, "unique_crashes": res.UniqueCrashes,
		"new_behaviors": res.NewBehaviors, "corpus_size": res.CorpusSize,
	})
	app.emitEvent(events.ScanCompleted, map[string]any{
		"session": sess.ID, "crashes": len(sess.Crashes), "findings": len(sess.Findings),
	})
	printCampaignSummary(res)
	printFindingsSummary(sess)
	return nil
}

// listFindings prints the findings recorded in the last session.
func (b *scriptBinding) listFindings() error {
	sess := b.last
	if sess == nil {
		return fmt.Errorf("no session; run a module first")
	}
	sum := summarizeFindings(sess)
	app.emitf("session %s total=%d high=%d medium=%d low=%d info=%d crashes=%d",
		sess.ID, sum.Total, sum.High, sum.Medium, sum.Low, sum.Info, sum.Crashes)
	return nil
}

// listCrashes prints the crashes recorded in the last session.
func (b *scriptBinding) listCrashes() error {
	sess := b.last
	if sess == nil {
		return fmt.Errorf("no session; run a module first")
	}
	app.emitf("session %s unique crashes: %d", sess.ID, len(sess.Crashes))
	return nil
}

// show prints capability metadata for a module id.
func (b *scriptBinding) show(id string) error {
	catalog := capabilities.Catalog()
	rows := make([][]string, 0, len(catalog)+1)
	if id == "assess" || id == "" {
		rows = append(rows, []string{"assess", "set a target and run a bounded fuzzing campaign", "exploit"})
	}
	for _, t := range catalog {
		if id == "" || t.ID == id {
			rows = append(rows, []string{t.ID, t.Name, t.Category})
		}
	}
	if len(rows) == 0 {
		return fmt.Errorf("unknown module %q", id)
	}
	app.printer.PrintTable([]string{"capability", "name", "category"}, rows)
	return nil
}

// report renders the last session's campaign report, optionally to a file.
func (b *scriptBinding) report(path string) error {
	if b.last == nil {
		return fmt.Errorf("no session to report; run a module first")
	}
	body := renderSessionFormatted(b.last, app.printer.Format())
	if path != "" {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return errs.WrapExitError(1, "writing report", err)
		}
		app.emitf("report written to %s", path)
		return nil
	}
	_, _ = fmt.Fprintln(app.printer.Writer(), body)
	return nil
}

// prop resolves live properties for $(...) interpolation from the last run.
func (b *scriptBinding) prop(name string) binding.Value {
	switch name {
	case "session.id", "result.id":
		if b.last != nil {
			return b.last.ID
		}
		return nil
	case "session.target", "result.target":
		if b.last != nil {
			return b.last.TargetID
		}
		return nil
	}
	if b.lastRes == nil {
		return nil
	}
	switch name {
	case "result.executions":
		return fmt.Sprintf("%d", b.lastRes.Executions)
	case "result.corpus":
		return fmt.Sprintf("%d", b.lastRes.CorpusSize)
	case "result.behaviors":
		return fmt.Sprintf("%d", b.lastRes.NewBehaviors)
	case "result.crashes":
		return fmt.Sprintf("%d", b.lastRes.UniqueCrashes)
	case "result.duration":
		return b.lastRes.Duration.String()
	}
	return nil
}

// exec runs a one-shot sekhmet command line. The interpreter only reaches here
// after AllowShellOut reported true.
func (b *scriptBinding) exec(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if code := ExecuteArgs(strings.Fields(line)); code != 0 {
		return fmt.Errorf("command exited with code %d", code)
	}
	return nil
}

// hasCapability reports whether id is a registered sekhmet capability.
func hasCapability(id string) bool {
	for _, t := range capabilities.Catalog() {
		if t.ID == id {
			return true
		}
	}
	return false
}

func asStr(args []binding.Value, i int) string {
	if i >= len(args) || args[i] == nil {
		return ""
	}
	if s, ok := args[i].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", args[i])
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
