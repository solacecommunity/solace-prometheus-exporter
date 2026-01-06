#make
PKG_LIST := $(shell go list ./... | grep -v /vendor/)
BINARY_NAME=solace_prometheus_exporter
CMD_PATH=./cmd/solace-prometheus-exporter

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
	@echo "Building $(BINARY_NAME)..."
	@go build -a -ldflags '-s -w -extldflags "-static"' -o bin/$(BINARY_NAME) $(CMD_PATH)

clean: ## Remove previous build
	@rm -f reports/cover.html reports/cover.out solace_prometheus_exporter

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint:
	golangci-lint run
