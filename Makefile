GO ?= go
BIN_DIR ?= bin
GOOSE_VERSION := v3.27.3
GOOSE_BUILD_TAGS := no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb
GOLANGCI_LINT_VERSION := v2.13.2
GOVULNCHECK_VERSION := v1.7.0
REDOCLY_VERSION := 2.49.0
DATABASE_URL ?= postgres://aibos:aibos_local@localhost:5432/aibos?sslmode=disable

.PHONY: fmt fmt-check lint test test-integration build docker-up docker-down migrate-up openapi-lint security clean

fmt:
	$(GO) fmt ./...

fmt-check:
	@files="$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'))"; test -z "$$files" || (echo "$$files" && exit 1)

lint:
	$(GO) vet ./...
	$(GO) run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

test:
	$(GO) test -race ./...

test-integration:
	$(GO) test -race -tags=integration ./test/integration

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BIN_DIR)/api ./cmd/api
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BIN_DIR)/worker ./cmd/worker

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

migrate-up:
	$(GO) run -tags='$(GOOSE_BUILD_TAGS)' github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir migrations postgres "$(DATABASE_URL)" up

openapi-lint:
	npx --yes @redocly/cli@$(REDOCLY_VERSION) lint api/openapi.yaml

security:
	$(GO) mod verify
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

clean:
	rm -f $(BIN_DIR)/api $(BIN_DIR)/worker
