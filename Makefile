GIT_BRANCH       := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT       := $(shell git rev-parse HEAD)
GIT_COMMIT_SHORT := $(shell git rev-parse --short HEAD)
GIT_VERSION      := $(shell git describe --tags --always --dirty)
GIT_TAG          := $(shell git describe --tags)
BUILD_TIME       := $(shell date -u +"%Y-%m-%dT%H:%M:%S %Z")

LDFLAGS := -ldflags '\
-X "gbs2gb/constants.GitBranch=$(GIT_BRANCH)" \
-X "gbs2gb/constants.GitCommit=$(GIT_COMMIT)" \
-X "gbs2gb/constants.GitCommitShort=$(GIT_COMMIT_SHORT)" \
-X "gbs2gb/constants.GitVersion=$(GIT_VERSION)" \
-X "gbs2gb/constants.GitTag=$(GIT_TAG)" \
-X "gbs2gb/constants.BuildTime=$(BUILD_TIME)"'
BINARY_NAME = gbs2gb
GOCMD    = go
GOFORMAT = $(GOCMD) fmt
GOBUILD  = $(GOCMD) build -trimpath
GOCLEAN  = $(GOCMD) clean
GOTEST   = $(GOCMD) test
GOGET    = $(GOCMD) get
GOBUILDEXT = $(GOBUILD) -o $(BINARY_NAME) $(LDFLAGS)

.EXPORT_ALL_VARIABLES:
	GOAMD64 = v3
.PHONY: build image test format clean run debug deps
build:
	$(GOBUILDEXT)
	@echo Build completed
buildall:
	@echo Building windows-x64…
	@GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-x64.exe $(LDFLAGS)
	@echo Building macos-x64…
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-macos-x64 $(LDFLAGS)
	@echo Building macos-arm64…
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-macos-arm64 $(LDFLAGS)
	@echo Building linux-x64…
	@GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-x64 $(LDFLAGS)
	@echo Building linux-arm64…
	@GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-linux-arm64 $(LDFLAGS)
	@echo Builds completed
image:
	docker buildx build --progress=plain -t $(BINARY_NAME) .
test: build
	$(GOTEST) -v ./...
format:
	$(GOFORMAT) ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
run: build
	./$(BINARY_NAME)
debug: build
	./$(BINARY_NAME) --debug
deps:
	$(GOGET)
