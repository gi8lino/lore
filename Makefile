# Makefile

## Location to install Go dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Frontend
WEB_BUILD := scripts/web/build.sh
CSS_BUILD := scripts/web/build-css.sh
CSS_ENTRY := web/src/css/app.css
CSS_OUTPUT := web/dist/css/app.css
NODE ?= node
NPM ?= npm
NPX ?= npx
TSC ?= ./node_modules/.bin/tsc
NODE_MODULES := node_modules/.package-lock.json

## Tool Binaries
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2
# renovate: datasource=npm depName=prettier
PRETTIER_VERSION ?= 3.9.6

## Build Configuration
BINARY ?= lore
COMMAND ?= ./cmd
RUN_ARGS ?=
SITE_CONFIG ?= docs/lore-site.toml
BUILD_VERSION ?= dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS ?= -s -w -X main.Version=$(BUILD_VERSION) -X main.Commit=$(BUILD_COMMIT)

## Site Configuration
SITE_CONFIG ?= lore-site.toml
SITE_PORT ?= 8081

# Default tag prefix. Override with an empty value for unprefixed tags.
VERSION_PREFIX ?= v

##@ Tagging

# Find the latest tag with the configured prefix, or use 0.0.0 when none exists.
LATEST_TAG = $(shell git tag --list "$(VERSION_PREFIX)*" --sort=-v:refname | head -n 1)
VERSION = $(shell [ -n "$(LATEST_TAG)" ] && echo $(LATEST_TAG) | sed "s/^$(VERSION_PREFIX)//" || echo "0.0.0")

.PHONY: patch
patch: ## Create a new patch release (x.y.Z+1)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

.PHONY: minor
minor: ## Create a new minor release (x.Y+1.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.%d.0", $$1, $$2+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

.PHONY: major
major: ## Create a new major release (X+1.0.0)
	@NEW_VERSION=$$(echo "$(VERSION)" | awk -F. '{printf "%d.0.0", $$1+1}') && \
	git tag "$(VERSION_PREFIX)$${NEW_VERSION}" && \
	echo "Tagged $(VERSION_PREFIX)$${NEW_VERSION}"

.PHONY: tag
tag: ## Show the latest tag.
	@echo "Latest version: $(LATEST_TAG)"

.PHONY: push
push: ## Push tags to the configured remote.
	git push --tags

##@ Development

.PHONY: generate
generate: ## Generate the Lucide icon catalog from the installed Go dependency.
	go generate ./internal/icons

.PHONY: check-generated
check-generated: generate ## Verify committed generated files are current.
	git diff --exit-code -- internal/icons/catalog_gen.go

.PHONY: css
css: ## Bundle split CSS sources into web/dist/css/app.css.
	@CSS_ENTRY="$(CSS_ENTRY)" CSS_OUTPUT="$(CSS_OUTPUT)" $(CSS_BUILD)

.PHONY: web
web: $(NODE_MODULES) ## Build the frontend distribution from web/src.
	@CSS_BUILD="$(CSS_BUILD)" CSS_ENTRY="$(CSS_ENTRY)" CSS_OUTPUT="$(CSS_OUTPUT)" TSC="$(TSC)" $(WEB_BUILD)

.PHONY: check-web
check-web: web ## Build the frontend and verify browser assets.
	@test -s "$(CSS_OUTPUT)"
	@test -s web/dist/sw.js
	@test -z "$$(find web/dist -type f -name '*.ts' -print -quit)"
	@find web/dist/js -type f -name '*.js' -exec $(NODE) --check {} \;
	@$(NODE) --check web/dist/sw.js

.PHONY: typecheck
typecheck: $(NODE_MODULES) ## Type-check all authored TypeScript without emitting files.
	$(TSC) -p tsconfig.json --noEmit
	$(TSC) -p web/src/ts/service-worker/tsconfig.json --noEmit
	$(TSC) -p test/ts/tsconfig.json --noEmit

.PHONY: test-web
test-web: check-web ## Compile and run the TypeScript frontend unit tests.
	@set -eu; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT INT TERM; \
	$(TSC) -p test/ts/tsconfig.json --outDir "$$tmp"; \
	$(NODE) --test "$$tmp"/test/ts/*.test.js

.PHONY: download
download: $(NODE_MODULES) ## Download Go and frontend dependencies.
	go mod download

.PHONY: run
run: generate web ## Run Lore locally.
	go run $(COMMAND) serve --debug --access-log --log-format text $(RUN_ARGS) --database-url="postgres://lore:lore@localhost:5432/lore?sslmode=disable"

.PHONY: build
build: generate web ## Build the Lore binary.
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(COMMAND)

.PHONY: site
site: generate web ## Build the published read-only documentation site.
	go run $(COMMAND) build --config "$(SITE_CONFIG)"

.PHONY: site-serve
site-serve: generate web ## Build and serve the documentation site locally.
	go run $(COMMAND) build \
		--config "$(SITE_CONFIG)" \
		--site-url "http://127.0.0.1:$(SITE_PORT)/"
	@echo "Serving Lore documentation at http://127.0.0.1:$(SITE_PORT)"
	python3 -m http.server $(SITE_PORT) --bind 127.0.0.1 --directory docs/site

.PHONY: vet
vet: generate web ## Run Go static analysis.
	go vet ./...

.PHONY: test
test: test-web vet ## Run frontend and backend unit tests.
	go test -covermode=atomic -count=1 -timeout=3m ./...

.PHONY: test-race
test-race: test-web vet ## Run unit tests with the Go race detector.
	go test -race -count=1 -timeout=3m ./...

.PHONY: cover
cover: test-web ## Display Go test coverage.
	go test -coverprofile=coverage.out -covermode=atomic -count=1 -timeout=3m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean up generated application files.
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf web/dist

##@ Formatting

.PHONY: fmt
fmt: fmt-web fmt-go ## Format all supported files.

.PHONY: fmt-web
fmt-web: $(NODE_MODULES) ## Format CSS and TypeScript source files.
	$(NPX) --yes prettier@$(PRETTIER_VERSION) --write "web/src/**/*.css" "web/src/**/*.ts" "test/**/*.ts"

.PHONY: fmt-go
fmt-go: generate web ## Format Go code.
	go fmt ./...

.PHONY: lint
lint: typecheck check-web lint-go ## Run all linters and formatting checks.

.PHONY: lint-go
lint-go: web golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: web golangci-lint ## Run golangci-lint and apply fixes.
	$(GOLANGCI_LINT) run --fix

##@ Dependencies

$(NODE_MODULES): package.json package-lock.json
	$(NPM) ci

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.

$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool installs a versioned tool and links its stable binary name.
# $1 - target path with name of binary
# $2 - package URL
# $3 - version
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
