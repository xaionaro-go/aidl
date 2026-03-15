.PHONY: generate genversions test e2e vet build build-examples lint clean readme smoke \
       aidlcli genaidlcli list-commands check-generated release

# Generated top-level directories.
GENERATED_DIRS := android com fuzztest libgui_test_server parcelables src

# All non-3rdparty Go packages.
GO_PACKAGES = $(shell go list -e ./... | grep -v /3rdparty/)

# Regenerate version-aware transaction code tables from AOSP tags.
genversions:
	go run ./tools/cmd/genversions

# Generate all Go code from AOSP AIDL definitions.
generate:
	go run ./tools/cmd/aospgen -3rdparty tools/pkg/3rdparty -output . -smoke-tests

# Run unit tests (compiler + runtime packages).
test:
	go test -v -race $(GO_PACKAGES)

# Run E2E tests (requires Android emulator or device).
e2e:
	go test -tags e2e ./tests/e2e/... -run TestAidlcli -v -timeout 300s

# Run go vet on all packages.
vet:
	go vet $(GO_PACKAGES)

# Build all commands, tools, and examples.
build:
	go build ./tools/cmd/...
	go build ./cmd/...
	@for d in examples/*/; do echo "Building $$d..."; go build "./$$d"; done

# Build aidlcli release binaries for arm64 and amd64.
release:
	@mkdir -p builds
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o builds/aidlcli-linux-arm64 ./cmd/aidlcli/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o builds/aidlcli-linux-amd64 ./cmd/aidlcli/

# Run linter.
lint:
	golangci-lint run ./...

# Regenerate README package table.
readme:
	go run ./tools/cmd/genreadme

# Regenerate E2E smoke tests.
smoke:
	go run ./tools/cmd/gen_e2e_smoke .

# Regenerate aidlcli registry and command dispatch code.
genaidlcli:
	go run ./tools/cmd/genaidlcli

# Build the aidlcli tool.
aidlcli:
	@mkdir -p builds
	go build -o builds/aidlcli ./cmd/aidlcli

# List all available aidlcli subcommands.
list-commands:
	go run ./cmd/aidlcli --help 2>&1 | grep '^ ' | awk '{print $$1}'

# Verify generated code matches a clean regeneration.
check-generated:
	make clean
	make generate
	make smoke
	make readme
	git diff --exit-code

# Remove all generated code.
clean:
	rm -rf $(GENERATED_DIRS)
	find . -maxdepth 1 -name '*.go' -exec grep -l 'Code generated' {} \; | xargs -r rm -f
