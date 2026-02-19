.PHONY: all build test lint run clean frontend-install frontend-dev frontend-build embed-frontend help

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOLINT=golangci-lint
BINARY=compliance-operator-dashboard
BUILD_DIR=./bin
EMBED_DIR=./internal/api/frontend_dist

all: build

## build: Build frontend then Go binary with embedded SPA
build: frontend-build embed-frontend
	@echo "Building Go binary..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) .

## test: Run Go unit tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

## run: Build and run the dashboard
run: build
	$(BUILD_DIR)/$(BINARY) serve

## frontend-install: Install frontend dependencies
frontend-install:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

## frontend-dev: Start Vite dev server with API proxy
frontend-dev:
	@echo "Starting frontend dev server..."
	cd frontend && npm run dev

## frontend-build: Build frontend for production
frontend-build: frontend-install
	@echo "Building frontend..."
	cd frontend && npm run build

## embed-frontend: Copy frontend dist to embed location
embed-frontend:
	@echo "Copying frontend dist for embedding..."
	rm -rf $(EMBED_DIR)
	cp -r frontend/dist $(EMBED_DIR)

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -rf $(EMBED_DIR)
	mkdir -p $(EMBED_DIR)
	touch $(EMBED_DIR)/.gitkeep

## help: Show available targets
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
