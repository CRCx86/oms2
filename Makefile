# Build variables
BINARY_NAME = oms2
BUILD_DIR = build
VERSION ?= $(shell git tag --points-at HEAD | tail -n 1)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%S")
COMMIT_SHA = $(shell git rev-parse --short HEAD)
LDFLAGS = -ldflags "-w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.commit=${COMMIT_SHA}"

# Docker variables
DOCKER_IMAGE ?= zinov/oms2
DOCKER_TAG ?= dev
DOCKER_LIQUIBASE_IMAGE ?= zinov/liquibase-oms2
LOGGING_TAG ?= oms2_app

date = $(shell date -u +"%Y-%m-%d-%H-%M")
username = $(shell git config user.name)
name ?= new_migration
filename = ${date}-${name}
file = ./migrations/${filename}.sql
migrations_changelog = ./migrations/changelog.yaml

.PHONY: migration
migration: ## Add migration file. Usage: $ make migration name="add-to-table"
	echo "-- liquibase formatted sql" >> ${file}
	echo "-- changeset ${username}:${filename}" >> ${file}
	echo "SOME SQL HERE" >> ${file}
	echo "-- rollback SOME SQL TO UNDO" >> ${file}
	echo "  - include:" >> ${migrations_changelog}
	echo "      file: ${filename}.sql" >> ${migrations_changelog}

.PHONY: env
env: ## Show env configuration
	go run ./cmd --help

.PHONY: liquibase-docker
liquibase-docker:
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker build -f ./deployments/liquibase/Dockerfile --rm -t ${DOCKER_LIQUIBASE_IMAGE}:${DOCKER_TAG} .

.PHONY: docker
docker: ## Build a Docker image
	docker build -f ./deployments/Dockerfile --rm -t ${DOCKER_IMAGE}:${DOCKER_TAG} .

.PHONY: dcup
dcup: liquibase-docker ## Local docker-compose up
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml up -d --build db esv701 liquibase
	deployments/wait.sh ${BINARY_NAME}_liquibase_1
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml up -d --build --scale liquibase=0

.PHONY: dcdown
dcdown:: ## Local docker-compose down
	DOCKER_IMAGE=${DOCKER_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml down

.PHONY: restart
restart:: ## Local docker-compose restart oms2
	docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml restart "app"

.PHONY: ps
ps:: ## Local docker-compose ps
	docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml ps

.PHONY: dcgrayup
dcgrayup:: ## Graylog docker-compose up
	docker-compose -f ./deployments/graylog/docker-compose.yml up -d --build

.PHONY: dcgraydown
dcgraydown:: ## Graylog docker-compose down
	docker-compose -f ./deployments/graylog/docker-compose.yml down

.PHONY: dbup
dbup:: liquibase-docker ## Local Postgres docker-compose up
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME} -f ./deployments/local/docker-compose.yml up -d --build db esv701 liquibase

.PHONY: dbdown
dbdown:: ## Postgres docker-compose down
	DOCKER_IMAGE=${DOCKER_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME} -f ./deployments/local/db/docker-compose.yml down

.PHONY: dep
dep: ## Install dependencies
	$(eval PACKAGE := $(shell go list -m))
	@go mod download
	@go mod vendor

.PHONY: gen
gen: ## Code generation easyjson marshalers
	@go generate ./cmd/oms2/main.go

.PHONY: test
test: dep gen ## Run unit tests
	@go test -v ./...

.PHONY: build
build: test ## Build a binary executable file
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${PACKAGE}/cmd

.PHONY: clean
clean:: ## Remove all containers, images and volumes
	-docker container stop $$(docker ps -q -a)
	-docker container rm $$(docker ps -q -a)
	-docker image rm -f $$(docker image ls -q)
	docker volume prune -f
	docker system prune -f

.PHONY: gc-ci
gc-ci:: ## Удаляет мусор после тестовых сборок для CI
	docker volume prune -f
	-docker rmi -f $$(docker images -f "dangling=true" -q)
	-docker rmi -f $$(docker images "oms2-inttest_integration-tests" -q)
	-docker rmi -f "${DOCKER_IMAGE}:${DOCKER_TAG}"

.PHONY: gc
gc:: ## Удаляет мусор после выключения тестового окружения, кроме образа приложения
	docker volume prune -f
	-docker rmi -f $$(docker images -f "dangling=true" -q)
	-docker rmi -f $$(docker images "oms2-inttest_integration-tests" -q)
	-docker rmi -f $$(docker images "oms2-inttest-debug_integration-tests" -q)
##	-docker rmi -f "${DOCKER_IMAGE}:${DOCKER_TAG}"

.PHONY: inttest-ci
inttest-ci: inttest-ci-down liquibase-docker ## CI integration tests
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest -f ./deployments/test/docker-compose.ci.yml up -d --build db liquibase
	deployments/wait.sh ${BINARY_NAME}-inttest_liquibase_1
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest -f ./deployments/test/docker-compose.ci.yml up --abort-on-container-exit --build --scale liquibase=0
	docker-compose -p ${BINARY_NAME}-inttest -f ./deployments/test/docker-compose.ci.yml logs integration-tests

.PHONY: inttest-ci-down
inttest-ci-down: ## ci integration tests down
	docker network prune -f
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest -f ./deployments/test/docker-compose.ci.yml down

.PHONY: inttest-debug-up
inttest-debug-up: liquibase-docker ## Run local integration tests for debugging
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest-debug -f ./deployments/test/docker-compose.tests.yml up -d --build db liquibase
	deployments/wait.sh ${BINARY_NAME}-inttest-debug_liquibase_1
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest-debug -f ./deployments/test/docker-compose.tests.yml up -d --build --scale liquibase=0

.PHONY: inttest-debug-down
inttest-debug-down:: ## Stop local integration test
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest-debug -f ./deployments/test/docker-compose.tests.yml down
	make gc

.PHONY: inttest-debug-restart-tests
inttest-debug-restart-tests:: ## Rebuild and run integration tests container
	DOCKER_IMAGE=${DOCKER_IMAGE} DOCKER_LIQUIBASE_IMAGE=${DOCKER_LIQUIBASE_IMAGE} VERSION=${DOCKER_TAG} docker-compose -p ${BINARY_NAME}-inttest-debug -f ./deployments/test/docker-compose.tests.yml up --build -d integration-tests

.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)
