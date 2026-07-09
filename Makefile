GOLANGCI_LINT_VERSION := v2.6.2
COMPOSE := docker compose --project-directory . -f env/docker-compose/docker-compose.yml

.PHONY: run build test test-integration lint compose-up compose-down

run: ## Запуск сервиса локально
	go run ./cmd/subscriptions

build: ## Сбор бинарника в /bin
	go build -o bin/subscriptions ./cmd/subscriptions

test: ## Запуск юнит тестов
	go test -race ./...

test-integration: ## Запуск интеграционных тестов (нужен запущенный postgres: make compose-up)
	go test -race -tags integration ./tests/...

compose-up: ## Поднять локальное окружение (postgres)
	$(COMPOSE) up -d

compose-down: ## Остановить локальное окружение
	$(COMPOSE) down

lint: ## Запуск golangci-lint
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...


