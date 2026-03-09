# ============================================================
#  CapPlan Makefile
#  Usage: make <target>
# ============================================================

DISTRO       ?= ubuntu
PROM_VERSION ?= 2.51.2
NODE_VERSION ?= 1.8.2
PLATFORM     ?= linux-amd64
DC           = docker compose -f deployments/docker/docker-compose.yml

.PHONY: help setup setup-go setup-python setup-binaries install-deps \
        db-up db-down db-logs db-psql db-migrate \
        agent forecast run clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

setup: setup-go setup-python setup-binaries ## First-time setup: Go + Python venv + binaries

setup-go: ## Download Go module dependencies
	go mod download && go mod tidy

setup-python: ## Create .venv and install Python requirements
	uv venv .venv
	uv pip install -r requirements.txt --python .venv/bin/python

setup-binaries: prometheus/prometheus node_exporter/node_exporter ## Download Prometheus + Node Exporter

prometheus/prometheus:
	@mkdir -p prometheus
	curl -fsSL "https://github.com/prometheus/prometheus/releases/download/v$(PROM_VERSION)/prometheus-$(PROM_VERSION).$(PLATFORM).tar.gz" \
		| tar -xz --strip-components=1 -C prometheus

node_exporter/node_exporter:
	@mkdir -p node_exporter
	curl -fsSL "https://github.com/prometheus/node_exporter/releases/download/v$(NODE_VERSION)/node_exporter-$(NODE_VERSION).$(PLATFORM).tar.gz" \
		| tar -xz --strip-components=1 -C node_exporter

install-deps: ## Run setup/setup.go for your distro (DISTRO=ubuntu)
	cd setup && go run setup.go -distro $(DISTRO)

db-up: ## Start TimescaleDB container
	$(DC) up -d

db-down: ## Stop TimescaleDB container
	$(DC) down

db-logs: ## Tail database logs
	$(DC) logs -f timescaledb

db-psql: ## Open psql shell in container
	docker exec -it capplan-db psql -U capplan -d capplan

db-migrate: ## Apply schema to running container
	docker exec -i capplan-db psql -U capplan -d capplan \
		< deployments/docker/init-scripts/01_schema.sql

agent: ## Run the metrics agent
	cd agent && go run metrics.go

forecast: ## Train the XGBoost forecast model
	cd forecast && ../.venv/bin/python train_model.py

run: db-up agent forecast ## Full pipeline: db → agent → forecast

clean: ## Remove data dirs and stop containers
	rm -rf agent/data prometheus/data
	$(DC) down
