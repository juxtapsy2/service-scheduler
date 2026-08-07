.DEFAULT_GOAL := help
COMPOSE := docker compose
BACKEND_DIR := service-scheduler-backend
FRONTEND_DIR := service-scheduler-frontend

.PHONY: help up down build ps logs restart \
        be be-build be-restart be-logs be-test \
        fe fe-build fe-restart fe-logs fe-dev \
        db db-reset clean

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start all services
	$(COMPOSE) up -d

down: ## Stop all services
	$(COMPOSE) down

build: ## Build all images
	$(COMPOSE) build

ps: ## Show running services
	$(COMPOSE) ps

logs: ## Tail logs from all services
	$(COMPOSE) logs -f --tail=100

restart: ## Restart all services
	$(COMPOSE) restart

be: ## Tail backend logs
	$(COMPOSE) logs -f --tail=100 service-scheduler-backend

be-build: ## Rebuild the backend image (re-applies migrations on start)
	$(COMPOSE) up -d --build service-scheduler-backend

be-restart: ## Restart the backend
	$(COMPOSE) restart service-scheduler-backend

be-logs: be ## Alias for be

be-test: ## Run backend Go tests (requires Postgres; loads .env)
	set -a; . ./.env; set +a; cd $(BACKEND_DIR) && go test ./...

fe: ## Tail frontend logs
	$(COMPOSE) logs -f --tail=100 service-scheduler-frontend

fe-build: ## Rebuild the frontend image
	$(COMPOSE) up -d --build service-scheduler-frontend

fe-restart: ## Restart the frontend
	$(COMPOSE) restart service-scheduler-frontend

fe-logs: fe ## Alias for fe

fe-dev: ## Run the frontend with Vite dev server (hot reload)
	cd $(FRONTEND_DIR) && npm run dev

db: ## Open a psql shell in the postgres container
	docker exec -it appointment-postgres psql -U postgres -d appointment_scheduler

db-reset: ## Recreate the database volume (WARNING: drops all data)
	$(COMPOSE) down -v
	$(COMPOSE) up -d

clean: ## Remove containers, images and volumes
	$(COMPOSE) down --rmi all -v
