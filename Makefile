# TOOLCHAIN
GO		:= CGO_ENABLED=0 GOBIN=$(CURDIR)/bin go
GOFMT	:= $(GO)fmt

# ENVIRONMENT
VERBOSE		=

# GO TOOLS
GOLANGCI_LINT	:= bin/golangci-lint
GOTESTSUM		:= bin/gotestsum

# MISC
COVERPROFILE	:= coverage.out
DIST_DIR		:= dist

# FLAGS
GO_TEST_FLAGS		:= -race -coverprofile=$(COVERPROFILE)

# DEPENDENCIES
GOMODDEPS = go.mod go.sum

# Enable verbose test output if explicitly set.
GOTESTSUM_FLAGS	=
ifdef VERBOSE
	GOTESTSUM_FLAGS += --format=standard-verbose
endif

# FUNCTIONS
# func go-list-pkg-sources(package)
go-list-pkg-sources = $(GO) list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $(1)
# func go-pkg-sourcefiles(package)
go-pkg-sourcefiles = $(shell $(call go-list-pkg-sources,$(strip $1)))

.PHONY: all
all: dep fmt lint test ## Run dep, fmt, lint and test

.PHONY: clean
clean: ## Remove build and test artifacts
	@echo ">> cleaning up artifacts"
	@rm -rf $(COVERPROFILE) $(DIST_DIR)

.PHONY: cover
cover: $(COVERPROFILE) ## Calculate the code coverage score
	@echo ">> calculating code coverage"
	@$(GO) tool cover -func=$(COVERPROFILE) | tail -n1

.PHONY: dep-clean
dep-clean: ## Remove obsolete dependencies
	@echo ">> cleaning dependencies"
	@$(GO) mod tidy

.PHONY: dep-upgrade
dep-upgrade: ## Upgrade all direct dependencies to their latest version
	@echo ">> upgrading dependencies"
	@$(GO) get -d $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@make dep

.PHONY: dep
dep: dep-clean dep.stamp ## Install and verify dependencies and remove obsolete ones

dep.stamp: $(GOMODDEPS)
	@echo ">> installing dependencies"
	@$(GO) mod download
	@$(GO) mod verify
	@touch $@

.PHONY: fmt
fmt: ## Format and simplify the source code using `gofmt`
	@echo ">> formatting code"
	@! $(GOFMT) -s -w $(shell find . -path -prune -o -name '*.go' -print) | grep '^'

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint the source code
	@echo ">> linting code"
	@$(GOLANGCI_LINT) run

.PHONY: test
test: $(GOTESTSUM) ## Run all unit tests. Run with VERBOSE=1 to get verbose test output ('-v' flag).
	@echo ">> running tests"
	@$(GOTESTSUM) $(GOTESTSUM_FLAGS) -- $(GO_TEST_FLAGS) ./...

.PHONY: tools
tools: $(GOLANGCI_LINT) $(GOTESTSUM) ## Install all tools into the projects local $GOBIN directory

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# MISC TARGETS

$(COVERPROFILE):
	@make test

# GO TOOLS

$(GOLANGCI_LINT): dep.stamp $(call go-pkg-sourcefiles, github.com/golangci/golangci-lint/cmd/golangci-lint)
	@echo ">> installing golangci-lint"
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint

$(GOTESTSUM): dep.stamp $(call go-pkg-sourcefiles, gotest.tools/gotestsum)
	@echo ">> installing gotestsum"
	@$(GO) install gotest.tools/gotestsum