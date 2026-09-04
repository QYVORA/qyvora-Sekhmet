# Safety Model

SEKHMET is engineered to be safe to run unattended by default. The
`internal/safety` package provides a `Guardian` that enforces hard limits on a
campaign before and during execution.

## Guardrails

| Guard | Default intent | Trigger |
|-------|----------------|---------|
| `MaxExecutions` | Cap total executions | Campaign loop halts when hit |
| `MaxInputBytes` | Cap mutation size | Oversized inputs rejected / clipped |
| `MaxRuntime` | Wall-clock budget | Campaign aborts at deadline |
| `MaxConcurrent` | Bound worker fan-out | Scheduler refuses to over-schedule |
| `RequireAuthorized` | Authorization gate | Blocks remote execution until acknowledged |
| Corpus bytes | Bound stored corpus | Prevents unbounded disk growth |

## Behavior on breach

When a guard is exceeded, the `Guardian` records the exceeded limit
(`Exceeded()`), returns a classified error, and the campaign stops cleanly —
it never silently continues past a limit. This makes runaway fuzzing
impossible without an explicit operator override.

## Dry run

`--dry-run` walks a campaign's configuration (target, budgets, corpus, planned
execution count) and reports what *would* run, without executing anything. Use
it before launching any non-trivial campaign.

## Concurrency

Concurrent workers are bounded by `MaxConcurrent` and the in-memory structures
(engine, scheduler, feedback tracker, corpus) are mutex-/atomic-guarded and
verified under `go test -race`.
