# SEKHMET — Capability Audit, Gap Analysis and Conformance Review

**Framework:** `qyvora-sekhmet` · **Audit date:** 2026-10-21 · **Audit type:** read-only, evidence-based
**Deliverable:** A (per-framework audit report), sections 1–16 as required by `PROMPT.md` §13

---

## 1. Framework identity and intended domain

| | |
|---|---|
| Canonical name | `sekhmet` |
| Domain | **Authorized fuzzing and vulnerability discovery.** Baseline-first, feedback-driven mutation fuzzing against targets the operator is authorized to test; local process, HTTP or built-in simulation execution only. |
| Repository | `products/QYVORA-AI/frameworks/qyvora-sekhmet` |
| Module | `github.com/QYVORA/qyvora-sekhmet` (independent Go module, `go 1.26.5`; does **not** import `qyvora-common` — verified in `go.mod` and by `rg qyvora-common` = none) |
| Entry point | `cmd/sekhmet` → `internal/cli` (`ExecuteArgs`, `internal/cli/root.go:85-135`) |
| Version reported by shipped binary | `{"framework":"sekhmet","version":"v0.1.0","commit":"680a745","date":"2026-10-21T01:20:51Z","build_user":"wsuits6","go_version":"go1.27.1","arch":"amd64","os":"linux","website":…,"built_in":"Tamale, Ghana"}` (`version -o json`, verified live, exit 0 — full §4 spec metadata set) |
| Version vs git | HEAD `680a745` **is tag `v0.10.1`** (`git describe`), `0` commits ahead of `origin/main`, working tree **clean**; the binary still reports `v0.1.0` because `Makefile:15` defaults `VERSION ?= v0.1.0` (see §12 D-10) |
| Size (measured) | 8,395 non-test Go lines, 1,246 test Go lines, 15 `_test.go` files, 27 packages (`find … \| xargs wc -l`); `bin/sekhmet` 11.3 MB, **stripped** (`file`, `Makefile:20` `-s -w`) |
| Shared dependencies | `qyvora-script v0.1.0`, `qyvora-tui v0.11.0` (brand helpers only; session engine not linked — see §10), `spf13/cobra`+`viper`, `chzyer/readline`, `fatih/color`, `muesli/termenv`, `go.yaml.in/yaml/v3` |
| CLI grammar | `<tool> <tier> <module> [<action>] [target] [flags]`, tiers `recon` / `exploit` / `report` (`internal/cli/root.go:170-197`), plus 10 hidden pre-tier aliases (`:204-215`) |
| Git state at audit start and end | clean `main`; unchanged by this audit |

Domain boundary statement (own README §"What it is / What it is NOT", `internal/cli/root.go:137-143`): one-shot CLI to run controllable,
authorized fuzzing campaigns — profiling normal behaviour, mutating, executing, classifying crashes/hangs/anomalies,
deduplicating and reporting. No scanning of arbitrary hosts, no exploitation of third-party systems, no persistence.

## 2. Audit scope and evidence sources

**Method.** Static source inspection plus non-destructive local execution of the shipped binary, local `go test` /
`go test -race` / `go vet` / `gofmt`, and the shared conformance engine. No feature implementation, no refactoring,
no configuration changes, no repository edits other than this file.

**Evidence read.** `README.md`, `docs/{CLI,Roadmap,Safety-Model,Security-Model}.md`, `CHANGELOG.md`, `SECURITY.md`,
`Makefile`, `.goreleaser.yaml`, `.golangci.yml`, `.github/workflows/{ci,release}.yml`, `internal/cli/*` (root, app,
fuzz, baseline, target, report, crashes, analyze, corpus, session, wordlists, replay, minimize, script, updates,
capabilities, completion, console, tui), `internal/{fuzz,mutation,generation,scheduler,corpus,baseline,feedback,detection,execution,minimization,safety,events,exitcode,config,capabilities,output}/*`,
`pkg/models/*`, plus ecosystem contracts: `shared/qyvora-common/contract/contract.go`,
`shared/qyvora-common/conformance/runner.go`, and
`knowledge/qyvora-docs/09-technical/cross-project/standards/QYVORA-TOOL-OUTPUT-SPEC.md` (§1–§8, incl. the sekhmet
rows in §1/§3/§4/§5/§6/§7).

**Commands executed (all local, exit codes captured without shell pipelines unless noted):** `version` (`-o json`),
`capabilities` (`-o json`), `updates`, `completion [bash|zsh|…]`, `--help` and per-command `--help`,
`session list/show`, `target list/set/show`, `corpus list`, `wordlists list/path/search`, `analyze`,
`report --session <id>` in terminal/json/yaml/markdown/html, `crashes --session <id>`, `script build examples/assess.qs`,
`baseline --samples 2 --warmup 1 --events <file>`, `fuzz --dry-run`, plus deliberate negative probes (~70
invocations: unknown flags, unknown subcommands, tier-parent typos, missing `--session`, invalid formats,
unauthorized target, unsupported target type), `go test ./...`, `go test -race ./...`, `go vet ./...`, `gofmt -l .`,
and `qyvora-conformance -skip-build -bin …:sekhmet`.

**Environment isolation.** Structured runs used `QYVORA_SEKHMET_SESSION_DIR=/tmp/opencode/sk/sessions` or
`/tmp/opencode/sk2/sessions` so session/target files land outside the repository.

**Disclosures (artifacts and deviations created during this audit):**

1. **An exit-code probe chain started a real 2-minute fuzz campaign.** The bare command `sekhmet fuzz` (no flags) was
   executed as part of a systematic argument probe; because a `simulation` target was already registered and authorized
   in the repository's runtime store, it started a campaign immediately at 03:41:58 and ran until the audit harness
   killed the command at 03:43:58. The engine persisted `sessions/sess-398854da40d8704c.session.json`
   (state `finished`, 238,121 executions, 1,989.6/s, corpus 19,136, 0 crashes). Target: the **built-in deterministic
   simulation target only** — local, in-process, no external system and no network. The same chain had already run the
   default `baseline` (35 executions) against that simulation target. No process was left running; `git status` stayed
   clean (`sessions/` and `targets/` are gitignored, `.gitignore:8,26`). This incident also produced useful §13
   evidence (graceful termination → coherent saved session) and demonstrated finding **D-7**.
2. **Deliberate small probes:** `baseline --samples 2 --warmup 1 --events …` (3 local simulation executions, verified
   event envelope, session dir redirected to `/tmp`); `fuzz --executions 1` / `baseline --samples 1` against a
   registered `--type network` target — both fail at runner construction (`execution: unsupported target type "network"`)
   **before any socket is opened, so no packet was sent**; an unauthorized `--type cli` target to verify the exit-3
   refusal (refused before execution). All three wrote only to `/tmp/opencode/sk2/sessions/`.
3. **Pre-existing runtime artifacts at audit start (not created by this audit):** `sessions/sess-03863a4364964cdf.session.json`
   (02:43), `sessions/sess-c7530ed02e5b62c6.session.json` (2026-10-18), `sessions/targets/targets.json`
   (simulation target authorized 2026-10-18), and `/tmp/opencode/sk/sessions/*` from an earlier audit pass.
4. Temporary logs only in `/tmp/opencode/` (`sk_race*.log`, `sk_test*.log`, `conf3.json`, probe outputs).
   **No source, test, configuration, dependency, documentation or asset file was created, modified or deleted. Only this
   `AUDIT.md` is new.** The untracked-but-gitignored `continue_AI.sh` (`.gitignore:22`) was left untouched.
5. `sekhmet updates` performs **no network call** (verified output: "offline (no automatic update checks)") — unlike
   sibling `nzinga`, whose `updates` probe made one GitHub API request (disclosed in `qyvora-nzinga/AUDIT.md` §2).

**Verified vs inferred.** Every behaviour claim below marked *verified* was reproduced against the shipped binary or read
at the cited `file:line`. Claims about absent behaviour are labelled *inferred from absence* (search performed, nothing
found) or *unverified*.

## 3. Existing capability inventory

Commands (all verified to exist via `--help`, exit 0):

| Surface | Members | Evidence |
|---|---|---|
| Root utilities | `version`, `capabilities` (alias `caps`), `completion` (bash/zsh/fish/powershell), `updates` (aliases `update`/`upgrade`), `script` (`build`/`run`), `help` | `internal/cli/root.go:167` |
| Tier `recon` | `baseline`, `analyze`, `corpus` (`import`/`crop`/`list`), `session` (`list`/`show`), `target` (`set`/`list`/`show`), `wordlists` (`list`/`path`/`search`) | `internal/cli/root.go:178-185` |
| Tier `exploit` | `fuzz`, `replay`, `minimize` | `internal/cli/root.go:191` |
| Tier `report` | root render (`--session`, `--out`), `crashes` | `internal/cli/root.go:193-195`, `internal/cli/report.go:23-50` |
| Hidden pre-tier aliases | `baseline`, `analyze`, `corpus`, `session`, `target`, `wordlists`, `fuzz`, `replay`, `minimize`, `crashes` (all verified resolving) | `internal/cli/root.go:204-215` |
| Retired / unwired | TUI (`commandTUI()` defined at `internal/cli/tui.go:96` but **no caller anywhere** — `rg 'commandTUI\(\)'` → definition only); console REPL (`internal/cli/console.go:1` `//go:build ignore`, "no longer wired in") | verified by search + bare invocation prints banner+help |

Engine and modules:

- **Campaign hot path** `SELECT corpus → MUTATE ×128/seed → EXECUTE → CLASSIFY → FEEDBACK → DEDUP → REPORT`
  (`internal/fuzz/fuzz.go:156-280`, per-seed loop `:204-276`); baseline is always built before fuzzing
  ("never fuzz blindly", `internal/cli/fuzz.go:63-66`).
- **Execution runners:** process/CLI (`{fuzz}`, `{stdin}`, `--input-file` templates), HTTP, simulation
  (`internal/execution/execution.go:83-107`, `http.go:21`, `sim.go:25`); `os/exec` process APIs, never shell
  concatenation (`execution.go:110-113` comment); process-group signalling on unix/windows/other.
- **Mutation:** 17 operators in `AllOperators` (`internal/mutation/mutation.go`), seeded RNG; `splice` is a
  documented no-op and `lengthMutation` has a no-op default (`internal/mutation/operators.go`).
- **Feedback:** `none|behavioral|edges|blocks` kinds (`internal/feedback/feedback.go`), but the baseline hardcodes
  `Coverage: models.CoverageProfile{Kind: "behavioral"}` (`internal/baseline/baseline.go:91`).
- **Scheduling:** six power strategies `fast|explore|exploit|rare|balanced|adaptive` (`internal/scheduler/scheduler.go`).
- **Corpus:** content-hash dedup, prioritization, trim (`internal/corpus/corpus.go`).
- **Detection:** signal-aware classification + sanitizer stderr regex
  (`internal/detection/detection.go:54-60`), SHA-256 crash-signature dedup (`internal/fuzz/fuzz.go:386+`).
- **Minimization:** delta debugging (chunk removal + fine pass) (`internal/minimization/minimization.go:20-61`).
- **Safety:** `internal/safety/safety.go` budgets/limits/authorization guard (default `MaxExecutions: 0` = unlimited,
  `:33`), `--dry-run` plan, `-y/--authorized`, target `--authorize`, `--allow-shellout` gate for `.qs` scripts
  (`internal/cli/root.go:159`, `script.go:82`).
- **Sessions/reports:** JSON session store (default `./sessions`), report in terminal/json/yaml/markdown/html
  (`internal/output/output.go:23-27`), `--events` JSONL with lazy file (`internal/cli/app.go:85-156`).
- **Exit codes:** 0/1/2/130 **plus `3` (authorization refused)** — `internal/exitcode/exitcode.go:14-19`, wired at
  `internal/cli/app.go:57-65` (verified live, §10).
- **SecLists:** resolve + list/search without vendoring (`internal/wordlists/wordlists.go`).
- **Scripting:** shared `qyvora-script` engine bound to sekhmet operations (`internal/cli/script.go:80-108`);
  `script build` is parse-only (`script.go:73-79`, verified output).

## 4. Capability status matrix

Statuses: IMPLEMENTED / PARTIALLY IMPLEMENTED / STUBBED / PLANNED / MISSING / UNVERIFIED / NOT APPLICABLE.

| # | Capability | Status | Evidence |
|---|---|---|---|
| 1 | Baseline profiling (warmup + samples, exit/runtime/output variance) | IMPLEMENTED (verified) | `baseline --samples 2 --warmup 1` exit 0, table incl. `coverage kind: behavioral`; `internal/baseline/baseline.go` |
| 2 | Campaign fuzzing (parallel workers, budgets, HUD, session save) | IMPLEMENTED (verified incidentally) | 2-minute campaign 238,121 exec @1,989/s persisted coherent session (§2 #1); `internal/fuzz/fuzz.go` |
| 3 | Crash/hang/anomaly/timeout classification vs baseline | IMPLEMENTED | `isAnomaly` vs `ExitProfile` (`fuzz.go:319-329`); `TestSimFuzzDetectsAndDeduplicatesCrashes` |
| 4 | Sanitizer detection (ASan/UBSan/MSan/TSan/LSan, SEGV) | IMPLEMENTED, text-regex only | `internal/detection/detection.go:54-60`; no sanitizer build/link integration (inferred from absence) |
| 5 | SHA-256 crash-signature dedup | IMPLEMENTED | `fuzz.go:386+`, `Session.AddCrash` (`pkg/models/session.go:88-101`) |
| 6 | Delta-debug minimization | IMPLEMENTED (tested) | `internal/minimization/minimization.go:20-61`; `minimization_test.go` |
| 7 | Replay of a single input | IMPLEMENTED, unexecuted this audit | `internal/cli/replay.go:32-60` incl. interrupt→130 path; not run (would execute target) |
| 8 | 17 mutation operators | IMPLEMENTED with 1 documented no-op | `internal/mutation/mutation.go` `AllOperators`; `splice` no-op (`operators.go`); `lengthMutation` default no-op |
| 9 | "JSON structure" mutation (README claim) | MISSING | no JSON-aware operator in `AllOperators` (verified enumeration) vs `README.md:58-59` |
| 10 | Grammar/structure-aware generation | STUBBED / UNWIRED | `internal/generation/generation.go` exists; `rg 'qyvora-sekhmet/internal/generation'` → **no importers** |
| 11 | Edge/block coverage feedback (README claim) | STUBBED | `baseline.go:91` hardcodes `behavioral`; `config.Strategy` has **zero callers** (`rg config.Strategy` → none) |
| 12 | Behavioral feedback + power scheduling (6 strategies) | IMPLEMENTED | `internal/feedback`, `internal/scheduler/scheduler.go`; **scheduler package has no tests** |
| 13 | Corpus import/crop/list, content-hash dedup, retention cap | IMPLEMENTED (tested) | `internal/corpus/corpus.go`, `corpus_test.go`, `internal/cli/corpus.go` |
| 14 | Sessions + report renderers (terminal/json/yaml/markdown/html) | IMPLEMENTED (verified all 5) | `report --session … -o <fmt>` exit 0 for each; `internal/output/output.go:76-141` |
| 15 | Event stream (`--events`, envelope) | IMPLEMENTED, envelope CONFORMANT | 4-line capture all fields valid (§10); lazy file `internal/cli/app.go:142-156` |
| 16 | Event vocabulary coverage | PARTIALLY IMPLEMENTED | 26 verbs declared (`internal/events/events.go:37-63`), only ~7 ever emitted (§10); read-only commands emit nothing |
| 17 | Exit codes 0/1/2/130 | IMPLEMENTED for flags/unknown root command; **NONCONFORMANT for usage errors raised by cobra validators** | probes: unknown flag→2, unknown root cmd→2; but 15 leaf commands with stray args → **exit 1** (§10, D-3) |
| 18 | Exit code `3` (authorization refused) | IMPLEMENTED (verified live) | unauthorized non-sim target → `exit=3`, "current target is not authorized: probe-noauth" (`app.go:57-65`) |
| 19 | Exit code `3` for unsupported execution backend (spec §1 sekhmet extension) | MISSING / mis-mapped | runner build errors are wrapped as **2**: `fuzz.go:57` (`WrapExitError(2, …)`); verified `fuzz --executions 1` on `--type network` → exit 2 |
| 20 | `version -o json` full metadata set | IMPLEMENTED (verified) | spec §4 field set present (§1); `internal/version` |
| 21 | Capability advertisement | PARTIALLY IMPLEMENTED | 10 records with risk/noise/auth fields but **bare array**, not `contract.CapabilityManifest` (§10) |
| 22 | Canonical `contract.Result` report shape | PARTIALLY IMPLEMENTED / documented variance | reports are raw `models.Session` (spec §6 documents this), but empty collections are `omitempty`-dropped, violating spec §6 rule 3 (§10) |
| 23 | Authorization model (sim auto-grant, `--authorize`, `-y`, runtime refusal) | IMPLEMENTED (verified both outcomes) | `target.go:40-66`, `app.go:57-65`, `safety_test.go:32` |
| 24 | `--dry-run` plan | IMPLEMENTED (verified) | `fuzz --dry-run` exit 0, prints plan without executing (`fuzz.go:44-47,185-193`) |
| 25 | SecLists integration | IMPLEMENTED; environment-conditional | `wordlists list` → exit 2 honest "SecLists not found; set QYVORA_SEKHMET_SECLISTS_DIR or seclists.dir" (verified; SecLists not installed here) |
| 26 | `script` (.qs) engine, shell-out gated | IMPLEMENTED, partially exercised | `script build examples/assess.qs` verified (parse-only plan, exit 0); `script run` **not executed** (its example runs a fuzz campaign) |
| 27 | Interactive TUI / console REPL | NOT APPLICABLE (retired) | unwired `commandTUI`, `//go:build ignore` console; bare invocation → banner+help exit 0 |
| 28 | Self-update | STUBBED (offline by design, mislabeled) | `internal/cli/updates.go:19-32` prints local info; `--update` routes to it (`root.go:109-116`); update channel printed as `github.com/QYVORA/qyvora-Sekhmet` — wrong casing (verified) |
| 29 | Config precedence (flags > `QYVORA_SEKHMET_*` > file > defaults) | IMPLEMENTED, partially dead | `internal/config/config.go:54-78`; `Strategy()`/`BaselineSamples()` never called (dead code) |
| 30 | Coverage instrumentation (AFL++/libFuzzer/kcov/ptrace) | NOT APPLICABLE in this environment | grep clean for all integrations (inferred from absence); see §9 |
| 31 | CI: gofmt+vet+test+clean build | IMPLEMENTED | `.github/workflows/ci.yml:36-47` |
| 32 | CI: race detector, linter, conformance engine | MISSING | `ci.yml` has no `-race`, no `golangci-lint`, no `qyvora-conformance` despite `.golangci.yml` and `Makefile:57 test-race` existing; **`go test -race` fails today (D-8/T-1)** |

## 5. Reconnaissance and discovery findings

- **Targeting is explicit and scoped.** `target set` records name/type/path/endpoint/args/input-file/authorization
  (`internal/cli/target.go:24-79`); `requireTarget()` refuses unauthorized, non-simulation targets with **exit 3**
  before anything executes (`internal/cli/app.go:57-65`, verified). The simulation target auto-grants authorization
  (`target.go:60-62`).
- **Advertised-but-unexecutable target type (P1).** Help offers `--type cli, http, network, simulation`
  (`target.go:73`) and `models.TargetNetwork`/`TargetAuto` exist (`pkg/models/target.go:13-19`), but
  `execution.New` has no case for `network`/`auto` → `execution: unsupported target type "network" for this build`
  (`execution.go:83-107`), returned as **exit 2** from `fuzz`/`baseline` (`fuzz.go:57`, verified). A network target
  can therefore be registered and listed but never run — and the refusal is classified as a usage error.
- **SecLists is resolved, not vendored.** Env/config/conventional paths searched (`internal/wordlists/wordlists.go:14-36`);
  absent installation produces a clean, actionable exit 2 (verified). Categories/files counted without loading.
- **Corpus is a real store:** import from files/dirs, crop to best-N, list; content-hash dedup; retention flag
  (`--retention`, default 100000) on `fuzz` (`internal/cli/fuzz.go:155`, `internal/cli/corpus.go`).
- **Baseline is the discovery mechanism, by design:** warmup + sampled executions produce exit-code/signal/runtime/
  output profiles (`internal/baseline`), and every campaign runs it first (`fuzz.go:63-66`). Gap (P2): baseline is
  always rebuilt per campaign with fixed samples and no cross-session reuse (`--events` shows `baseline.started` each
  run; no "load baseline from session" flag — inferred from flag set).
- **Gap (P2):** no target discovery/asset enumeration of any kind — the `sekhmet.target.discover` capability name
  (`internal/capabilities/capabilities.go:57`) overstates what `target set/list/show` does; discovery means
  "declare and validate" here (honest description in the capability text, misleading ID).
- **Positive:** target/session state is plain JSON (`sessions/targets/targets.json`) with authorization timestamps —
  auditable by an orchestrator.

## 6. Analysis and assessment findings

- **Classification is baseline-relative, not static:** exit codes present in the baseline's `ExitProfile` are
  expected; anything else can be an anomaly (`fuzz.go:319-329`); signals and sanitizer stderr drive crash
  signatures; hangs consume both hang and timeout counters (`fuzz.go:256-261`).
- **Dedup contract is implemented as specified:** `Finding.Fingerprint` (rule/category/classification/title/target/
  attributes, `pkg/models/finding.go:141-160`), `Session.AddFinding/AddCrash` merge evidence by hash and upgrade
  confidence only upward (`pkg/models/session.go:76-113`, `finding.go:155-160`) — matches spec §7.
- **Feedback is behavior-only in practice:** novelty = observed behavioral signature; `coverage_gained` and
  `new_behaviors` are the same counter family (session result of the incidental campaign: both `19,139`), i.e.
  "coverage" is not code coverage. README's "edge / block coverage feeds power scheduling" (`README.md:60-61`) is
  aspirational (§4 #11).
- **Generation seam exists but is dead:** `internal/generation` (TokenGen etc.) is never imported — a ready-made
  structure-aware seed generator that no code path reaches (P2 candidate, §8).
- **Scheduler/corpus concurrency defect (P1, verified):** `go test -race ./internal/fuzz` intermittently fails
  (1/5 direct runs + full-suite run) with **DATA RACE** between `corpus.Store.MarkExecuted`
  (`internal/corpus/corpus.go:154,156` — `seed.Executions++`, `seed.Novelty +=`, `seed.Interestingness =`) and
  `models.Seed.Priority()` reads during sorting (`pkg/models/seed.go:45` via `corpus.go:132-133`). Root cause:
  `Store.Prioritized()` copies seed **pointers** under `RLock`, releases the lock, then sorts outside it
  (`corpus.go:126-137`), while other workers mutate those seeds under the write lock. Consequence: torn scheduling
  priorities and undefined race behavior in the production hot path (`fuzz.go:199,271`). CI cannot see it because it
  does not run `-race` (§13 T-1).
- **Minimization is offline-safe:** delta-debugging reduces a triggering input while preserving the `keep` predicate
  (`minimization.go:20-61`); interrupted runs return 130 (`internal/cli/minimize.go:75`).
- **Gap (P2):** `analyze` filters findings by `--status` but performs no cross-session trend/risk scoring
  (`internal/cli/analyze.go`), and there is no report-level severity aggregation (no `risk_score` equivalent — the
  session model has none).
- **Gap (P3):** `SessionResult.Start/End` are serialized as zero times (`"0001-01-01T00:00:00Z"`) while the session's
  own `start/end` are correct (verified in two saved sessions) — a data-quality bug for consumers reading
  `result.start`/`result.end`.

## 7. Exploitation-related capability findings

**Status: IN-DOMAIN as controlled execution; no offensive exploitation surface beyond the authorized target.**

- The `exploit` tier is sekhmet's campaign tier: `fuzz`, `replay`, `minimize` (`root.go:191`) — all of which drive
  only the **registered** target. There is no exploit delivery, no payload generation, no third-party interaction.
- **Safety wiring is real and tested:** execution budgets/size caps/concurrency limits (`internal/safety/safety.go`,
  `safety_test.go` — `TestExecutionLimit`, `TestInputSizeCap`, `TestAuthorizationGate`, `TestRuntimeLimit`),
  authorization gate before execution (`app.go:57-65`), `--dry-run` (prints plan, exits 0, verified), and
  script shell-out behind `--allow-shellout` / `QYVORA_ALLOW_SHELLOUT` (`root.go:159`, `script.go:82`).
- **HTTP mode is constrained to a declared endpoint** with per-execution timeout, optional `--insecure-tls`
  (explicitly labeled "authorized testing only", `root.go:161`) and max-output caps; no host discovery or
  crawling exists (`internal/execution/http.go`).
- **Process mode never goes through a shell** (`execution.go:110-113`), removing a class of injection problems.
- **Weak points (P1/P2):**
  1. Bare `sekhmet fuzz` starts an **unlimited** campaign immediately against the current target with no
     confirmation, `--executions 0` (unlimited) and no `--runtime` (defaults at `fuzz.go:147-155`), and the safety
     guardian default is `MaxExecutions: 0` = unlimited (`safety.go:33`). Verified live — this is how the audit's
     2-minute campaign started (§2 #1). For an authorized non-sim target the campaign would run until externally
     killed; results still persist, but the default is footgun-shaped (D-7).
  2. The mis-mapped unsupported-backend exit (§4 #19) means automation can not distinguish "fix your command" from
     "this build cannot run this target" — exactly the confusion the exit-code contract exists to prevent.
- **No privilege escalation, persistence, or recon-abuse paths** exist (inferred from absence across `internal/`).

## 8. High-level implementation candidates

High-level = buildable now with existing hardware, dependencies and authorization model (PROMPT §6).

1. **Fix the corpus data race** — sort under the store lock, or snapshot seed metrics into value copies before
   sorting, or make `Seed` metric fields atomic. Small, isolated, testable with `-race` (closes §6/§14 G-1).
2. **Usage-error normalization** — map cobra `NoArgs`/`ExactArgs`/`OnlyValidArgs` validator errors to exit 2
   (one `SetFlagErrorFunc`-style hook on `rootCmd`, plus `ExecuteArgsContext` prefix-matching like jabari/aksum;
   spec §1 already documents the pattern). Closes the largest conformance delta (G-2).
3. **Unsupported-backend → exit 3** — change `fuzz.go:57` (and baseline/replay/minimize equivalents) to wrap
   `execution.New` failures with `exitcode.AuthorizationRefused`-style `3`/`Unsupported`, per spec §1's sekhmet
   extension (G-3).
4. **README/docs regeneration** — the four broken quick-start examples, the console claim, the `JSON structure`
   operator claim, and the generated tier tables in `docs/CLI.md` (G-5, G-6). All fixes are documentation-only.
5. **Wire the generation seam** — import `internal/generation` behind `--strategy structure` and route
   `config.Strategy()` into seed generation, turning two dead modules into the advertised grammar/structure-aware
   fuzzing (G-7).
6. **Enable edge/block feedback for real** — honor `feedback.kind` instead of hardcoding `behavioral`
   (`internal/baseline/baseline.go:91`), or downgrade the README claim (cheapest honest option) (G-7).
7. **Capability manifest wrapper + golden tests** — wrap the existing 10 capability records in
   `{"framework","version","commands":[…]}` and add golden tests for `version -o json`, `capabilities -o json`,
   one event line, and `report -o json` (G-4, T-2).
8. **Safe fuzz defaults** — require `--executions`/`--runtime` or default to a bounded budget (e.g. 100k) unless
   `--unlimited` is passed; the `safety` guardian already exists to carry the limit (G-8).
9. **CI hardening** — add `go test -race`, `golangci-lint` (config already present) and `qyvora-conformance` to
   `ci.yml` (G-9). Note the race test must be fixed first (candidate 1).
10. **Emit the declared event vocabulary** — `report.generated`, `corpus.added/trimmed`, `input.minimized`,
    `finding.discovered`, `hang.detected`, `anomaly.detected` are all declared and documented in spec §3 but never
    emitted; a small emitter at the existing call sites gives agents the lifecycle they were promised (G-10).

## 9. Low-level deferred technique inventory

**DEFERRED — SPECIALISED PREREQUISITES.** Adjacent fuzzing/intelligence techniques that cannot be implemented or
exercised in this environment. Recorded, not silently dropped (PROMPT §6).

| Technique | Objective | Requires | Status | Why deferred / unlock conditions |
|---|---|---|---|---|
| Compiler-coverage guided fuzzing (AFL++/libFuzzer-style edge feedback from real binaries) | True code-coverage guidance | clang/LLVM toolchain, rebuild of every target binary with instrumentation, shared-memory bitmap contract | MISSING → DEFERRED | Targets here are pre-built/stripped or simulation; unlock: instrumented target builds + LLVM toolchain |
| Sanitizer-instrumented targets (ASan/UBSan builds) | Turn the text-regex detector into signal-true detection | Rebuild targets with `-fsanitize=…`, symbolized toolchain | MISSING → DEFERRED | Detector (`detection.go:54-60`) already parses output; unlock: sanitizer builds of targets |
| Binary-only dynamic instrumentation (ptrace/QEMU/Frida-style guided mutation) | Coverage from uninstrumented binaries | ptrace/QEMU user-mode, platform-specific runtimes, root/namespace permissions | MISSING → DEFERRED | No instrumentation backend in `internal/execution` (inferred from absence); unlock: instrumentation backend dependency |
| Kernel / VM fuzzing (syzkaller-style) | Kernel syscall coverage | KVM/QEMU, VM images, syzkaller tooling, kernel build configs | MISSING → DEFERRED | No virtualization in environment; unlock: VM provisioning |
| Distributed / multi-host campaigns (cluster scheduling, shared corpus) | Scale beyond one host | Coordinator service, network transport, shared object storage | MISSING → DEFERRED | Single-process scheduler only (`internal/scheduler`); unlock: coordinator component |
| Stateful/protocol-aware fuzzing with model learning (auto-generated grammar inference from traffic) | Model protocol state machines | Traffic capture, protocol DSL, learning pipeline | MISSING → DEFERRED | static grammar/token generation only (and unwired); unlock: protocol capture + model component |
| Hardware-in-the-loop and RF/wireless protocol fuzzing (CAN, SDR, BLE) | Firmware/RF robustness | Physical interfaces, SDR/CAN hardware, licensed spectrum use | MISSING → DEFERRED | No hardware; unlock: device/SDR provisioning |
| Differential fuzzing across implementations | Consensus bugs | Two+ comparable implementations, oracle harness | MISSING → DEFERRED | single-target runner contract; unlock: oracle framework |
| GPU-accelerated mutation throughput | >10k exec/s on large state spaces | GPU runtime + batch execution design | MISSING → DEFERRED | no GPU in environment; unlock: GPU hardware/runtime |

## 10. Common-layer conformance

Reference: `shared/qyvora-common/contract/contract.go`, `conformance/runner.go`, `knowledge/…/QYVORA-TOOL-OUTPUT-SPEC.md`
§1–§8. Sekhmet does not import the contract (verified: no `qyvora-common` in `go.mod` or any `.go` file) — native
implementation by design.

| Contract element | Sekhmet behaviour | Verdict |
|---|---|---|
| Exit codes 0/1/2/130 (spec §1) | `internal/exitcode/exitcode.go:14-19`; flags/unknown root command → 2 verified; unknown **root** command → 2 verified | CONFORMANT for those probes |
| Spec §1 Rule 1: "usage error MUST NOT be exit 1" | cobra validator errors surface as plain errors → `ExecuteArgsContext` returns **1** (`root.go:118-128`): verified for `version/capabilities/completion/updates/report/crashes/analyze/baseline/fuzz/replay/minimize/target list/session list/corpus list/wordlists list <stray-arg>` (15 invocations) | **NONCONFORMANT (verified)** — D-3/G-2 |
| Exit code `3` (auth refused) | `app.go:57-65`; live: unauthorized cli target → exit 3 | CONFORMANT (verified) |
| Spec §1 sekhmet extension: `3` = unsupported/no execution backend | runner construction failures wrapped as **2** (`fuzz.go:57`); live: `--type network` → exit 2 | **NONCONFORMANT (verified)** — constant is also *named* `AuthorizationRefused`, not `Unsupported` |
| `VersionResult` (spec §4) | full field set incl. `website`/`support`/`built_in` (verified) | CONFORMANT (shape); version **value** `v0.1.0` ≠ tag `v0.10.1` (stamping gap, D-10) |
| `Envelope` JSONL (spec §2) | 4/4 captured lines valid: `schema_version:"1.0"`, RFC3339 `timestamp`, 32-hex `execution_id`, `framework:"sekhmet"`, `level`, dotted `event`, `data` | CONFORMANT (verified) |
| Event vocabulary (spec §3 sekhmet row = 16 verbs) | code declares 26 (`events.go:37-63`); live observed: `scan.started`, `baseline.started`, `baseline.completed`, `scan.completed`; emitted set in code ≈ 7 verbs (`baseline.go`, `fuzz.go`, `script.go`, `fuzz.go:388`). `scan.*` are emitted but **absent from the spec row**; 11 spec-listed verbs (`baseline.samples`, `corpus.*`, `behavior.discovered`, `hang.detected`, `anomaly.detected`, `input.minimized`, `finding.discovered`, `evidence.collected`, `coverage.updated`, `report.generated`) are **never emitted** | PARTIALLY CONFORMANT (both directions) — G-10 |
| Event stream lifecycle | file destinations are lazy (`app.go:142-156`); read-only commands emit nothing, so `--events file` is never created for them (verified: `session list`, `report`) | PARTIALLY CONFORMANT — documented design comment, but agents get no completion event for read commands |
| `CapabilityManifest` (spec/`contract.CapabilityManifest`) | `capabilities -o json` → bare array of 10 objects (verified) | **NONCONFORMANT (shape)**, exit 0 — G-4 |
| Report JSON (spec §6 sekhmet section) | raw `models.Session` as documented; **but** `findings`/`crashes`/`errors`/`end` carry `omitempty` (`pkg/models/session.go:44-56`) so empty collections are **omitted**, contradicting spec §6 rule 3 ("empty collections serialize as `[]`, never `null`"); `session list -o json` on an empty store emits literal **`null`** (verified) | **NONCONFORMANT (verified)** on rule 3; shape otherwise as documented |
| Dedup contract (spec §7 sekhmet) | fingerprint + evidence merge + upward-only confidence (`session.go:76-113`, `finding.go:155-160`) | CONFORMANT (implemented as specified) |
| Severity/confidence vocabularies (spec §5) | `critical|high|medium|low|informational`, confidence `confirmed|high|medium|low` (`pkg/models/confidence.go`, `finding.go`) | CONFORMANT |
| TUI retirement (`conformance/runner.go`) | shipped `bin/sekhmet` is stripped → runner cannot inspect symbols | CONFORMANT in substance (unwired by code), UNVERIFIED by runner on release artifacts |

**Conformance engine run (performed):**
`go run ./cmd/qyvora-conformance -json -skip-build -bin "…/bin/nzinga:nzinga,…/bin/sekhmet:sekhmet"` →
**0/2 conformant** (exit 1). Both frameworks: `version-semver PASS`, `version-exit0 PASS`, `usage-exit-2 PASS`,
`unknown-cmd-exit-2 PASS`, `capabilities PASS`, `tui-retired FAIL ("cannot inspect symbols: … no symbol section")`.
The sole failing gate for both is the stripped-binary limitation (`.goreleaser.yaml:45` / `Makefile:20` `-s -w`), not
behaviour; the runner's usage-exit gate probes only *unknown flags/unknown root command*, which is why the exit-1
validator defect above is invisible to it (see §13 T-1).

**Duplication notes.** `internal/{events,exitcode,errors,version}` and the `hiddenAlias` pattern re-implement shared
contracts independently — per the native-implementation rule, but these are where fleet drift appears. Sekhmet is the
only audited framework that *implements* exit 3, and the only one whose spec text promises an `Unsupported` semantic
its code does not have.

## 11. AI orchestration readiness

| Requirement | Status | Evidence |
|---|---|---|
| Stable framework identity | CONFORMANT | `version -o json` carries `framework:"sekhmet"` + full build metadata |
| Version/compatibility metadata | PARTIALLY CONFORMANT | shape OK; `Makefile:15` stamps `v0.1.0` while the repo is tagged `v0.10.1` (verified) |
| Machine-readable capabilities | PARTIALLY CONFORMANT | 10 records with `risk`, `noise_level`, `authorization_required`, `confirmation_required`, `reversible`, `output_schema`; bare array, not a manifest (§10) |
| Explicit operation inputs/outputs | PARTIALLY CONFORMANT | per-capability `output_schema`; no command-level `inputs`/`deterministic` flags vs `contract.Command` |
| Predictable exit codes | **PARTIALLY CONFORMANT (worst gap)** | 0/1/2/3 verified as intended for flags/root-commands/auth; **15 usage probes exit 1**, unsupported-backend exits 2 instead of 3 (§10); tier-parent typo → help + **exit 0** (`target/session/corpus/wordlists/recon/exploit bogus`, verified) |
| Structured results and evidence | PARTIALLY CONFORMANT | rich session/finding/evidence model, fingerprint dedup; empty collections dropped, `null` for empty lists, zero `result.start/end` |
| Prerequisite/authorization declaration | CONFORMANT | capabilities advertise auth/risk; runtime refusal is early, explicit, exit 3 |
| Timeouts / cancellation | PARTIALLY CONFORMANT | `signal.NotifyContext` (`root.go:92`), per-execution `--timeout`, `context` cancel → 130 in replay/minimize (`replay.go:60`, `minimize.go:75`); interrupt→130 **UNVERIFIED at runtime** |
| Safe execution defaults | PARTIALLY CONFORMANT | auth gates, dry-run, budgets exist — but default campaign is **unlimited** with no confirmation (D-7) |
| Read-only vs state-changing distinction | CONFORMANT | `changes_state`-equivalent via `authorization_required`/`reversible`; only session/target/corpus files are written |
| Honest unsupported reporting | NONCONFORMANT | unsupported backend reported as usage error 2 with a clear message; needs 3 (§4 #19) |
| Result/event correlation | CONFORMANT | one `execution_id` per run across all events (verified 4/4) |
| Shared common-layer compatibility | PARTIALLY CONFORMANT | native implementation by design; four shape/semantic deltas in §10 |

**Recommended architecture (proposal only, not implemented):** (a) per-repo contract golden tests that unmarshal
`version -o json`, `capabilities -o json`, one event line and `report -o json` against `qyvora-common/contract` types;
(b) one exit-code helper covering `2` for validator errors and `3` for unsupported/auth-refusal (naming aligned with
the spec); (c) orchestrators must currently accept the bare capability array and `omitempty` reports as documented
variance until the wrappers land.

## 12. CLI, repository-layout, and documentation findings

**CLI (verified).** Global flags: `--allow-shellout`, `-y/--authorized`, `-c/--config`, `--dry-run`, `--events`,
`--insecure-tls`, `--json`, `-o/--output` (`terminal,json,markdown,html,yaml`), `-q`, `-v`, `--timeout`, `--update`
(`root.go:152-161`). Positive behaviours: unknown flag → 2, unknown root command → 2 with clean `unknown command …`,
invalid `-o` → 2 listing valid values, missing `--session` → 2 (`report`/`crashes`/`analyze`), SecLists missing → 2
with an actionable message, unauthorized target → 3, empty `exploit`-style probes handled, `--events stdout` reroutes
human output to stderr to keep stdout pure (`app.go:92-99`).

**Findings:**

- **D-1 (P1) `README.md` quick start is entirely broken — every example exits 2.** Verified:
  `target set --kind process --cmd …` (`README.md:104`) → `unknown flag: --kind`;
  `baseline --target ./target` (`:107`) → `unknown flag: --target`;
  `fuzz --target … --runs … --jobs …` (`:110`) → `unknown flag: --target`;
  `report --session … --format json` (`:114`) → `unknown flag: --format`;
  the simulation block (`:120-122`) fails the same way. Real flags are `--type/--path`, no `--target`/`--runs`/
  `--jobs` (it's `--workers`/`--executions`), and `-o/--output`.
- **D-2 (P1) `README.md` claims an interactive console that does not exist.** `README.md:72-73` ("run `sekhmet` with
  no args to work interactively") vs verified bare invocation: banner + help, exit 0 (`root.go:141-143`); the REPL
  source is build-tagged out (`internal/cli/console.go:1`) and `commandTUI()` is never registered. Related: README
  `:58-59` claims a "JSON structure" mutation operator that `AllOperators` does not contain, and `:60-61` claims
  edge/block coverage that `baseline.go:91` never enables.
- **D-3 (P1) usage errors exit 1 instead of 2.** Every `cobra.NoArgs`/`ExactArgs` leaf command returns 1 when given
  a stray argument (15 verified invocations; mechanism `root.go:118-128`). Violates spec §1 Rule 1 (§10).
- **D-4 (P1) advertised `--type network` cannot execute and mis-reports its refusal.** `target set --type network`
  succeeds; `fuzz`/`baseline` then fail with `unsupported target type` at **exit 2** (verified; §4 #19, §5).
- **D-5 (P1) `docs/CLI.md` documents commands that do not exist.** Generated tier × command cross-product:
  `sekhmet recon fuzz|tui|capabilities|version|report|…` and `exploit tui|version|…` print the **parent tier help with
  exit 0** (verified for `recon capabilities`, `recon version`, `recon fuzz`); `report` section says
  "_No report-tier modules._" (`docs/CLI.md:57`) although the tier holds `report` and `crashes`;
  the utilities table lists `sekhmet tui` (`:72`, not registered) and a malformed row `` `docs script <file>` ``.
- **D-6 (P2) tier-parent typos exit 0.** `target/session/corpus/wordlists/recon/exploit <bogus>` → parent help,
  exit 0 (verified) — scripted typos look like success.
- **D-7 (P2) bare `fuzz` starts an unlimited campaign with no confirmation.** Defaults `--executions 0`,
  `--runtime 0`, `--workers 4` (`fuzz.go:147-155`) + `safety.go:33` unlimited; verified incidentally (§2 #1).
- **D-8 (P2) CI omits race, lint and conformance.** `ci.yml:36-47` = gofmt/vet/test/clean build only, while
  `.golangci.yml` and `Makefile:57 test-race` exist — and `go test -race` **fails today** (§13 T-1).
- **D-9 (P2) generated docs vs reality beyond CLI.md:** `updates` prints channel
  `github.com/QYVORA/qyvora-Sekhmet` (`internal/cli/updates.go:21`, verified) — wrong repository casing
  (`qyvora-sekhmet` per module path); `--update` implies an update path but only prints offline info.
- **D-10 (P3) `Makefile:15` `VERSION ?= v0.1.0`** vs git tag `v0.10.1` — local builds misreport the code under test
  (release path is correct via goreleaser, `.goreleaser.yaml:43-45`).
- **D-11 (P3) machine-output hygiene:** empty `session list -o json` → `null`; `SessionResult.start/end` zero
  times; target IDs use the `sess-` prefix (`pkg/models/session.go:59` helper reused for targets — cosmetic but
  confusing in `sessions/targets/targets.json`).

**Repository layout.** Standard fleet layout: `cmd/ internal/ pkg/ docs/ examples/ sessions/ bin/ assets/ scripts/`,
root `README.md`, `CHANGELOG.md`, `SECURITY.md`, `GOVERNANCE.md`, `CONTRIBUTING.md`, `SUPPORT.md`,
`CODE_OF_CONDUCT.md`, `NOTICE`, `LICENSE`, `Makefile`, `.goreleaser.yaml`, installers (`install.sh`/`install.ps1`),
`.github/workflows/{ci,release}.yml`. `bin/`, `sessions/`, `reports/` gitignored (`.gitignore:4,8,10`).
**Stray local artifact:** root `continue_AI.sh` (single line invoking `opencode -s ses_…`) is untracked and
gitignored (`.gitignore:22`) — local automation left in a product repo (P3, disclose-or-remove).
Release workflow has a real asset-completeness gate (`release.yml` "Verify the draft contains every asset").

## 13. Testing and verification findings

**Performed on 2026-10-21 (this audit):**

| Check | Command | Result |
|---|---|---|
| Unit tests | `go test ./... -count=1 -timeout 180s` | **exit 0**; 12 `ok`, 15 `no test files`, no `FAIL`; 15 test files / 1,246 test lines |
| Vet | `go vet ./...` | **exit 0** |
| Format | `gofmt -l .` | **0 files** (clean) |
| **Race detector** | `go test -race -count=1 ./...` | **exit 1 — DATA RACE** in `internal/fuzz` (`TestSimFuzzDetectsAndDeduplicatesCrashes`); race between `corpus.Store.MarkExecuted` (`corpus.go:154,156`) and `Seed.Priority` reads (`seed.go:45`) during `Prioritized` sort outside the lock (`corpus.go:126-137`). Intermittent: 1/5 direct package runs + 1 full-suite run reproduced (2 warnings each) |
| Race (nzinga, for contrast) | `go test -race ./...` | exit 0, 15 `ok` — sekhmet's race is its own |
| Conformance engine | `qyvora-conformance -skip-build -bin …` | 0/2 conformant; only `tui-retired` fails (stripped binaries); see §10 |
| Event envelope | `baseline --samples 2 --events …` → 4 JSONL lines parsed field-by-field | all conform |
| Exit-code matrix | ~70 invocations (positive + negative) | as recorded in §4/§10/§12 |
| `script` engine | `script build examples/assess.qs` | exit 0, correct parse-only plan; `script run` **not executed** (would fuzz) |

**Test coverage shape.** Tested: `banner, cli, corpus, detection, execution, feedback, fuzz, minimization, mutation,
output, safety, session`. **No tests** for core packages: `baseline` (profiling logic), `scheduler` (all six
strategies), `corpus`-adjacent `generation`, `config`, `target`, `capabilities`, `models` (incl. the dedup contract
in `session.go`/`finding.go`), `events`, `exitcode`, `wordlists`, `console`, `logger`, `errors`, `version`, `cmd`.

**CI/CD (read, not executed).** `ci.yml`: gofmt → vet → `go test ./...` → clean `CGO_ENABLED=0` build.
`release.yml`: goreleaser + asset-completeness gate. Neither runs `-race`, lint, or `qyvora-conformance`.

**Gaps (prioritised):**

- **T-1 (P1) The race detector failure is invisible to CI and to the conformance runner.** The hot path has a real
  data race (§6), CI does not run `-race` (D-8), and `make test-race` (Makefile:57) is unused. Fix candidate 1 must
  land before CI can enable it.
- **T-2 (P2) No golden/fixture tests for machine output.** No test asserts `version -o json`, `capabilities -o json`,
  an event line, or `report -o json` shape — which is why the manifest, vocabulary and `omitempty` deltas went
  undetected.
- **T-3 (P2) Untested core:** `baseline` and `scheduler` (the two modules that define campaign correctness) and
  `models` (the dedup contract the spec certifies) have no tests.
- **T-4 (P3) Interrupt path untested** (exit 130 + event flush): wiring present (`root.go:92`, `replay.go:60`,
  `minimize.go:75`), never exercised → **UNVERIFIED**.
- **T-5 (P3) No live HTTP campaign test** (only the sim runner is exercised in tests, `execution/sim_test.go`,
  `fuzz_test.go`); HTTP runner behaviour under fuzz load is **UNVERIFIED** (audit constraint: no live targets).

## 14. Prioritized gaps and recommendations

| ID | P | Gap | Why it matters | Evidence | Prerequisite / next step |
|---|---|---|---|---|---|
| G-1 | P1 | Data race in corpus prioritization (hot path) | Undefined behaviour under concurrency; corrupted scheduling priorities; would block CI race-gating | `corpus.go:126-137` vs `:154,156`; `-race` repro (§13) | Sort under lock or snapshot metrics; add a `-race` corpus/scheduler test |
| G-2 | P1 | Usage errors from cobra validators exit 1 (spec §1 Rule 1) | Orchestrators retry/abort on "tool failed" instead of "fix invocation" | 15 probes; `root.go:118-128` | Map validator errors to 2 (pattern already documented in spec §1 for jabari/aksum) |
| G-3 | P1 | Unsupported execution backend exits 2, not 3; constant named for auth only | Skips-vs-retries decision impossible; spec §1 sekhmet extension unmet | `fuzz.go:57`; live network-type probe | Wrap `execution.New` errors with `3`/`Unsupported` |
| G-4 | P1 | `capabilities -o json` is a bare array of 10, not `CapabilityManifest` | Manifest-driven discovery can't read `framework`/`version`/`commands` | live capture; `contract.CapabilityManifest` | Wrap existing records + golden test |
| G-5 | P1 | README quick start 100% broken (5 examples, all exit 2) | First-contact failure for every new user/agent | `README.md:104,107,110,114,120-122` | Correct flags; regenerate from `--help` |
| G-6 | P1 | `docs/CLI.md` documents non-existent commands; report tier claimed empty | Operators script commands that silently print help (exit 0) | `docs/CLI.md:21,28,50,57,72`; verified | Regenerate from the real command tree; drop TUI rows |
| G-7 | P2 | Advertised capabilities that are dead: JSON-structure operator, edge/block coverage, generation engine, `config.Strategy` | Overstates maturity; orchestrators may select unsupported strategies | `README.md:58-61`; `baseline.go:91`; `rg generation` no importers; `rg config.Strategy` none | Wire or downgrade each claim |
| G-8 | P2 | Bare `fuzz` = unlimited, no confirmation | Runaway campaigns on authorized targets; surprising resource use | `fuzz.go:147-155`, `safety.go:33`, §2 #1 | Default budget + `--unlimited` opt-in |
| G-9 | P2 | CI lacks race/lint/conformance despite configs existing | Contract and concurrency regressions ship silently | `ci.yml:36-47`; `.golangci.yml`; `Makefile:57` | Land G-1, then enable all three in CI |
| G-10 | P2 | Event vocabulary: 11 spec-listed verbs never emitted; `scan.*` emitted but undocumented | Agents keying on lifecycle events never fire | `events.go:37-63` vs `rg emitEvent` vs spec §3 | Emit at existing call sites; update spec row |
| G-11 | P2 | Report hygiene: empty collections dropped (`omitempty`), `null` for empty lists, zero `result.start/end` | Violates spec §6 rule 3; consumers branch on missing keys | `session.go:21-56`; live `null`; saved sessions | Remove `omitempty` on collections; stamp result times |
| G-12 | P3 | Tier-parent unknown subcommand → help + exit 0 | Typos look like success | `recon/exploit/target/session/corpus/wordlists bogus` → 0 | Return 2 for unknown subcommands |
| G-13 | P3 | Doc/metadata drift: `updates` channel casing, `--update` no-op, `VERSION ?= v0.1.0` vs tag `v0.10.1`, stray `continue_AI.sh` | Trust and consistency for automation reading build info | `updates.go:21`; `Makefile:15`; `.gitignore:22` | Fix text; derive version from `git describe` |
| G-14 | P3 | No cross-session analysis; target IDs reuse `sess-` prefix | Trends unavailable; ID semantics confusing | `internal/cli/analyze.go`; `session.go:59` | Design-only; separate ID prefix |

Priority scheme per PROMPT §7: P0 critical (none found — the data race is serious but has no demonstrated memory
corruption or security impact in these field types; no authorization bypass), P1 high, P2 medium, P3 low.
**Priority is not implementation authorization.**

## 15. Dependencies, prerequisites, and unresolved questions

**Dependencies (declared).** Go ≥1.26.5 (toolchain go1.27.1 used), `qyvora-script v0.1.0`, `qyvora-tui v0.11.0`
(brand helpers), `spf13/cobra v1.10.2`, `spf13/viper v1.21.0`, `go.yaml.in/yaml/v3`, `chzyer/readline`,
`fatih/color`, `muesli/termenv`. Optional external resources: a SecLists installation
(`QYVORA_SEKHMET_SECLISTS_DIR`/`SECLISTS`/conventional paths), API keys none.

**Prerequisites for live use.** An authorized target (`--authorize` on `target set`, or `-y` at run time; simulation
targets auto-grant); a reachable local binary or HTTP endpoint; `--allow-shellout` (or `QYVORA_ALLOW_SHELLOUT=1`)
for `.qs` scripts using `exec`/`run.cmd`.

**Unresolved questions (documented uncertainty, not findings):**

1. Whether `script run` enforces the shell-out gate exactly as documented — `script build` verified (parse-only);
   `script run` **not executed** because its shipped example runs a campaign (§4 #26).
2. Whether the HTTP runner behaves correctly under campaign load (timeouts, keep-alive, large bodies) —
   **UNVERIFIED**, no live endpoint was fuzzed (audit constraint).
3. Interrupt → exit 130 with event flush — wiring present, behaviour **UNVERIFIED** (T-4).
4. How often the corpus race manifests in production workloads (1/5 under `-race` here; unmeasured without race
   instrumentation) — flagged, not quantified.
5. Whether the conformance runner can ever check `tui-retired` on release artifacts — impossible while release
   builds are stripped (`-s -w`); affects every fleet tool (shared with `qyvora-nzinga/AUDIT.md` §10).

## 16. No-feature-implementation statement

**No feature implementation was performed during this audit.** No production code, test, configuration, dependency,
build script, documentation file or asset was created, modified or deleted. The framework was not refactored, no
missing command or technique was implemented, no common-layer or conformance-engine change was made, and no planned
capability was recorded as implemented. The repository's clean `main` at `680a745` (tag `v0.10.1`) was preserved
unchanged; `git status` was clean at audit start and audit end. The only repository addition is this `AUDIT.md`.
Audit runtime artifacts were directed to `/tmp/opencode/…` except the gitignored `sessions/` file disclosed in §2 #1.
All recommendations in §8, §10, §11 and §14 remain recommendations pending an explicit implementation phase.
