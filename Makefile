export GO111MODULE=on

#===================#
#== Env Variables ==#
#===================#
DOCKER_COMPOSE_FILE ?= docker-compose.yaml
BINARY_NAME         ?= qstorm
VERSION ?= $(shell git describe --tags --always --dirty)
help: ## Show this help
	@IFS=$$'\n' ; \
	help_lines=(`fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##/:/'`); \
	printf "%-25s %s\n" "target" "help" ; \
	printf "%-25s %s\n" "------" "----" ; \
	for help_line in $${help_lines[@]}; do \
		IFS=$$':' ; \
		help_split=($$help_line) ; \
		help_command=`echo $${help_split[0]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		printf '\033[36m%-25s\033[0m %s\n' $$help_command $$help_info; \
	done

#===============#
#== App Build ==#
#===============#

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "-X main.version=$(VERSION)" -o ./bin/$(BINARY_NAME) ./cmd/qstorm/

build-docker: ## Build the Docker image
	docker build -t $(BINARY_NAME):latest . --no-cache

#=======================#
#== ENVIRONMENT SETUP ==#
#=======================#

env: ## Copy .env.sample to .env if it doesn't exist
ifeq (,$(wildcard .env))
	cp .env.sample .env
endif

clean: ## Remove built binaries
	rm -rf bin/*

#====================#
#== QUALITY CHECKS ==#
#====================#

lint: ## Run golangci-lint
	docker run -t --rm -v ${PWD}:/app -v $$(go env GOMODCACHE):/go/pkg/mod \
		-w /app golangci/golangci-lint:v2.11.4 golangci-lint run -v

test: ## Run unit tests with coverage
	@echo "Running unit tests..."
	go test -tags unit -race -shuffle=on \
		$$(go list ./... | grep -v mock | grep -v generated | tr '\n' ' ') \
		-coverpkg=./... -coverprofile coverage.out

test-integration: ## Run integration tests (requires emulator)
	@echo "Running integration tests..."
	PUBSUB_EMULATOR_HOST=localhost:8095 go test -tags integration -race -v \
		-coverprofile coverage-integration.out -coverpkg=./... ./... -timeout 60s

fmt: ## Format code
	gci write -s standard -s default . --skip-generated --skip-vendor && \
	gofumpt -l -w .

generate: ## Run go generate across all packages
	go generate ./...

#==========================#
#== DOCKER INFRASTRUCTURE ==#
#==========================#

docker-start: ## Start Docker environment
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d --build --remove-orphans

docker-stop: ## Stop Docker environment
	docker compose -f $(DOCKER_COMPOSE_FILE) stop

docker-clean: docker-stop ## Remove Docker containers and volumes
	docker compose -f $(DOCKER_COMPOSE_FILE) rm -v -f

docker-restart: docker-stop docker-start ## Restart Docker environment

#========================#
#== LOCAL DEVELOPMENT  ==#
#========================#

gcp-pubsub: ## Start PubSub emulator and create topic
	@docker compose up -d pubsub
	@echo "Waiting for PubSub emulator..."
	@for i in $$(seq 1 30); do \
		curl -s -X PUT http://localhost:8095/v1/projects/qstorm-project/topics/qstorm-topic > /dev/null 2>&1 && break; \
		sleep 1; \
	done
	@curl -sf http://localhost:8095/v1/projects/qstorm-project/topics/qstorm-topic > /dev/null && \
		echo "PubSub: ready (port 8095, topic created)" || \
		echo "PubSub: FAILED to create topic"

kafka: ## Start Kafka (plaintext + SASL)
	@docker compose up -d kafka kafka-sasl
	@echo "Waiting for Kafka brokers..."
	@for i in $$(seq 1 30); do \
		docker compose logs kafka 2>&1 | grep -q "Kafka Server started" && break; \
		sleep 1; \
	done
	@docker compose logs kafka 2>&1 | grep -q "Kafka Server started" && \
		echo "Kafka: ready (port 9092)" || \
		echo "Kafka: FAILED to start"
	@for i in $$(seq 1 30); do \
		docker compose logs kafka-sasl 2>&1 | grep -q "Kafka Server started" && break; \
		sleep 1; \
	done
	@docker compose logs kafka-sasl 2>&1 | grep -q "Kafka Server started" && \
		echo "Kafka SASL: ready (port 9093)" || \
		echo "Kafka SASL: FAILED to start"

rabbitmq: ## Start RabbitMQ
	@docker compose up -d rabbitmq
	@echo "Waiting for RabbitMQ..."
	@for i in $$(seq 1 30); do \
		docker compose logs rabbitmq 2>&1 | grep -q "Server startup complete" && break; \
		sleep 1; \
	done
	@docker compose logs rabbitmq 2>&1 | grep -q "Server startup complete" && \
		echo "RabbitMQ: ready (port 5672)" || \
		echo "RabbitMQ: FAILED to start"

environment: gcp-pubsub kafka rabbitmq ## Start all services and wait for readiness
	@echo "All services ready."

run: build ## Build and run with example config
	./bin/$(BINARY_NAME) example/gcp_pubsub_test_config.json

.PHONY: help build build-docker env clean \
        lint test test-integration fmt generate \
        docker-start docker-stop docker-clean docker-restart \
        gcp-pubsub kafka rabbitmq environment run
