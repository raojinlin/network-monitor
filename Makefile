# Traffic Monitor Makefile

.PHONY: all clean frontend backend package dev help

# Default target
all: clean frontend backend package

# Variables
BUILD_DIR := build
FRONTEND_DIR := frontend
BACKEND_DIR := backend
BINARY_NAME := traffic-sniff
VERSION := $(shell date +"%Y%m%d-%H%M%S")

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

help: ## Show this help message
	@echo "Traffic Monitor Build System"
	@echo "============================="
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(RESET) %s\n", $$1, $$2}'

clean: ## Clean build artifacts
	@echo "$(GREEN)[INFO]$(RESET) Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(FRONTEND_DIR)/dist
	@go clean -cache
	@echo "$(GREEN)[SUCCESS]$(RESET) Cleaned build artifacts"

deps: ## Install dependencies
	@echo "$(GREEN)[INFO]$(RESET) Installing dependencies..."
	@cd $(FRONTEND_DIR) && npm install
	@cd $(BACKEND_DIR) && go mod download && go mod tidy
	@echo "$(GREEN)[SUCCESS]$(RESET) Dependencies installed"

frontend: ## Build frontend only
	@echo "$(GREEN)[INFO]$(RESET) Building frontend..."
	@cd $(FRONTEND_DIR) && npm install && npm run build
	@echo "$(GREEN)[SUCCESS]$(RESET) Frontend built"

backend: ## Build backend only
	@echo "$(GREEN)[INFO]$(RESET) Building backend..."
	@mkdir -p $(BUILD_DIR)
	@cd $(BACKEND_DIR) && go build -o ../$(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@echo "$(GREEN)[SUCCESS]$(RESET) Backend built"

backend-all: ## Build backend for current platform only (pcap requires CGO)
	@echo "$(YELLOW)[WARNING]$(RESET) Cross-platform builds are disabled due to pcap CGO requirements"
	@echo "$(YELLOW)[WARNING]$(RESET) To build for other platforms, run this command on the target platform"
	@echo "$(GREEN)[INFO]$(RESET) Building backend for current platform..."
	@mkdir -p $(BUILD_DIR)
	@cd $(BACKEND_DIR) && go build -o ../$(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go
	@echo "$(GREEN)[SUCCESS]$(RESET) Backend built for current platform"

package: ## Package the application
	@echo "$(GREEN)[INFO]$(RESET) Packaging application..."
	@mkdir -p $(BUILD_DIR)
	@cp -r $(FRONTEND_DIR)/dist $(BUILD_DIR)/frontend
	@mkdir -p $(BUILD_DIR)/data
	@echo "$(GREEN)[SUCCESS]$(RESET) Application packaged"

release: clean frontend backend-all package ## Create a complete release
	@echo "$(GREEN)[INFO]$(RESET) Creating release archive..."
	@tar -czf traffic-monitor-$(VERSION).tar.gz -C $(BUILD_DIR) .
	@echo "$(GREEN)[SUCCESS]$(RESET) Release created: traffic-monitor-$(VERSION).tar.gz"

dev-frontend: ## Start frontend development server
	@echo "$(GREEN)[INFO]$(RESET) Starting frontend development server..."
	@cd $(FRONTEND_DIR) && npm run dev

dev-backend: ## Start backend development server
	@echo "$(GREEN)[INFO]$(RESET) Starting backend development server..."
	@cd $(BACKEND_DIR) && go run cmd/server/main.go -port=8081 -interface=en0

test: ## Run tests
	@echo "$(GREEN)[INFO]$(RESET) Running tests..."
	@cd $(BACKEND_DIR) && go test ./...
	@echo "$(GREEN)[SUCCESS]$(RESET) Tests completed"

lint: ## Run linters
	@echo "$(GREEN)[INFO]$(RESET) Running linters..."
	@cd $(BACKEND_DIR) && go fmt ./... && go vet ./...
	@cd $(FRONTEND_DIR) && npm run lint 2>/dev/null || echo "No lint script found"
	@echo "$(GREEN)[SUCCESS]$(RESET) Linting completed"

docker: ## Build Docker image
	@echo "$(GREEN)[INFO]$(RESET) Building Docker image..."
	@docker build -t traffic-monitor:$(VERSION) .
	@docker tag traffic-monitor:$(VERSION) traffic-monitor:latest
	@echo "$(GREEN)[SUCCESS]$(RESET) Docker image built: traffic-monitor:$(VERSION)"

install: all ## Install to system
	@echo "$(GREEN)[INFO]$(RESET) Installing to system..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)[SUCCESS]$(RESET) Installed to /usr/local/bin/"

uninstall: ## Uninstall from system
	@echo "$(GREEN)[INFO]$(RESET) Uninstalling from system..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)[SUCCESS]$(RESET) Uninstalled from system"

# Development helpers
.PHONY: dev-frontend dev-backend test lint docker install uninstall