.PHONY: help build clean swagger run push start-container stop-container gen-secret

AIR ?= ~/go/bin/air

GOARCH ?= amd64
GOOS ?= linux

OUTPUT_PATH ?= $(PWD)/bin
# GIT_SHA := $(shell git rev-parse --short HEAD)
BIN_NAME ?= eventmanager
# OUT_BIN := $(BIN_NAME)-$(GIT_SHA)
OUT_BIN := $(BIN_NAME)
LINK_BIN := $(OUTPUT_PATH)/$(BIN_NAME)

ENV ?= local

# Default help target
help:
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
build: ## Build the project
	@echo "Building..."
	@mkdir -p ${OUTPUT_PATH}
	go mod download
	GOARCH=${GOARCH} GOOS=${GOOS} go build -o ${OUTPUT_PATH}/${OUT_BIN} -x
	@ln -sfn $(notdir $(OUT_BIN)) $(LINK_BIN)
	@echo "Built $(OUT_BIN)"
	@echo "Symlink $(LINK_BIN) -> $(notdir $(OUT_BIN))"

clean: ## Clean the build output
	@printf "⚠️  Are you sure? [y/N]: " ; \
	read ans ; \
	case "$$ans" in \
			[yY][eE][sS]|[yY]) echo "🧹 Cleaning..." ; rm -rf $(OUTPUT_PATH) ;; \
			*) echo "❌ Cancelled." ;; \
	esac

run: ## Run the program
	@ENV=${ENV} $(AIR)

test: ## Run tests
	@go test -timeout 30s ./tests/... -coverprofile=coverage.out -coverpkg=./... -v
	@go tool cover -html=coverage.out -o coverage.html

frontend-build:
	cd frontend && PATH=$(PATH):/home/rocky/event-manager/.tools/node-v25.9.0-linux-x64/bin npm run build