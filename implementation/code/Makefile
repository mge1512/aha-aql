# generated from spec: aha-aql.spec.md sha256:9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262

SPEC_SHA256 := 9f8c797a0d3d12f04a911db2be76d267e525060106645483faab751b2e587262
BINARY      := mcp-server-aha-aql
VERSION     := 0.13.0
PREFIX      ?= /usr
BINDIR      ?= $(PREFIX)/bin
MANDIR      ?= $(PREFIX)/share/man
UNITDIR     ?= $(PREFIX)/lib/systemd/system
DESTDIR     ?=
GOFLAGS     ?= -mod=vendor
LDFLAGS     := -X main.serverVersion=$(VERSION)

.PHONY: all build test install clean man vendor-tarball

all: build

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

test: build
	CGO_ENABLED=0 go vet $(GOFLAGS) . ./independent_tests/...
	CGO_ENABLED=0 go test $(GOFLAGS) -count=1 -timeout 15m ./independent_tests/...

man: $(BINARY).1

$(BINARY).1: $(BINARY).1.md
	pandoc $(BINARY).1.md -s -t man -o $(BINARY).1

install: build
	install -D -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -D -m 0644 $(BINARY).1 $(DESTDIR)$(MANDIR)/man1/$(BINARY).1
	install -D -m 0644 $(BINARY).service $(DESTDIR)$(UNITDIR)/$(BINARY).service

# Vendor tarball for OBS builds that keep vendor/ out of the source tarball.
vendor-tarball:
	go mod vendor
	tar -cJf $(BINARY)-$(VERSION)-vendor.tar.xz vendor/

clean:
	rm -f $(BINARY) $(BINARY)-$(VERSION)-vendor.tar.xz
