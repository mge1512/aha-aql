# generated from spec: aha-aql.spec.md sha256:b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30

BINARY      := mcp-server-aha-aql
VERSION     := 0.20.1
SPEC_SHA256 := b5064b9ba50274af776104231d9794d500d571b98535d7bff53fa730836f6f30
PREFIX      ?= /usr
BINDIR      ?= $(PREFIX)/bin
MANDIR      ?= $(PREFIX)/share/man
UNITDIR     ?= $(PREFIX)/lib/systemd/system
GO          ?= go
LDFLAGS     := -X main.serverVersion=$(VERSION)

export CGO_ENABLED := 0

.PHONY: all build test install clean man vendor-tarball

all: build

# vendor/ is a directory, not a phony target. This rule runs when vendor/
# is missing or older than go.mod or go.sum.
vendor: go.mod go.sum
	$(GO) mod vendor
	touch vendor

build: vendor
	$(GO) build -mod=vendor -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

test: vendor
	$(GO) vet -mod=vendor ./...
	$(GO) test -mod=vendor -count=1 -timeout 20m ./independent_tests/...

man: $(BINARY).1

$(BINARY).1: $(BINARY).1.md
	pandoc $(BINARY).1.md -s -t man -o $(BINARY).1

install: build man
	install -D -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -D -m 0644 $(BINARY).1 $(DESTDIR)$(MANDIR)/man1/$(BINARY).1
	install -D -m 0644 $(BINARY).service $(DESTDIR)$(UNITDIR)/$(BINARY).service

# Generate the vendor tarball for OBS/RPM builds; run once per release.
vendor-tarball: vendor
	tar -cJf $(BINARY)-$(VERSION)-vendor.tar.xz vendor/

clean:
	rm -f $(BINARY)
