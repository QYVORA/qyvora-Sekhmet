# Security Model

SEKHMET is an **authorized** fuzzing and vulnerability-discovery framework.
This document describes the authorization and scoping model that keeps it
safe to operate by default.

## Core principle

SEKHMET never discovers, enumerates, or reaches out to unknown targets. Every
campaign runs against a target the operator declares explicitly. The operator
must have explicit authorization to test that target.

## Target scoping

- **Process targets** are scoped to the declared binary path. Inputs are
  delivered via commit-time `{fuzz}` / `{stdin}` templates; arguments outside
  the templates are treated as constants. No command shell is used, so inputs
  cannot reach a shell.
- **HTTP targets** likewise require an explicit scheme/host/port/path and an
  explicit authorization acknowledgement.
- The **simulation target** is a deterministic in-process model used for CI,
  tests and learning; it makes no external calls.

## Authorization gate

Remote/network targets require interaction-level authorization before any
execution. `--dry-run` audits a campaign (registration, budgets, corpus,
planned executions) without executing anything.

## Event integrity

All findings and runs emit evidence through the QYVORA JSONL event stream with
the `{schema_version, timestamp, execution_id, framework, level, event, data}`
envelope, `framework: "sekhmet"`. Sessions persist every classified result,
so findings are reproducible and auditable.

## Fingerprinting & deduplication

Findings are fingerprinted by SHA-256 (normalized stderr + signal + exit
class) so identical root causes collapse to a single unique finding rather
than thousands of near-identical crash logs.

## Reporting a vulnerability

See [SECURITY.md](../SECURITY.md). Report privately; do not open a public
issue.
