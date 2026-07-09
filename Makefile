GOLANGCI_LINT_VERSION := v2.6.2

.PHONY: run build test lint tidy

run: ## Запуск сервиса локально
	go run ./cmd/subscriptions

build: ## Сбор бинарника в /bin
	go build -o bin/subscriptions ./cmd/subscriptions

test: ## Запуск юнит тестов
	go test -race ./...

lint: ## Запуск golangci-lint
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...


