SHELL := /bin/bash
.DEFAULT_GOAL := help

SERVICES      := gateway crawler rag voice
GOOSE_VERSION ?= v3.24.1
GOOSE         := go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
# sqlc embeds a C parser (cgo), so `go run ...@version` is slow and breaks on some toolchains: install the binary (brew install sqlc; tested with 1.31.1).
SQLC          := sqlc

# Load .env if present (never committed).
ifneq (,$(wildcard .env))
include .env
export
endif
DATABASE_URL ?= postgres://unipd:unipd@localhost:5432/unipd?sslmode=disable

.PHONY: help
help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-16s %s\n",$$1,$$2}'

.PHONY: build test lint fmt tidy
build: ## Build all services into ./bin
	@mkdir -p bin && for s in $(SERVICES); do go build -o bin/$$s ./cmd/$$s || exit 1; done
test: ## Run unit tests
	go test -race -count=1 ./...
lint: ## Run golangci-lint
	golangci-lint run ./...
fmt: ## Format code
	golangci-lint fmt ./...
tidy: ## go mod tidy
	go mod tidy

.PHONY: up down db-reset migrate migrate-reset migrate-down migrate-status sqlc sqlc-check
up: ## Start Postgres (+pgvector)
	docker compose up -d --wait postgres
down: ## Stop containers
	docker compose down
db-reset: ## DESTROY local DB volume and re-migrate
	docker compose down -v && $(MAKE) up && $(MAKE) migrate
migrate: ## Apply migrations
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up
migrate-reset: ## DESTRUCTIVE: roll back ALL migrations, then re-apply (checks the down scripts)
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" reset
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up
migrate-down: ## Roll back one migration
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" down
migrate-status: ## Show migration status
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" status
sqlc: ## Generate Go code from SQL queries
	$(SQLC) generate
sqlc-check: ## Validate queries against the migrations without generating code
	$(SQLC) compile

.PHONY: $(addprefix run-,$(SERVICES))
$(addprefix run-,$(SERVICES)): run-%: ## Run one service locally (make run-gateway)
	go run ./cmd/$*

.PHONY: eval
eval: ## Run the eval harness (see eval/README.md)
	cd eval && uv run python -m unipd_eval run --suite rag
