BINARY    := motivation
CONFIG    := ./quotes.yaml
ADDR      := 127.0.0.1:8765
PREFIX    ?= $(HOME)/.local/bin

HELPER_APP   := bin/MotivationNotify.app
HELPER_BIN   := $(HELPER_APP)/Contents/MacOS/MotivationNotify
HELPER_PLIST := $(HELPER_APP)/Contents/Info.plist
HELPER_SRC   := notify-helper/main.swift

.PHONY: build run tidy fmt vet clean install uninstall helper help

build: helper ## Build the Go binary and the notification helper
	go build -o $(BINARY) .

helper: $(HELPER_BIN) ## Build the Swift notification helper .app bundle

$(HELPER_BIN): $(HELPER_SRC) notify-helper/Info.plist
	mkdir -p $(HELPER_APP)/Contents/MacOS
	cp notify-helper/Info.plist $(HELPER_PLIST)
	swiftc -O -o $(HELPER_BIN) $(HELPER_SRC) -framework AppKit -framework UserNotifications
	codesign --force --sign - $(HELPER_APP)

run: build ## Run the server (CONFIG=path ADDR=host:port)
	./$(BINARY) -config $(CONFIG) -addr $(ADDR)

tidy: ## Sync go.mod / go.sum
	go mod tidy

fmt: ## Format Go sources
	go fmt ./...

vet: ## Static checks
	go vet ./...

clean: ## Remove built binary and helper app
	rm -f $(BINARY)
	rm -rf bin

install: build ## Install binary + helper to $(PREFIX) (default ~/.local/bin)
	install -d $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)
	mkdir -p $(PREFIX)/MotivationNotify.app/Contents/MacOS
	cp $(HELPER_PLIST) $(PREFIX)/MotivationNotify.app/Contents/Info.plist
	install -m 0755 $(HELPER_BIN) $(PREFIX)/MotivationNotify.app/Contents/MacOS/MotivationNotify
	codesign --force --sign - $(PREFIX)/MotivationNotify.app
	@echo "installed $(PREFIX)/$(BINARY) and $(PREFIX)/MotivationNotify.app"

uninstall: ## Remove installed binary + helper app from $(PREFIX) and flush notification caches
	rm -f $(PREFIX)/$(BINARY)
	rm -rf $(PREFIX)/MotivationNotify.app
	-killall usernoted 2>/dev/null || true
	-killall NotificationCenter 2>/dev/null || true
	@echo "removed $(PREFIX)/$(BINARY) and $(PREFIX)/MotivationNotify.app"

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
