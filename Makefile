.DEFAULT_GOAL := help

# Variables
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html
BIN_DIR := bin
BIN_NAME := upfluence-app

.PHONY: help
help: ## Display this help message
	@echo "\033[1;36mAvailable commands in this Makefile:\033[0m"
	@awk 'BEGIN {FS = ":.*?## "} /^[-a-zA-Z0-9_]+:.*?## / {printf "  \033[32m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run all tests (unit and e2e)
	go test -v ./...

.PHONY: test-unit
test-unit: ## Run unit tests only
	go test -v ./configuration/... ./eventbus/... ./logger/... ./server/... ./service/...

.PHONY: test-e2e
test-e2e: ## Run End-to-End tests only
	go test -v ./e2e/...

.PHONY: test-race
test-race: ## Run all tests with race detector enabled (-race)
	go test -v -race ./...

.PHONY: coverage
coverage: ## Generate and display coverage summary in terminal
	go test -coverprofile=$(COVERAGE_FILE) -coverpkg=./... ./...
	go tool cover -func=$(COVERAGE_FILE)

.PHONY: coverage-html
coverage-html: ## Generate interactive HTML coverage report (coverage.html)
	go test -coverprofile=$(COVERAGE_FILE) -coverpkg=./... ./...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "\033[32m✔ HTML coverage report generated in $(COVERAGE_HTML)\033[0m"
	@echo "To open in browser:"
	@echo "  xdg-open $(COVERAGE_HTML)  # Linux"
	@echo "  open $(COVERAGE_HTML)      # macOS"

.PHONY: build
build: ## Build application binary
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BIN_NAME) main.go
	@echo "\033[32m✔ Binary compiled to $(BIN_DIR)/$(BIN_NAME)\033[0m"

.PHONY: run
run: ## Run application
	go run main.go

.PHONY: clean
clean: ## Remove coverage artifacts and compiled binaries
	rm -rf $(COVERAGE_FILE) $(COVERAGE_HTML) $(BIN_DIR)
	@echo "\033[32m✔ Cleanup completed\033[0m"

.PHONY: lint
lint: ## Execute the linter
	@ golangci-lint run

.PHONY: docker-build
docker-build: ## Build Docker image locally
	docker build -t $(BIN_NAME):latest .

.PHONY: docker-run
docker-run: ## Run Docker container locally on port 8080
	docker run --rm -p 8080:8080 $(BIN_NAME):latest
