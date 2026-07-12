BINARY      := scat
OUTPUT_DIR  := dist
MODULE_PATH := $(shell go list -m)
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-X '$(MODULE_PATH)/cmd.version=$(VERSION)'"

# macOS Developer ID signing / notarization (see nlink-jp/.github
# CONVENTIONS.md §Code Signing). Defaults match any Developer ID
# Application cert in the keychain and the org-standard notary
# profile. Builds without these fall back to ad-hoc / un-notarized
# with a one-line warning — see scripts/codesign-darwin.sh.
CODESIGN_IDENTITY ?= Developer ID Application
NOTARY_PROFILE    ?= nlink-jp-notary

# Place Go caches inside the project directory so they are always writable,
# including in sandboxed environments. The .cache/ directory is git-ignored.
export GOMODCACHE ?= $(CURDIR)/.cache/go-mod
export GOCACHE    ?= $(CURDIR)/.cache/go-build
export GOTMPDIR   ?= $(CURDIR)/.cache/tmp
_CACHE_INIT := $(shell mkdir -p "$(GOMODCACHE)" "$(GOCACHE)" "$(GOTMPDIR)")

# darwin ships arm64 only (no amd64, no universal). linux/windows keep their matrix.
PLATFORMS := darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: build build-all package test lint check clean help

## build: Build binary for the current OS/Arch → dist/scat
build:
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY) .
	@scripts/codesign-darwin.sh $(OUTPUT_DIR)/$(BINARY) "$(CODESIGN_IDENTITY)"

## build-all: Cross-compile for all target platforms
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@for p in $(PLATFORMS); do os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = windows ] && ext=".exe"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY)-$$os-$$arch$$ext . ; \
	done
	@scripts/codesign-darwin.sh $(OUTPUT_DIR)/$(BINARY)-darwin-arm64 "$(CODESIGN_IDENTITY)" "$(BINARY)"

## package: Build all platforms, archive with version suffix (zip for
## darwin/windows, tar.gz for linux), bundle the canonical binary +
## README.md + LICENSE, and notarize the darwin build → dist/. Asset
## naming follows the org Release Archive Standard
## (scat-vX.Y.Z-<os>-<arch>.<ext>).
package: build-all
	@cd $(OUTPUT_DIR) && for p in $(PLATFORMS); do os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = windows ] && ext=".exe"; \
		stage=_pkg; rm -rf $$stage; mkdir -p $$stage; \
		cp "$(BINARY)-$$os-$$arch$$ext" "$$stage/$(BINARY)$$ext"; \
		cp ../README.md ../LICENSE $$stage/; \
		base="$(BINARY)-$(VERSION)-$$os-$$arch"; \
		if [ "$$os" = linux ]; then ( cd $$stage && tar -czf "../$$base.tar.gz" * ); \
		else ( cd $$stage && zip -q "../$$base.zip" * ); fi; \
		rm -rf $$stage; \
	done
	@scripts/notarize-darwin.sh $(OUTPUT_DIR)/$(BINARY)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"

## test: Run the test suite
test:
	go test ./...

## lint: Run linters
lint:
	go vet ./...

## check: lint + test
check: lint test

## clean: Remove dist/ and caches
clean:
	rm -rf $(OUTPUT_DIR) .cache

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

# Homebrew tap generation (see scripts/release-brew.mk). After `make package`,
# `make brew` generates this formula from the built darwin-arm64 zip into the
# local nlink-jp/homebrew-tap checkout. The package target is unchanged.
BREW_KIND := formula
BREW_DESC := Send content to Slack and other services from your terminal
include scripts/release-brew.mk
