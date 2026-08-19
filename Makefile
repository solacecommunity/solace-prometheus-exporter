#make
PKG_LIST := $(shell go list ./... | grep -v /vendor/)
BINARY_NAME=solace_prometheus_exporter
CMD_PATH=./cmd/solace-prometheus-exporter

# Build metadata for solace_exporter_build_info. Derived from git for local builds and overridable
# (e.g. `make build VERSION=1.4.2`); falls back to the same defaults as internal/version outside a git checkout.
# The leading "v" of a tag is stripped so a local build reports the same version string as the pipeline.
# $(or ...) rather than `|| echo dev`: with a pipe, the shell reports sed's exit status, so the fallback
# would never fire outside a git checkout and VERSION would silently end up empty.
VERSION ?= $(or $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'),dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
VERSION_PKG := solace_exporter/internal/version
# github.com/prometheus/common/version backs the "Build context" startup log line only; we never register its
# collector, which would clash with solace_exporter_build_info under the same name.
PROM_VERSION_PKG := github.com/prometheus/common/version
# Recursively expanded on purpose: this keeps the git/date shell-outs above out of `make test`, `lint`, `help`, ...
LDFLAGS = -s -w -extldflags "-static" \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).BuildDate=$(BUILD_DATE) \
	-X $(PROM_VERSION_PKG).Version=$(VERSION) \
	-X $(PROM_VERSION_PKG).Revision=$(COMMIT) \
	-X $(PROM_VERSION_PKG).BuildDate=$(BUILD_DATE)

.PHONY: dep vet test test-coverage build clean help lint

dep: ## Get the dependencies
	@go mod vendor

vet: ## Run go vet
	@go vet ${PKG_LIST}

test: ## Run unit tests
	@go test -short ${PKG_LIST}

test-coverage: ## Run tests with coverage
	mkdir -p reports
	@go test -short -coverprofile reports/cover.out ${PKG_LIST}
	@go tool cover -html reports/cover.out -o reports/cover.html

build: ## Build binary
	@echo "Building $(BINARY_NAME) $(VERSION) ($(COMMIT), $(BUILD_DATE))..."
	@go build -a -ldflags '$(LDFLAGS)' -o bin/$(BINARY_NAME) $(CMD_PATH)

clean: ## Remove previous build
	@rm -f reports/cover.html reports/cover.out solace_prometheus_exporter

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint:
	golangci-lint run
