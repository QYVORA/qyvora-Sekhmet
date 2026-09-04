# SEKHMET Makefile
#
# High-performance, baseline-aware, feedback-driven fuzzing and
# vulnerability-discovery framework. See README.md for usage.

BINARY ?= sekhmet
PREFIX ?= /usr/local
DESTDIR ?=

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
USER    ?= $(shell id -u -n)

LDFLAGS := -s -w \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Version=$(VERSION) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Commit=$(COMMIT) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Date=$(DATE) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.BuildUser=$(USER)

BINDIR := $(DESTDIR)$(PREFIX)/bin

.PHONY: all build test test-race vet fmt check install uninstall clean

all: build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/sekhmet

test:
	go test ./... -count=1 -timeout 60s

test-race:
	go test -race ./... -count=1 -timeout 120s

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal pkg

check: fmt vet test

install: build
	install -d $(BINDIR)
	install -m 0755 bin/$(BINARY) $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

clean:
	rm -rf bin
