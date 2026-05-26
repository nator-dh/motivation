BINARY    := motivation
CONFIG    := ./quotes.yaml
ADDR      := 127.0.0.1:8765
PREFIX    ?= $(HOME)/.local/bin

.PHONY: build run tidy fmt vet clean install install-notifier help

build: ## Build the binary into ./$(BINARY)
	go build -o $(BINARY) .

run: ## Run the server (CONFIG=path ADDR=host:port)
	go run . -config $(CONFIG) -addr $(ADDR)

tidy: ## Sync go.mod / go.sum
	go mod tidy

fmt: ## Format Go sources
	go fmt ./...

vet: ## Static checks
	go vet ./...

clean: ## Remove built binary
	rm -f $(BINARY)

install: build ## Install binary to $(PREFIX) (default ~/.local/bin)
	install -d $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)
	@echo "installed $(PREFIX)/$(BINARY)"

install-notifier: ## brew install terminal-notifier (enables clickable notifications)
	brew install terminal-notifier

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
