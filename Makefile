SHELL := /bin/bash
BUILD_NUMBER := `date +%Y%m%d%H%M`
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET := $(shell tput -Txterm sgr0)

.PHONY: help

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "${YELLOW}%-16s${GREEN}%s${RESET}\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	@go mod tidy
	@go mod download

build: ## Build binary for local operating system
	@go generate ./...
	@go build -o mysql2parquet main.go

clean: ## Remove build related file
	@go clean
