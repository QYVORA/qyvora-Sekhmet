# SEKHMET

**High-performance, baseline-aware, feedback-driven fuzzing & vulnerability discovery**

SEKHMET is a terminal-first Go framework for **authorized** fuzzing and
vulnerability discovery. Instead of fuzzing blindly, it first understands a
target's **normal behaviour** (a baseline), then mutates and executes inputs
against it, classifying **crashes**, **hangs** and **anomalies** relative to
that baseline — and prioritizing the mutations most likely to reach new code.

> The CLI is `sekhmet`.

---

## What it is

A single interface to run a controllable fuzzing campaign against a target you
are authorized to test:

```
Target → Baseline → Corpus → Mutate → Execute → Classify → Dedup → Report
```

SEKHMET supports three execution modes:

- **Process** — run a local binary with `{fuzz}` / `{stdin}` input templates
- **HTTP** — send fuzzed payloads to a known endpoint
- **Simulation** — a deterministic built-in target for CI, testing and
  learning without a real-world binary

## What it is NOT

- Not an automated vulnerability scanner that attacks anything reachable
- Not a malware / persistence-delivery framework
- Not an unauthorized-access tool

Local targets are scoped to the declared path; remote (HTTP/network) targets
require explicit authorization acknowledgement, and `--dry-run` audits a
campaign without executing anything. SEKHMET is intended only for systems you
are authorized to test. See [SECURITY.md](SECURITY.md) and
[docs/Security-Model.md](docs/Security-Model.md).

## Status

**Foundation.** Baseline profiling, corpus management, 17 mutation operators,
a feedback tracker (behavioral / edges / blocks), power scheduling, crash /
hang / anomaly classification with SHA-256 fingerprint dedup, delta
minimization, replay, sessions, SecLists wordlists, and terminal/JSON/YAML
reporting are implemented and tested. See [CHANGELOG.md](CHANGELOG.md) and
[docs/Roadmap.md](docs/Roadmap.md).

## Features

- **Baseline-first** — profile exit codes, signals, runtime and output
  variance before fuzzing; classify results against that profile
- **Execution modes** — process (`{fuzz}`, `{stdin}` templates), HTTP, and a
  deterministic simulation target
- **Adaptive mutation** — 17 structured operators (bit/byte, block, dictionary
  insert, JSON structure, boundary, length, splice, …) driven by a seeded RNG
- **Feedback-directed** — novelty scoring over behavioral / edge / block
  coverage feeds power scheduling (fast / explore / exploit / rare / balanced
  / adaptive)
- **Crash detection** — signal-aware classification plus ASan/UBSan/MSan
  text matching, all deduplicated by SHA-256 signature
- **Minimization** — delta-debugging reducer turns an interesting input into
  a minimal reproducer
- **Sessions & reports** — persistent sessions with live results; terminal,
  JSON and YAML output; JSONL event stream (`--events`)
- **Wired safety** — execution budgets, size caps, concurrency limits,
  authorization gates and dry-run; see [docs/Safety-Model.md](docs/Safety-Model.md)
- **SecLists integration** — search and load words without vendoring ~5 GB
- **Interactive console** — run `sekhmet` with no args to work interactively

## Install

Build from source (Go 1.26+):

```sh
make build            # builds bin/sekhmet
make test-race        # full test suite with race detector
make check            # gofmt + vet + test
sudo make install     # install to /usr/local/bin
```

## Quick start

Profile a process target's normal behaviour, then fuzz it:

```sh
# 1. Register a target
sekhmet target set --name example --kind process --cmd "./target" --arg "{fuzz}"

# 2. Establish a baseline of normal behaviour
sekhmet baseline --target ./target --samples 64

# 3. Start a campaign
sekhmet fuzz --target ./target --runs 100000 --jobs 4

# 4. Inspect findings
sekhmet crashes --session <id>
sekhmet report  --session <id> --format json
```

Use the deterministic simulation target to see a full run end to end:

```sh
sekhmet target set --name sim --kind simulation
sekhmet baseline --target sim
sekhmet fuzz --target sim --runs 100000
```

## Commands

```
sekhmet baseline     profile the target's normal behaviour
sekhmet fuzz         run a fuzzing campaign
sekhmet analyze      classify and summarize a session's findings
sekhmet corpus       manage the seed corpus (import / crops / list)
sekhmet crashes      list deduplicated crashes from a session
sekhmet minimize     minimize an interesting input
sekhmet replay       reproduce an input against a target
sekhmet session      manage fuzzing sessions (list / show)
sekhmet report       render a campaign report
sekhmet target       manage targets (set / list / show)
sekhmet wordlists    SecLists integration (list / search)
sekhmet capabilities list machine-readable capabilities
sekhmet version      print version and build information
```

Run `sekhmet --help` for flags and `sekhmet <command> --help` per command.

## Event stream & automation

SEKHMET emits a machine-readable JSONL event stream via `--events` (stdout,
stderr, or a file) using the QYVORA event envelope —
`{schema_version, timestamp, execution_id, framework, level, event, data}` —
with `framework: "sekhmet"`. Agents and CI consume this directly.

## Configuration

Options are resolved by precedence: **CLI flags > `QYVORA_SEKHMET_*`
environment variables > config file > defaults**. Use `QYVORA_SEKHMET_SESSION_DIR`
to relocate session storage (default `./sessions`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/](docs/). This project adopts
the QYVORA [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
