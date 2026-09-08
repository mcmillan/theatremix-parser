# theatremix-parser — build, test and package.
#
#   make            build bin/theatremix-parser for this machine
#   make check      gofmt + go vet + go test (what CI runs)
#   make dist       package a release archive for GOOS/GOARCH (default: host)
#   make dist-all   package every release target
#   make release    bump VERSION, commit, tag v<VERSION> and push
#   make help       list all targets
#
# Versioning: the VERSION file holds a single integer; releases are tagged
# v<VERSION>. Builds not made from exactly that tag are stamped
# <VERSION>-dev.<sha>[.dirty] so they cannot be mistaken for a release.

BINARY  := theatremix-parser
PKG     := ./cmd/$(BINARY)

FILE_VERSION := $(strip $(shell cat VERSION))
ifeq ($(shell printf '%s' '$(FILE_VERSION)' | grep -Ec '^[0-9]+$$'),0)
$(error VERSION file must contain a single integer, got '$(FILE_VERSION)')
endif
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
ON_TAG  := $(shell git describe --tags --exact-match 2>/dev/null)
DIRTY   := $(shell git diff --quiet 2>/dev/null || echo .dirty)
ifeq ($(ON_TAG),v$(FILE_VERSION))
VERSION ?= $(FILE_VERSION)
else
VERSION ?= $(FILE_VERSION)-dev.$(GIT_SHA)$(DIRTY)
endif

GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXT     = $(if $(filter windows,$(GOOS)),.exe,)

TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

DIST_NAME = $(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH)
DIST_DIR  = dist/$(DIST_NAME)

.PHONY: all build install test vet fmt fmt-check check golden dist dist-all clean help \
        version verify-tag bump release

all: build

version: ## Print the version a build would be stamped with
	@echo $(VERSION)

build: ## Build bin/theatremix-parser (honours GOOS/GOARCH/VERSION)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY)$(EXT) $(PKG)

install: ## Install the CLI into $(go env GOPATH)/bin
	CGO_ENABLED=0 go install $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PKG)

test: ## Run the test suite
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format Go sources in place
	gofmt -w .

fmt-check: ## Fail if any Go source is not gofmt-clean
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then printf '%s\n' "$$out" "gofmt: files need formatting"; exit 1; fi

check: fmt-check vet test ## gofmt + vet + test — what CI runs

golden: ## Regenerate the CLI golden JSON after a reviewed output change
	go test ./cmd/... -update -run TestGolden

dist: ## Package $(DIST_NAME).tar.gz (or .zip on Windows) into dist/
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(BINARY)$(EXT) $(PKG)
	cp README.md LICENSE $(DIST_DIR)/
ifeq ($(GOOS),windows)
	cd dist && rm -f $(DIST_NAME).zip && zip -qr $(DIST_NAME).zip $(DIST_NAME)
else
	cd dist && tar -czf $(DIST_NAME).tar.gz $(DIST_NAME)
endif
	rm -rf $(DIST_DIR)

dist-all: ## Package every target in TARGETS
	@for t in $(TARGETS); do \
		$(MAKE) --no-print-directory dist GOOS=$${t%/*} GOARCH=$${t#*/} VERSION=$(VERSION) || exit 1; \
	done

clean: ## Remove bin/ and dist/
	rm -rf bin dist

verify-tag: ## Fail unless TAG (default: the tag at HEAD) equals v<VERSION>
	@tag="$(or $(TAG),$(ON_TAG))"; \
	if [ "$$tag" != "v$(FILE_VERSION)" ]; then \
		echo "tag '$$tag' does not match VERSION file (v$(FILE_VERSION))" >&2; exit 1; \
	fi; echo "tag $$tag matches VERSION"

bump: ## Increment VERSION in place (no commit)
	@next=$$(( $(FILE_VERSION) + 1 )); printf '%s\n' "$$next" > VERSION; \
	echo "VERSION: $(FILE_VERSION) -> $$next"

release: ## Bump VERSION, commit, tag v<VERSION> and push (CI then publishes)
	@test -z "$$(git status --porcelain)" || { echo "release: working tree is not clean" >&2; exit 1; }
	@test "$$(git rev-parse --abbrev-ref HEAD)" = main || { echo "release: switch to main first" >&2; exit 1; }
	@git fetch -q origin && test "$$(git rev-parse HEAD)" = "$$(git rev-parse origin/main)" \
		|| { echo "release: main is not in sync with origin/main" >&2; exit 1; }
	@$(MAKE) --no-print-directory check
	@next=$$(( $(FILE_VERSION) + 1 )); \
	if git rev-parse -q --verify "refs/tags/v$$next" >/dev/null; then \
		echo "release: tag v$$next already exists" >&2; exit 1; fi; \
	printf '%s\n' "$$next" > VERSION; \
	git add VERSION && git commit -q -m "Release v$$next" && \
	git tag -a "v$$next" -m "Release v$$next" && \
	git push --atomic origin main "v$$next" && \
	echo "released v$$next — CI will build and publish it"

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'
