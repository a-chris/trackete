SHELL := /bin/bash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
IMAGE ?= trackete
TAG ?= $(VERSION)

.DEFAULT_GOAL := help

.PHONY: help install dev dev-api dev-web build build-web build-api test test-api test-web lint fmt docker-build docker-run docker-stop

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

install: ## Install frontend dependencies
	cd frontend && npm ci

dev: ## Run the API and the Vite dev server together
	$(MAKE) -j2 dev-api dev-web

dev-api: ## Run the Go API on :8080
	cd backend && DB_PATH=$${DB_PATH:-../data/trackete.db} go run ./cmd/server

dev-web: ## Run the Vite dev server on :5173
	cd frontend && npm run dev

build: build-web build-api ## Build frontend and backend

build-web: ## Build the React app into frontend/dist
	cd frontend && npm run build

build-api: ## Build the Go binary into backend/trackete
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o trackete ./cmd/server

test: test-api test-web ## Run all checks

test-api: ## Run Go vet and tests
	cd backend && go vet ./... && go test ./...

test-web: ## Typecheck, lint and build the frontend
	cd frontend && npm run typecheck && npm run lint && npm run build

lint: ## Lint the frontend
	cd frontend && npm run lint

fmt: ## Format Go sources and the frontend
	cd backend && go fmt ./...
	cd frontend && npx prettier --write "src/**/*.{ts,tsx,css}"

docker-build: ## Build the production image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) .

docker-run: docker-build ## Run the image on :8080 with a named volume
	docker run --rm --name trackete -p 8080:8080 -v trackete-data:/data $(IMAGE):$(TAG)

docker-stop: ## Stop the running container
	-docker stop trackete
