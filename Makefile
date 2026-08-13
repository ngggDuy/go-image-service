# Makefile for go-image-service. Run `make` to list targets.

.DEFAULT_GOAL := help
COMPOSE := docker compose

proto: ## Regenerate Go code from proto/imageprocess.proto
	protoc --go_out=. --go_opt=module=go-image-service \
		--go-grpc_out=. --go-grpc_opt=module=go-image-service \
		proto/imageprocess.proto

test: ## Run all tests
	go test ./...

up: ## Build and start the full stack (foreground)
	$(COMPOSE) up --build

up-d: ## Build and start the full stack (background)
	$(COMPOSE) up -d --build

down: ## Stop and remove containers
	$(COMPOSE) down

clean: ## Stop containers AND delete volumes (wipes DB + uploads)
	$(COMPOSE) down -v

logs: ## Follow logs from the two Go services
	$(COMPOSE) logs -f httpserver imageservice

ps: ## Show container status
	$(COMPOSE) ps

db: ## Open a psql shell in the Postgres container
	$(COMPOSE) exec postgres psql -U imageservice -d imageservice

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

.PHONY: proto test up up-d down clean logs ps db help
