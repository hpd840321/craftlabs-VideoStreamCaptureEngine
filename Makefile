.PHONY: build run test lint clean dev-frontend

APP=engine
CMD_DIR=./cmd/engine

build:
	cp -r web/dist cmd/engine/dist 2>/dev/null || true
	go build -o $(CMD_DIR)/$(APP) $(CMD_DIR)

run: build
	$(CMD_DIR)/$(APP) -config ./configs/config.example.yaml

test:
	go test ./internal/... -v -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -f $(CMD_DIR)/$(APP)

dev-frontend:
	cd web && npm run dev
