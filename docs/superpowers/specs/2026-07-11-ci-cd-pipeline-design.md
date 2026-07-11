# CI/CD Pipeline — 技术设计方案

**版本**: v1.0
**日期**: 2026-07-11
**状态**: Draft
**依赖**: VideoStreamCaptureEngine Phase 1 + 1.5 + 2（已完成）

---

## 1. 概述

为 VideoStreamCaptureEngine 建立完整的 CI/CD 流水线，覆盖编译、测试、lint、Docker 镜像构建和自动发布。

### 核心原则

| 原则 | 说明 |
|------|------|
| **Makefile 单一入口** | 所有构建逻辑封装在 Makefile，CI 只做薄层编排 |
| **本地 == CI** | `make ci` 在本地和 CI 中行为一致 |
| **零 CGO** | 项目无 CGO 依赖，`CGO_ENABLED=0` 简化交叉编译 |
| **版本可追溯** | 二进制注入版本号和 commit SHA |

---

## 2. 整体架构

```
开发者本地:  make ci  ──→ lint → build → test → docker-build
                           ↑
                          Makefile (single source of truth)
                           ↓
GitHub Actions:   trigger workflow  ──→  make <target>  ──→ 结果上报
                   ├─ push/PR → ci.yml
                   └─ tag v*   → release.yml
```

---

## 3. Makefile 改造

### 3.1 新增变量

```makefile
APP           = engine
CMD_DIR       = ./cmd/engine
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT       ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS       = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"
GOOS         ?= linux
GOARCH       ?= amd64
COVERAGE_DIR  = ./coverage
```

### 3.2 目标清单

| 目标 | 命令 | 说明 |
|------|------|------|
| `build` | `CGO_ENABLED=0 go build $(LDFLAGS) -o $(CMD_DIR)/$(APP) $(CMD_DIR)` | 编译 Go 二进制，注入版本信息 |
| `test` | `go test ./internal/... -v -race -count=1` | 运行全部单元测试 + race detector |
| `test-coverage` | `go test ./internal/... -coverprofile=$(COVERAGE_DIR)/cover.out -covermode=atomic && go tool cover -html=$(COVERAGE_DIR)/cover.out -o $(COVERAGE_DIR)/index.html` | 测试 + HTML 覆盖率报告 |
| `lint` | `golangci-lint run ./...` | Go lint |
| `frontend-lint` | `cd web && npx oxlint` | 前端 lint |
| `frontend-build` | `cd web && npm ci && npm run build` | 构建 React SPA |
| `docker-build` | `docker build -f deploy/Dockerfile -t $(APP):$(VERSION) .` | 构建 Docker 镜像 |
| `ci` | `lint && frontend-lint && frontend-build && build && test && docker-build` | 完整流水线 |
| `clean` | `rm -rf $(CMD_DIR)/$(APP) $(COVERAGE_DIR) web/dist` | 清理构建产物 |

### 3.3 默认目标

```makefile
.DEFAULT_GOAL := build
```

---

## 4. GitHub Actions 工作流

### 4.1 CI Workflow (`.github/workflows/ci.yml`)

**触发条件**: push 到 `master` + 所有 PR（含 fork）

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

      - name: Lint
        run: make lint

      - name: Frontend lint
        run: make frontend-lint

      - name: Frontend build
        run: make frontend-build

      - name: Build
        run: make build

      - name: Test with coverage
        run: make test-coverage

      - name: Upload coverage
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

      - name: Login to GHCR
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

### 4.2 Release Workflow (`.github/workflows/release.yml`)

**触发条件**: push tag `v*.*.*`

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

---

## 5. Dockerfile 修复

### 5.1 当前问题

| 问题 | 当前值 | 修复值 |
|------|--------|--------|
| Go 基础镜像版本 | `golang:1.22-alpine` | `golang:1.25-alpine` |
| CGO 模式 | `CGO_ENABLED=1` + gcc/musl-dev | `CGO_ENABLED=0`，去掉 gcc/musl-dev |
| 前端构建缓存 | 无 npm 缓存 | 利用 BuildKit 缓存 |

### 5.2 修复后的 Dockerfile

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

---

## 6. 版本信息注入

`cmd/engine/main.go` 新增两行：

```go
var (
    version = "dev"
    commit  = "unknown"
)
```

HTTP 响应头或 `/health` 端点返回版本信息：

```go
func init() {
    slog.Info("starting engine", "version", version, "commit", commit)
}
```

---

## 7. 分支保护规则

在 GitHub 仓库 Settings → Branches 中配置 `master`：

| 规则 | 值 |
|------|-----|
| Require status checks | ✅ CI / build-and-test 必须通过 |
| Require branches up-to-date | ✅ 合并前必须基于最新 master |
| Require signed commits | 可选（推荐） |
| Allow auto-merge | 可选 |

---

## 8. 新增/修改文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `.github/workflows/ci.yml` | 新增 | CI 工作流 |
| `.github/workflows/release.yml` | 新增 | Release 工作流 |
| `Makefile` | 修改 | 新增目标：lint, test-coverage, ci, docker-build, frontend-* |
| `deploy/Dockerfile` | 修改 | Go 版本 1.22→1.25，CGO_ENABLED=1→0 |
| `cmd/engine/main.go` | 修改 | 增加 version/commit ldflags |
| `.gitignore` | 修改 | 增加 coverage/ |

---

## 9. 风险与对策

| 风险 | 对策 |
|------|------|
| golangci-lint 配置缺失 | 在 Makefile 中 fallback：无配置时跳过，仅 warning |
| CI token 权限不足（fork PR） | fork PR 不触发 docker push，只运行 build+test |
| GitHub Actions runner 磁盘不足 | 前端 node_modules 用缓存 action，避免重复下载 |
| 镜像体积过大 | 多阶段构建 + alpine 基础镜像，~30MB 最终镜像 |
