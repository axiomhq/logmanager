# TOOLCHAIN
GO	  := CGO_ENABLED=0 go
CGO	  := CGO_ENABLED=1 go

# ENVIRONMENT
VERBOSE =

# MISC
COVERPROFILE := coverage.out

# FLAGS
GO_TEST_FLAGS		:= -race -coverprofile=$(COVERPROFILE)

# DEPENDENCIES
GOMODDEPS = go.mod go.sum

# Enable verbose test output if explicitly set.
GOTESTSUM_FLAGS	=
ifdef VERBOSE
	GOTESTSUM_FLAGS += --format=standard-verbose
endif

.PHONY: all
all: dep fmt lint test ## Run dep, fmt, lint and test

.PHONY: clean
clean: ## Remove build and test artifacts
	@echo ">> cleaning up artifacts"
	@rm -rf $(COVERPROFILE) dist/ bin/

.PHONY: cover
cover: $(COVERPROFILE) ## Calculate the code coverage score
	@echo ">> calculating code coverage"
	@$(GO) tool cover -func=$(COVERPROFILE) | tail -n1

.PHONY: dep-clean
dep-clean: ## Remove obsolete dependencies
	@echo ">> cleaning dependencies"
	@$(GO) mod tidy

.PHONY: dep-upgrade
dep-upgrade: ## Upgrade all direct and tool dependencies to their latest version
	@echo ">> upgrading dependencies"
	@$(GO) get $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all) $(shell $(GO) list tool)
	@$(MAKE) dep

.PHONY: dep
dep: dep-clean dep.stamp ## Install and verify dependencies and remove obsolete ones

dep.stamp: $(GOMODDEPS)
	@echo ">> installing dependencies"
	@$(GO) mod download
	@$(GO) mod verify
	@touch $@

.PHONY: fmt
fmt: ## Format and simplify the source code using `golangci-lint fmt`
	@echo ">> formatting code"
	@$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt

.PHONY: lint
lint: ## Lint the source code
	@echo ">> linting code"
	@$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

.PHONY: test
test: ## Run all unit tests. Run with VERBOSE=1 to get verbose test output ('-v' flag).
	@echo ">> running tests"
	@$(CGO) run gotest.tools/gotestsum $(GOTESTSUM_FLAGS) -- $(GO_TEST_FLAGS) ./...

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# MISC TARGETS

$(COVERPROFILE):
	@$(MAKE) test
