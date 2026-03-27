BINARY      := scat
OUTPUT_DIR  := dist
MODULE_PATH := $(shell go list -m)
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-s -w -X '$(MODULE_PATH)/cmd.version=$(VERSION)'"

# Place Go caches inside the project directory so they are always writable,
# including in sandboxed environments. The .cache/ directory is git-ignored.
export GOMODCACHE ?= $(CURDIR)/.cache/go-mod
export GOCACHE    ?= $(CURDIR)/.cache/go-build
export GOTMPDIR   ?= $(CURDIR)/.cache/tmp
_CACHE_INIT := $(shell mkdir -p "$(GOMODCACHE)" "$(GOCACHE)" "$(GOTMPDIR)")

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build build-all package test lint check clean help

## build: Build binary for the current OS/Arch → dist/scat
build:
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY) .

## build-all: Cross-compile for all target platforms
build-all:
	@mkdir -p $(OUTPUT_DIR)
	$(foreach platform,$(PLATFORMS),$(call build_platform,$(platform)))

define build_platform
$(eval OS   := $(word 1,$(subst /, ,$(1))))
$(eval ARCH := $(word 2,$(subst /, ,$(1))))
$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
$(eval OUT  := $(OUTPUT_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT))
	GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(OUT) . ;
endef

## package: Build all binaries and create .zip archives → dist/
package: build-all
	$(foreach platform,$(PLATFORMS),$(call package_platform,$(platform)))

define package_platform
$(eval OS   := $(word 1,$(subst /, ,$(1))))
$(eval ARCH := $(word 2,$(subst /, ,$(1))))
$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
$(eval BIN  := $(OUTPUT_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT))
$(eval ZIP  := $(OUTPUT_DIR)/$(BINARY)-$(VERSION)-$(OS)-$(ARCH).zip)
	zip -j $(ZIP) $(BIN) ;
endef

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
