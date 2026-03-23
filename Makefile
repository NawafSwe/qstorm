export GO111MODULE=on

#===================#
#== Env Variables ==#
#===================#
DOCKER_COMPOSE_FILE ?= docker-compose.yml
RUN_IN_DOCKER       ?= docker compose exec builder
BINARY_NAME         ?= qstorm

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
	go build -o ./bin/$(BINARY_NAME) ./cmd/qstorm/

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
		-w /app golangci/golangci-lint:v2.7.0 golangci-lint run -v

test: ## Run unit tests
	@echo "Running unit tests..."
	go test -tags unit -shuffle=on \
		$$(go list ./... | grep -v mock | grep -v generated | tr '\n' ' ') \
		-coverpkg=./... -coverprofile coverage.out

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -tags integration -shuffle=on ./...

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

environment: docker-start ## Start emulator and create PubSub topic
	@echo "Waiting for PubSub emulator to be ready..."
	@sleep 3
	@curl -s -X PUT http://localhost:8095/v1/projects/qstorm-project/topics/qstorm-topic > /dev/null && \
		echo "Topic 'qstorm-topic' created successfully" || \
		echo "Topic may already exist"

load-example: build ## Run load test with GCP PubSub example config
	./bin/$(BINARY_NAME) example/gcp_pubsub_test_config.json

.PHONY: help build build-docker env clean \
        lint test test-integration fmt generate \
        docker-start docker-stop docker-clean docker-restart \
        environment load-example
