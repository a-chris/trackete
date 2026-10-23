SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE ?= trackete
TAG ?= $(VERSION)
AIR_VERSION ?= v1.67.4
AIR ?= $(CURDIR)/backend/.bin/air

.DEFAULT_GOAL := help

.PHONY: help install dev dev-api dev-web build build-web build-api test test-api test-web lint fmt docker-build docker-run docker-stop

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

install: ## Install frontend dependencies
	cd frontend && bun install

dev: ## Run the API (live reload) and the Vite dev server together
	$(MAKE) -j2 dev-api dev-web

$(AIR):
	@echo "Installing Air $(AIR_VERSION) into backend/.bin (first run only)…"
	@cd backend && GOBIN=$(CURDIR)/backend/.bin go install github.com/air-verse/air@$(AIR_VERSION)

dev-api: $(AIR) ## Run the Go API with live reload on :8080
	cd backend && DB_PATH=$${DB_PATH:-../data/trackete.db} PORT=$${PORT:-8080} STATIC_DIR=$${STATIC_DIR:-} $(AIR)

dev-web: ## Run the Vite dev server on :5173
	cd frontend && bun --bun run dev

build: build-web build-api ## Build frontend and backend

build-web: ## Build the React app into frontend/dist
	cd frontend && bun --bun run build

build-api: ## Build the Go binary into backend/trackete
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o trackete ./cmd/server

test: test-api test-web ## Run all checks

test-api: ## Run Go vet and tests
	cd backend && go vet ./... && go test ./...

test-web: ## Typecheck, lint and build the frontend
	cd frontend && bun --bun run typecheck && bun --bun run lint && bun --bun run build

lint: ## Lint the frontend
	cd frontend && bun --bun run lint

fmt: ## Format Go sources and the frontend
	cd backend && go fmt ./...
	cd frontend && bunx --bun prettier --write "src/**/*.{ts,tsx,css}"

docker-build: ## Build the production image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) .

docker-run: docker-build ## Run the image on :8080 with a named volume
	docker run --rm --name trackete -p 8080:8080 -v trackete-data:/data $(IMAGE):$(TAG)

docker-stop: ## Stop the running container
	-docker stop trackete
