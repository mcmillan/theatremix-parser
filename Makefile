# theatremix-parser — build, test and package.
#
#   make            build bin/theatremix-parser for this machine
#   make check      gofmt + go vet + go test (what CI runs)
#   make dist       package a release archive for GOOS/GOARCH (default: host)
#   make dist-all   package every release target
#   make help       list all targets

BINARY  := theatremix-parser
PKG     := ./cmd/$(BINARY)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOFLAGS := -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXT     = $(if $(filter windows,$(GOOS)),.exe,)

TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

DIST_NAME = $(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH)
DIST_DIR  = dist/$(DIST_NAME)

.PHONY: all build install test vet fmt fmt-check check golden dist dist-all clean help

all: build

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

help: ## List targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "} {printf "  %-10s %s\n", $$1, $$2}'
