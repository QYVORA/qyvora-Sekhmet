# SEKHMET Makefile
#
# High-performance, baseline-aware, feedback-driven fuzzing and
# vulnerability-discovery framework. See README.md for usage.

BINARY ?= sekhmet
PREFIX ?= /usr/local
DESTDIR ?=

# The common contract requires a semantic version and rejects "dev", and an
# install from source stamps whatever VERSION holds. Defaulting it to "dev"
# therefore shipped a binary that failed its own version check, so the
# default is the released version. Override for a real build:
#   make install-user VERSION=v1.2.3
VERSION ?= v0.1.0
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
USER    ?= $(shell id -u -n)

LDFLAGS := -s -w \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Version=$(VERSION) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Commit=$(COMMIT) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.Date=$(DATE) \
	-X github.com/QYVORA/qyvora-sekhmet/internal/version.BuildUser=$(USER)

# --- install layout ------------------------------------------------------
# System-wide install (default PREFIX=/usr/local, typically needs root):
#   /usr/local/bin/sekhmet                          command
#   /usr/local/share/applications/sekhmet.desktop   desktop entry
#   /usr/local/share/icons/hicolor/512x512/apps/sekhmet.png
#   /usr/local/share/pixmaps/sekhmet.png
# User install (make install-user) mirrors the same layout under ~/.local.

ICON    := assets/sekhmet.png
DESKTOP := assets/sekhmet.desktop

BINDIR    := $(DESTDIR)$(PREFIX)/bin
ICONDIR   := $(DESTDIR)$(PREFIX)/share/icons/hicolor/512x512/apps
PIXMAPDIR := $(DESTDIR)$(PREFIX)/share/pixmaps
APPDIR    := $(DESTDIR)$(PREFIX)/share/applications

USERBIN    := $(HOME)/.local/bin
USERICON   := $(HOME)/.local/share/icons/hicolor/512x512/apps
USERPIXMAP := $(HOME)/.local/share/pixmaps
USERAPP    := $(HOME)/.local/share/applications

.PHONY: all build test test-race vet fmt check install install-user uninstall uninstall-user clean

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
	$(MAKE) install-data

install-data:
	install -d $(ICONDIR) $(PIXMAPDIR) $(APPDIR)
	install -m 0644 $(ICON) $(ICONDIR)/sekhmet.png
	install -m 0644 $(ICON) $(PIXMAPDIR)/sekhmet.png
	sed -e 's|@PREFIX@|$(PREFIX)|g' $(DESKTOP) > $(APPDIR)/sekhmet.desktop
	chmod 0644 $(APPDIR)/sekhmet.desktop
	update-desktop-database $(APPDIR) 2>/dev/null || true
	gtk-update-icon-cache -f $(DESTDIR)$(PREFIX)/share/icons/hicolor 2>/dev/null || true
	@echo "sekhmet installed to $(BINDIR) with icon and desktop entry."

install-user: build
	install -d $(USERBIN)
	install -m 0755 bin/$(BINARY) $(USERBIN)/$(BINARY)
	install -d $(USERICON) $(USERPIXMAP) $(USERAPP)
	install -m 0644 $(ICON) $(USERICON)/sekhmet.png
	install -m 0644 $(ICON) $(USERPIXMAP)/sekhmet.png
	sed -e 's|@PREFIX@|$(HOME)/.local|g' $(DESKTOP) > $(USERAPP)/sekhmet.desktop
	chmod 0644 $(USERAPP)/sekhmet.desktop
	update-desktop-database $(USERAPP) 2>/dev/null || true
	gtk-update-icon-cache -f $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@echo "sekhmet installed to $(USERBIN) with icon and desktop entry."
	@echo "Add $$HOME/.local/bin to your PATH if it is not already there."

uninstall:
	rm -f $(BINDIR)/$(BINARY)
	rm -f $(ICONDIR)/sekhmet.png $(PIXMAPDIR)/sekhmet.png $(APPDIR)/sekhmet.desktop
	update-desktop-database $(APPDIR) 2>/dev/null || true
	gtk-update-icon-cache -f $(DESTDIR)$(PREFIX)/share/icons/hicolor 2>/dev/null || true

uninstall-user:
	rm -f $(USERBIN)/$(BINARY)
	rm -f $(USERICON)/sekhmet.png $(USERPIXMAP)/sekhmet.png $(USERAPP)/sekhmet.desktop
	update-desktop-database $(USERAPP) 2>/dev/null || true
	gtk-update-icon-cache -f $(HOME)/.local/share/icons/hicolor 2>/dev/null || true

clean:
	rm -rf bin