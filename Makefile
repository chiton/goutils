.PHONY: all help up test test-integration install-gotestsum
COMPOSE_FILE=build/docker/docker-compose.yml

all: help

test: install-gotestsum ## runs all the unit tests
	CATEGORY=unit gotestsum -- -v -count 1 ./...

test-integration: .env install-gotestsum ## runs all tests including integration
	# https://stackoverflow.com/a/30969768
	# the -count 1 prevents test caching
	set -o allexport; source .env; CATEGORY=integration gotestsum -- -v -count 1 ./...; set +o allexport

install-gotestsum:
	if ! [ -n $(command -v gotestsum) ]; then go install gotest.tools/gotestsum@v1.7.0; else echo "gotestsum already installed"; fi

up: .env ## runs the message broker
	docker compose -f ${COMPOSE_FILE} --env-file .env up db -d

down: .env ## shuts down all the containers and networks
	docker compose -f ${COMPOSE_FILE} --env-file .env down

help: ## Show this help.
	@echo 'This makefile is used as a development tool to help get all services running quickly from source'
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    %-20s%s\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  %s\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)



