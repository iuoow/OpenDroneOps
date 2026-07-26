SHELL := sh

.PHONY: fmt lint test test-race build web-install web-typecheck web-test web-build compose-config run-server

fmt:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

build:
	go build ./cmd/...

web-install:
	npm --prefix web ci

web-typecheck:
	npm --prefix web run typecheck

web-test:
	npm --prefix web test

web-build:
	npm --prefix web run build

compose-config:
	POSTGRES_IMAGE=postgres:18.4 REDIS_IMAGE=redis:7.2.14 MOSQUITTO_IMAGE=eclipse-mosquitto:2.1.2 POSTGRES_PASSWORD=local-only-placeholder docker compose -f deployment/docker-compose.blueprint.yaml config --quiet

run-server:
	go run ./cmd/server
