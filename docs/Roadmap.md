# Roadmap

SEKHMET's foundation is complete: baseline profiling, corpus management,
adaptive mutation, feedback tracking, power scheduling, crash/hang/anomaly
detection with fingerprint dedup, minimization, replay, sessions, reporting,
safety guards, SecLists integration, CI and an interactive console.

Planned work, in rough priority order:

## Near term

- **In-memory queue** removal and true signal-driven scheduling (coverage
  feedback loops feeding adaptive power in real time).
- **Sanitizer config** presets (`-fsanitize=address,undefined`) so crash
  reproduction against ASan/UBSan builds is one command.
- **HTTP mode hardening** — headers, auth, body hooks and rate-limit-aware
  pacing.
- **HUD / live dashboard** for long campaigns (epoch updates, exec/s,
  coverage, findings) with an `internal/ui` renderer.
- **`.golangci.yml` tuning** once the near-term features land.

## Mid term

- **Coverage-guided corpus pruning** (minimal set that still reaches all
  observed edges).
- **Seed grammar DSL** so non-binary targets can describe structured input
  templates.
- **Structured crash repro** bundling binary + input + env for shareable
  minimal reproducers.

## Long term

- **Distributed campaign** orchestration across worker hosts, with a shared
  coverage corpus.
- **Regression harness** that replays a session's findings against a new
  build and reports newly-fixed / newly-appearing issues.
- **Ecosystem integration** with the broader QYVORA open-source tools.
