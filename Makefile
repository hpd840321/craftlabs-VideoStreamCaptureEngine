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

build: frontend-build
	rm -rf $(CMD_DIR)/dist && cp -r web/dist $(CMD_DIR)/dist
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
	which golangci-lint 2>/dev/null && golangci-lint run ./... || echo "golangci-lint not installed, skipping lint"

frontend-lint:
	cd web && (which oxlint 2>/dev/null && npx oxlint || echo "oxlint not available, skipping frontend lint")

frontend-build:
	cd web && npm ci && npm run build

docker-build:
	docker build -f deploy/Dockerfile -t $(APP):$(VERSION) .

ci: lint frontend-lint frontend-build build test

clean:
	rm -rf $(CMD_DIR)/$(APP) $(CMD_DIR)/dist $(COVERAGE_DIR) web/dist

dev-frontend:
	cd web && npm run dev
