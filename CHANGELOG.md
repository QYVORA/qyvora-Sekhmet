# Changelog

All notable changes to SEKHMET are documented here. This project follows
[Keep a Changelog](https://keepachangelog.com/) and uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- **Unified version system** — `internal/version` identity now also carries
  official QYVORA contact details (website, support, location), surfaced by
  `sekhmet version` in terminal and machine formats.
- **Contact details** — the `version` command, README, and `SECURITY.md`
  surface official QYVORA contact: https://qyvora.org ·
  qyvorasec@gmail.com · Tamale, Ghana.
- **Full markdown/HTML output** — `Print`/`PrintTable` render real markdown
  and HTML (no plain-text fallthrough) for `--output markdown|html`.
- **Machine-output purity** — informational messages move to stderr when a
  machine-readable format is active; `report` now honors `--output` by
  serializing the structured session model instead of the plain-text body.
- **ANSI hygiene** — terminal colors are disabled when stdout is piped or
  redirected or `NO_COLOR` is set, so no escape sequences leak into output.
- **Console** — non-interactive stdin falls back to a plain line reader,
  history persists to `~/.qyvora/sekhmet_history`, and tab completion is
  provided for builtins.
- Fatal config/event errors no longer call `os.Exit(1)` directly; they surface
  through `Execute`'s exit-code contract.

### Added


- **Baseline profiling** — profile a target's normal exit codes, signals,
  runtime and output variance before fuzzing (`sekhmet baseline`).
- **Execution modes** — process (`{fuzz}` / `{stdin}` templates), HTTP and a
  deterministic **simulation** target for CI and testing.
- **Corpus management** — import, crops, persistent store with SHA-256
  dedup, priority and trimming (`sekhmet corpus`).
- **Adaptive mutation** — 17 structured operators (bit/byte, block,
  dictionary insert, JSON structure, boundary, length, splice, …) with a
  seeded RNG (`internal/mutation`).
- **Feedback tracking** — behavioral / edges / blocks novelty scoring feeding
  power scheduling (`internal/feedback`).
- **Power scheduling** — fast / explore / exploit / rare / balanced /
  adaptive strategies with integrity guards (`internal/scheduler`).
- **Detection** — signal-aware crash/hang/anomaly classification plus
  ASan/UBSan/MSan text matching, deduplicated by SHA-256 signature
  (`internal/detection`).
- **Minimization** — delta-debugging reducer for minimal reproducers
  (`sekhmet minimize`).
- **Replay** — reproduce an input against a target (`sekhmet replay`).
- **Sessions & reports** — persistent sessions; terminal/JSON/YAML reporting
  (`sekhmet session`, `sekhmet report`).
- **Safety guards** — execution budgets, size caps, concurrency limits,
  authorization gates, dry-run (`internal/safety`).
- **SecLists integration** — list categories and search wordlists without
  vendoring ~5 GB (`sekhmet wordlists`).
- **Interactive console** — run `sekhmet` with no args for an interactive
  REPL (`internal/console`).
- **Capabilities** — machine-readable capability catalog (`internal/capabilities`).
- **CI & quality** — `.github/workflows/ci.yml`, `.golangci.yml`, Makefile.

### Changed

- Hot path (select → mutate → execute → classify → feedback) tuned for high
  throughput with async deep analysis off the critical path.
- **TLS verification is now the default** for `<target http|https>` execution;
  the new `--insecure-tls` flag opts into `InsecureSkipVerify` for testing
  servers with self-signed certificates.
- **Executable output is truncated at 4 MiB** (`execution.Options.MaxOutput`,
  sourced from `safety.DefaultLimits().MaxOutputBytes` in `fuzz`) using a
  capped buffer that drains without recording; results carry
  `output_truncated`. Helpers that previously read whole files
  (`readSeedFile`) now cap corpus loads at 64 MiB and return errors instead
  of panicking.
- **Process-group termination** — timed-out subprocesses are killed as a
  process group (`Setpgid` + `kill(-pid, SIGKILL)`) so child processes cannot
  outlive the runner; command output collection no longer buffers unbounded
  data.
- **Interrupt handling** — `fuzz`, `baseline`, `minimize` and `replay`
  persist current state and exit 130 on SIGINT/SIGTERM instead of propagating
  a raw abort.

### Fixed

- Data races in the mutation engine and scheduler under concurrent workers.
- Crash signature generation (`itoa`) producing garbage fingerprints.
- Baseline terminal output now renders a clean metric table.

## [0.1.0] - unreleased

Initial foundation release.
