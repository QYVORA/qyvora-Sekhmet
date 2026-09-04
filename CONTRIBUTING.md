# Contributing

Thanks for contributing to SEKHMET.

## Code of conduct

By participating you agree to abide by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Project scope

SEKHMET is an **authorized**, baseline-aware fuzzing and vulnerability
discovery framework. It is deliberately **not** a broad scanner, an automated
attack tool, or a persistence / exploitation-delivery framework. Contributions
that push against those boundaries will be declined even if technically
impressive. Every high-level operation requires explicit target authorization.

## Development setup

Requirements: Go 1.26+, `mage`/`make`.

```sh
git clone https://github.com/QYVORA/qyvora-Sekhmet.git
cd qyvora-sekhmet
make check        # gofmt + vet + test
make test-race    # full suite with the race detector
make build        # build bin/sekhmet
```

## Before opening a PR

- Run `make check` and `make test-race`; both must pass.
- Keep `golangci-lint` clean (`golangci-lint run` → 0 issues).
- Add/adjust tests for new behaviour; test packages must pass under `-race`.
- Preserve the QYVORA event envelope and `framework: "sekhmet"` in any output.
- One logical change per commit; rebase onto the latest `main`.

## Git workflow

Commits follow the repo's conventional style (lowercase, imperative subject,
Fixes/Refs trailer referencing issues). Push feature branches and open a pull
request; CI (format, vet, race tests, lint, build) must be green.
