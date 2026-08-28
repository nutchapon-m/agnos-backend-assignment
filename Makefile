# --- Configuration ---
MIGRATIONS_DIR ?= ./src/business/sdk/migrate/sql
DATABASE_URL ?= postgres://username:password@localhost:5432/dbname?sslmode=disable

ifneq (,$(wildcard .env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

.PHONY: alert rule build migrate-create migrate-up migrate-down migrate-force migrate-step test test-integration

run: 
	@go run src/main.go

build:
	@go build ./...

test:
	@go build ./... && go vet ./... && go test ./... -count=1

test-integration:
	@DATABASE_URL="$(DATABASE_URL)" go test ./repository/... -count=1 -v


# --- Migration Commands ---
migrate-create:
	@echo "Generating new migration..."
	@migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up:
	@echo "Applying all up migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-step:
	@echo "Applying step migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" step $(steps)

migrate-down:
	@echo "Applying all down migrations..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down

migrate-force:
	@echo "Forcing migration version..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(version)
