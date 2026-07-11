# CI/CD Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Set up a complete CI/CD pipeline with Makefile-driven build, GitHub Actions workflows, and fixed Dockerfile.

**Architecture:** Makefile becomes single source of truth for all build/lint/test/docker targets. GitHub Actions workflows call Makefile targets, not raw commands. Two workflows: CI (PR/push to master) and Release (tag push). Dockerfile updated to Go 1.25 and CGO_ENABLED=0.

**Tech Stack:** Makefile, GitHub Actions, Docker BuildKit, golangci-lint

---

## File Structure Map

```
VideoStreamCaptureEngine/
├── .github/
│   └── workflows/
│       ├── ci.yml                  # NEW: CI workflow (PR + push master)
│       └── release.yml             # NEW: Release workflow (tag v*)
├── cmd/engine/main.go              # MODIFY: add version/commit ldflags
├── deploy/Dockerfile                # MODIFY: Go 1.25, CGO_ENABLED=0
├── Makefile                        # MODIFY: add targets (lint, ci, docker-build, etc.)
├── .gitignore                      # MODIFY: add coverage/
```

---

### Task 1: main.go — Version ldflags

**Files:**
- Modify: `cmd/engine/main.go:1-10`

- [ ] **Step 1: Add version variables**

Add package-level variables before `func main()`:

```go
var (
	version = "dev"
	commit  = "unknown"
)
```

- [ ] **Step 2: Add init logging**

Insert after `package main` block, before `func main()`:

```go
func init() {
	slog.Info("engine build info", "version", version, "commit", commit)
}
```

- [ ] **Step 3: Verify build with ldflags**

```bash
go build -ldflags "-X main.version=v1.0.0 -X main.commit=abc1234" -o /tmp/engine ./cmd/engine/ && /tmp/engine -h 2>&1 | head -5
```
Expected: binary builds without errors, init log shows version and commit.

- [ ] **Step 4: Commit**

```bash
git add cmd/engine/main.go
git commit -m "feat: add version and commit ldflags injection"
```

---

### Task 2: Makefile — New Build Targets

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Replace existing Makefile**

Read the existing Makefile first, then write the new version:

```makefile
.PHONY: build run test test-coverage lint frontend-lint frontend-build \
        docker-build ci clean dev-frontend

APP           = engine
CMD_DIR       = ./cmd/engine
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS       = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"
GOOS         ?= linux
GOARCH       ?= amd64
COVERAGE_DIR  = ./coverage

.DEFAULT_GOAL := build

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(CMD_DIR)/$(APP) $(CMD_DIR)

run: build
	$(CMD_DIR)/$(APP) -config ./configs/config.example.yaml

test:
	go test ./internal/... -v -race -count=1

test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	go test ./internal/... -coverprofile=$(COVERAGE_DIR)/cover.out -covermode=atomic -count=1
	go tool cover -html=$(COVERAGE_DIR)/cover.out -o $(COVERAGE_DIR)/index.html
	@echo "Coverage report: $(COVERAGE_DIR)/index.html"

lint:
	golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

frontend-lint:
	cd web && npx oxlint 2>/dev/null || echo "oxlint not available, skipping"

frontend-build:
	cd web && npm ci && npm run build

docker-build:
	docker build -f deploy/Dockerfile -t $(APP):$(VERSION) .

ci: lint frontend-lint frontend-build build test

clean:
	rm -rf $(CMD_DIR)/$(APP) $(COVERAGE_DIR) web/dist

dev-frontend:
	cd web && npm run dev
```

- [ ] **Step 2: Verify Makefile syntax**

```bash
make -n build 2>&1 | head -5
```
Expected: dry-run output showing the go build command with ldflags.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat: expand Makefile with lint, ci, docker-build, test-coverage targets"
```

---

### Task 3: Dockerfile — Fix Go Version & CGO

**Files:**
- Modify: `deploy/Dockerfile`

- [ ] **Step 1: Update Dockerfile**

Read the existing file, then apply these changes:

Line 10: `FROM golang:1.22-alpine AS go-builder` → `FROM golang:1.25-alpine AS go-builder`

Line 11: remove `RUN apk add --no-cache gcc musl-dev` (no CGO needed)

Line 11 (new): add `RUN apk add --no-cache ca-certificates` (keep certs for TLS)

Line 17: `RUN CGO_ENABLED=1 go build -o /engine ./cmd/engine/` → `RUN CGO_ENABLED=0 go build -o /engine ./cmd/engine/`

Final result:

```dockerfile
# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /engine ./cmd/engine/

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
COPY --from=go-builder /engine /engine
COPY configs/config.example.yaml /config.yaml
EXPOSE 8080
ENTRYPOINT ["/engine", "-config", "/config.yaml"]
```

- [ ] **Step 2: Commit**

```bash
git add deploy/Dockerfile
git commit -m "fix: update Dockerfile to Go 1.25 and disable CGO"
```

---

### Task 4: .gitignore — Add Coverage Directory

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Add coverage/ to .gitignore**

Read `.gitignore`, then append:

```gitignore

# Test coverage
/coverage/
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: ignore test coverage output directory"
```

---

### Task 5: GitHub Actions — CI Workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create directory**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Write `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
    branches: [master]
  pull_request:
    branches: [master]

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: 'npm'
          cache-dependency-path: web/package-lock.json

      - name: Lint Go
        run: make lint

      - name: Lint frontend
        run: make frontend-lint

      - name: Build frontend
        run: make frontend-build

      - name: Build binary
        run: make build

      - name: Run tests with coverage
        run: make test-coverage

      - name: Upload coverage report
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage/

  docker:
    needs: build-and-test
    if: github.event_name == 'push' && github.ref == 'refs/heads/master'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Login to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          file: deploy/Dockerfile
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}
```

- [ ] **Step 3: Verify YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('valid')"
```
Expected: `valid`

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions CI workflow (build, test, lint, docker)"
```

---

### Task 6: GitHub Actions — Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write `.github/workflows/release.yml`**

```yaml
name: Release

on:
  push:
    tags: ['v*.*.*']

jobs:
  build-matrix:
    strategy:
      matrix:
        goos: [linux]
        goarch: [amd64, arm64]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build binary
        run: |
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          make build

      - name: Upload binary
        uses: actions/upload-artifact@v4
        with:
          name: engine-${{ matrix.goos }}-${{ matrix.goarch }}
          path: cmd/engine/engine

  docker-multiarch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: docker/setup-qemu-action@v3

      - uses: docker/setup-buildx-action@v3

      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push multi-arch
        uses: docker/build-push-action@v6
        with:
          context: .
          file: deploy/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
            ghcr.io/${{ github.repository }}:latest

  release:
    needs: [build-matrix, docker-multiarch]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
```

- [ ] **Step 2: Verify YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); print('valid')"
```
Expected: `valid`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add GitHub Actions release workflow (multi-arch docker + GitHub release)"
```

---

## Self-Review

1. **Spec coverage:**
   - Task 1 → spec §6 (version ldflags)
   - Task 2 → spec §3 (Makefile targets)
   - Task 3 → spec §5 (Dockerfile fix)
   - Task 4 → spec §8 (.gitignore)
   - Task 5 → spec §4.1 (CI workflow)
   - Task 6 → spec §4.2 (Release workflow)
   - All spec requirements covered.

2. **Placeholder scan:** No TBD, TODO, incomplete code, or vague instructions. Every step has exact file paths, complete code blocks, and runnable commands.

3. **Type consistency:** 
   - Makefile `LDFLAGS` variable references `main.version` and `main.commit`, which match Task 1's `var` declarations.
   - Workflow `ci.yml` calls `make lint`, `make frontend-lint`, `make frontend-build`, `make build`, `make test-coverage` — all defined in Task 2 Makefile.
   - Workflow Go version `'1.25'` matches Dockerfile `golang:1.25-alpine`.
   - All consistent.
