# Makefile

## Location to install local development tools to
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
DEV_PORT := $(LOCALBIN)/dev-port
OPEN_BROWSER := $(LOCALBIN)/open-browser
DEV_TAG := $(LOCALBIN)/dev-tag
GO_INSTALL_TOOL := $(LOCALBIN)/go-install-tool

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.3.0

# renovate: datasource=npm depName=prettier
PRETTIER_VERSION ?= 3.9.6

DEV_PORT_VERSIONED := $(DEV_PORT)-$(DEV_TOOLS_VERSION)
OPEN_BROWSER_VERSIONED := $(OPEN_BROWSER)-$(DEV_TOOLS_VERSION)
DEV_TAG_VERSIONED := $(DEV_TAG)-$(DEV_TOOLS_VERSION)
GO_INSTALL_TOOL_VERSIONED := $(GO_INSTALL_TOOL)-$(DEV_TOOLS_VERSION)

## Build Configuration
BINARY ?= lore
COMMAND ?= ./cmd
RUN_ARGS ?=
BUILD_VERSION ?= dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS ?= -s -w -X main.Version=$(BUILD_VERSION) -X main.Commit=$(BUILD_COMMIT)

# Debugging
COMPOSE_PROJECT ?= $(notdir $(CURDIR))
COMPOSE_FILE := deploy/compose.yaml
DB_CONTAINER_NAME ?= postgres
PDF_CONTAINER_NAME ?= html2pdf

# Named ports persist across separate Make invocations in this checkout.
dev-port = $(or $(shell $(DEV_PORT) $(1)),$(error Could not resolve port for $(1)))
LORE_ASSIGNED_PORT ?= $(call dev-port,app)
DB_ASSIGNED_PORT ?= $(call dev-port,postgres)
PDF_ASSIGNED_PORT ?= $(call dev-port,pdf)

## Site Configuration
SITE_CONFIG ?= docs/site.toml
SITE_PORT ?= 8081
SCREENSHOT_SCRIPT := scripts/screenshots/run.sh
SCREENSHOT_BROWSER_CHANNEL ?= chrome

## Formatting
PRETTIER_MD_SOURCES := README.md "docs/content/**/*.md"

VERSION_PREFIX ?= v

##@ Tagging

.PHONY: patch
patch: dev-tools ## Create a new patch release (x.y.Z+1).
	$(DEV_TAG) --prefix "$(VERSION_PREFIX)" patch

.PHONY: minor
minor: dev-tools ## Create a new minor release (x.Y+1.0).
	$(DEV_TAG) --prefix "$(VERSION_PREFIX)" minor

.PHONY: major
major: dev-tools ## Create a new major release (X+1.0.0).
	$(DEV_TAG) --prefix "$(VERSION_PREFIX)" major

.PHONY: tag
tag: dev-tools ## Show the latest tag.
	@echo "Latest version: $$($(DEV_TAG) --prefix "$(VERSION_PREFIX)" current)"

.PHONY: push
push: ## Push tags to the configured remote.
	git push --tags

##@ Development

.PHONY: ports-reset
ports-reset: dev-tools ## Clear saved ports after stopping local services.
	$(DEV_PORT) --reset

.PHONY: ports
ports: dev-tools ## Print selected local development ports.
	@$(DEV_PORT) app --port "$(LORE_ASSIGNED_PORT)" > /dev/null
	@$(DEV_PORT) postgres --port "$(DB_ASSIGNED_PORT)" > /dev/null
	@$(DEV_PORT) pdf --port "$(PDF_ASSIGNED_PORT)" > /dev/null
	@echo "Lore: http://127.0.0.1:$(LORE_ASSIGNED_PORT)/"
	@echo "Postgres: 127.0.0.1:$(DB_ASSIGNED_PORT)"
	@echo "PDF: http://127.0.0.1:$(PDF_ASSIGNED_PORT)/render"

# Start build work only after ports are printed, including with make -j.
.PHONY: dev-build
dev-build: ports
	$(MAKE) generate web

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

.PHONY: test-browser
test-browser: check-web ## Run browser regressions in Chrome (override BROWSER_CHANNEL if needed).
	$(NODE) --test test/browser/*.test.mjs

.PHONY: download
download: $(NODE_MODULES) dev-tools ## Download Go, frontend, and development dependencies.
	go mod download

.PHONY: postgres
postgres: ports ## Run postgres locally.
	@echo "Starting Postgres on dynamic host port: $(DB_ASSIGNED_PORT)"
	@LORE_POSTGRES_PORT=$(DB_ASSIGNED_PORT) docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d --wait --wait-timeout 60 $(DB_CONTAINER_NAME)

.PHONY: html-pdf
html-pdf: ports ## Run html2pdf locally.
	@LORE_PDF_PORT=$(PDF_ASSIGNED_PORT) docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d $(PDF_CONTAINER_NAME)

.PHONY: serve
serve: ports ## Run Lore using the saved ports (services and build must already be ready).
	@echo "Starting Lore application..."
	@LORE_POSTGRES_PORT=$(DB_ASSIGNED_PORT) go run $(COMMAND) serve \
		--debug \
		--access-log \
		--listen-address="127.0.0.1:$(LORE_ASSIGNED_PORT)" \
		--log-format text \
		--pdf-url="http://127.0.0.1:$(PDF_ASSIGNED_PORT)/render" \
		--database-url="postgres://lore:lore@127.0.0.1:$(DB_ASSIGNED_PORT)/lore?sslmode=disable" \
		$(RUN_ARGS)

.PHONY: open
open: ports ## Open the browser once Lore responds.
	$(OPEN_BROWSER) "http://127.0.0.1:$(LORE_ASSIGNED_PORT)/"

.PHONY: run
run: dev-build html-pdf postgres ## Build, start services, and run Lore with the browser.
	@$(OPEN_BROWSER) "http://127.0.0.1:$(LORE_ASSIGNED_PORT)/" & \
	browser_pid=$$!; \
	trap 'kill "$$browser_pid" 2>/dev/null || true' EXIT; \
	$(MAKE) serve

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

.PHONY: screenshots
screenshots: generate web $(NODE_MODULES) ## Regenerate documentation screenshots from docs/content using an isolated database.
	SCREENSHOT_BROWSER_CHANNEL="$(SCREENSHOT_BROWSER_CHANNEL)" $(SCREENSHOT_SCRIPT)

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
fmt: fmt-web fmt-go fmt-md ## Format all supported files.

.PHONY: fmt-web
fmt-web: $(NODE_MODULES) ## Format CSS and TypeScript source files.
	$(NPX) --yes prettier@$(PRETTIER_VERSION) --write "web/src/**/*.css" "web/src/**/*.ts" "test/**/*.ts"

.PHONY: fmt-go
fmt-go: generate web ## Format Go code.
	go fmt ./...

.PHONY: fmt-md
fmt-md: ## Format Markdown files with Prettier.
	$(NPX) --yes prettier@$(PRETTIER_VERSION) --write README.md "docs/**/*.md"

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

.PHONY: dev-tools
dev-tools: \
	$(DEV_PORT_VERSIONED) \
	$(OPEN_BROWSER_VERSIONED) \
	$(DEV_TAG_VERSIONED) \
	$(GO_INSTALL_TOOL_VERSIONED) ## Download the pinned development tools.
	@ln -sf "$(notdir $(DEV_PORT_VERSIONED))" "$(DEV_PORT)"
	@ln -sf "$(notdir $(OPEN_BROWSER_VERSIONED))" "$(OPEN_BROWSER)"
	@ln -sf "$(notdir $(DEV_TAG_VERSIONED))" "$(DEV_TAG)"
	@ln -sf "$(notdir $(GO_INSTALL_TOOL_VERSIONED))" "$(GO_INSTALL_TOOL)"

$(DEV_PORT_VERSIONED): | $(LOCALBIN)
	$(call download-dev-tool,dev-port,$@)

$(OPEN_BROWSER_VERSIONED): | $(LOCALBIN)
	$(call download-dev-tool,open-browser,$@)

$(DEV_TAG_VERSIONED): | $(LOCALBIN)
	$(call download-dev-tool,dev-tag,$@)

$(GO_INSTALL_TOOL_VERSIONED): | $(LOCALBIN)
	$(call download-dev-tool,go-install-tool,$@)

# download-dev-tool downloads a versioned tool from gi8lino/dev-tools.
# $1 - release asset name
# $2 - versioned destination path
define download-dev-tool
	@set -eu; \
	tmp="$(2).tmp"; \
	trap 'rm -f "$$tmp"' EXIT INT TERM; \
	echo "Downloading gi8lino/dev-tools $(DEV_TOOLS_VERSION) $(1)"; \
	curl --fail --silent --show-error --location \
		"https://github.com/gi8lino/dev-tools/releases/download/$(DEV_TOOLS_VERSION)/$(1)" \
		-o "$$tmp"; \
	chmod +x "$$tmp"; \
	mv "$$tmp" "$(2)"; \
	trap - EXIT INT TERM
endef

.PHONY: golangci-lint
golangci-lint: dev-tools ## Download golangci-lint locally if necessary.
	$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
