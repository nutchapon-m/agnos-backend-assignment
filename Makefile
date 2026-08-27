# --- Configuration ---
MIGRATIONS_DIR ?= ./business/sdk/migrate/sql
DATABASE_URL ?= postgres://username:password@localhost:5432/dbname?sslmode=disable

ifneq (,$(wildcard .env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

.PHONY: alert rule build

run:
	go run api/services/app/main.go \
	--service-name=tgms-ingest-bridge \
	--migration=true \
	--port=9999

bridge:
	go run api/services/bridge/main.go \
	--service-name=tgms-ingest-bridge \
	--exchange=tgms_telemetry_fanout \
	--exchange-type=fanout \
	--queue=tgms_telemetry \
	--binding-key=swd.dgr.*.telemetry

writer:
	go run api/services/writer/main.go \
	--service-name=tgms-ingest-writers \
	--exchange=tgms_telemetry_fanout \
	--exchange-type=fanout \
	--queue=tgms_telemetry \
	--binding-key=swd.dgr.*.telemetry

telemetryapp-proto-gen:
	protoc --go_out=app/domain/grpctelemetryapp --go_opt=paths=source_relative \
		--go-grpc_out=app/domain/grpctelemetryapp --go-grpc_opt=paths=source_relative \
		--proto_path=app/domain/grpctelemetryapp \
		app/domain/grpctelemetryapp/telemetryapp.proto

build:
	go build -o bin/alert ./api/services/bridge/main.go
	go build -o bin/rule ./api/services/writer/main.go

worker:
	go run api/services/worker/main.go \
		--service-name=tgms-ingest-worker \
		--exchange=tgms.events \
		--exchange-type=topic \
		--queue=tgms.ingest.events \
		--binding-key=core.device.#

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
