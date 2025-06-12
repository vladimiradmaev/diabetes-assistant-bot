.PHONY: help build run stop clean logs dev

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Определяем команду Docker Compose
DOCKER_COMPOSE_CMD := $(shell if command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; elif docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo ""; fi)

help: ## Показать справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

check-docker: ## Проверить наличие Docker Compose
	@if [ -z "$(DOCKER_COMPOSE_CMD)" ]; then \
		echo "$(RED)❌ Docker Compose не найден!$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)✅ Используется: $(DOCKER_COMPOSE_CMD)$(NC)"; \
	fi

build: check-docker ## Собрать Docker образы
	@echo "$(GREEN)Сборка Docker образов...$(NC)"
	$(DOCKER_COMPOSE_CMD) build

run: check-docker ## Запустить приложение с базой данных
	@echo "$(GREEN)Запуск приложения с PostgreSQL...$(NC)"
	$(DOCKER_COMPOSE_CMD) up -d
	@echo "$(GREEN)Приложение запущено! Логи: make logs$(NC)"

dev: check-docker ## Запустить в режиме разработки (с выводом логов)
	@echo "$(GREEN)Запуск в режиме разработки...$(NC)"
	$(DOCKER_COMPOSE_CMD) up

stop: check-docker ## Остановить приложение
	@echo "$(YELLOW)Остановка приложения...$(NC)"
	$(DOCKER_COMPOSE_CMD) down

clean: check-docker ## Остановить и удалить все контейнеры и volumes
	@echo "$(RED)Полная очистка (включая данные БД)...$(NC)"
	$(DOCKER_COMPOSE_CMD) down -v
	docker system prune -f

logs: check-docker ## Показать логи приложения (Docker + файловые логи)
	@echo "$(GREEN)Показываем логи приложения...$(NC)"
	@echo "$(YELLOW)Docker логи:$(NC)"
	@$(DOCKER_COMPOSE_CMD) logs --tail=20 app || true
	@echo "\n$(YELLOW)Файловые логи (если есть):$(NC)"
	@if [ -f "./logs/app.log" ]; then \
		tail -f ./logs/app.log; \
	else \
		echo "Файл ./logs/app.log не найден. Используйте 'make logs-docker' для Docker логов."; \
		$(DOCKER_COMPOSE_CMD) logs -f app; \
	fi

logs-docker: check-docker ## Показать только Docker логи (stdout)
	$(DOCKER_COMPOSE_CMD) logs -f app

logs-file: ## Показать только файловые логи
	@if [ -f "./logs/app.log" ]; then \
		tail -f ./logs/app.log; \
	else \
		echo "$(RED)Файл ./logs/app.log не найден!$(NC)"; \
		echo "$(YELLOW)Убедитесь, что LOG_OUTPUT=logs/app.log в .env$(NC)"; \
	fi

logs-db: check-docker ## Показать логи базы данных
	$(DOCKER_COMPOSE_CMD) logs -f db

# Команды для переключения режимов логирования
logs-to-stdout: check-docker ## Переключить логи в stdout (для продакшена)
	@echo "$(GREEN)Переключаем логи в stdout режим...$(NC)"
	@echo "LOG_OUTPUT=stdout" > .env.logging
	@echo "LOG_FORMAT=json" >> .env.logging
	@echo "$(YELLOW)Перезапускаем приложение...$(NC)"
	$(DOCKER_COMPOSE_CMD) down
	$(DOCKER_COMPOSE_CMD) up -d
	@echo "$(GREEN)Теперь используйте 'make logs-docker' для просмотра логов$(NC)"

logs-to-file: check-docker ## Переключить логи в файл (для разработки)
	@echo "$(GREEN)Переключаем логи в файловый режим...$(NC)"
	@echo "LOG_OUTPUT=logs/app.log" > .env.logging
	@echo "LOG_FORMAT=json" >> .env.logging
	@echo "$(YELLOW)Перезапускаем приложение...$(NC)"
	$(DOCKER_COMPOSE_CMD) down
	$(DOCKER_COMPOSE_CMD) up -d
	@echo "$(GREEN)Теперь используйте 'make logs-file' для просмотра логов$(NC)"

status: check-docker ## Показать статус сервисов
	$(DOCKER_COMPOSE_CMD) ps

restart: check-docker ## Перезапустить приложение
	@echo "$(YELLOW)Перезапуск приложения...$(NC)"
	$(DOCKER_COMPOSE_CMD) restart app

rebuild: check-docker ## Пересобрать и перезапустить приложение
	@echo "$(GREEN)Пересборка и перезапуск...$(NC)"
	$(DOCKER_COMPOSE_CMD) down
	$(DOCKER_COMPOSE_CMD) build --no-cache
	$(DOCKER_COMPOSE_CMD) up -d

# Локальная разработка без Docker
local-db: check-docker ## Запустить только PostgreSQL для локальной разработки
	$(DOCKER_COMPOSE_CMD) up -d db

local-run: local-db ## Запустить приложение локально с Docker PostgreSQL
	@echo "$(GREEN)Ожидание готовности базы данных...$(NC)"
	@until $(DOCKER_COMPOSE_CMD) exec db pg_isready -U postgres; do sleep 1; done
	@echo "$(GREEN)База данных готова. Запуск приложения...$(NC)"
	go run main.go 
# Команды для валидации
validate-config: ## Проверить валидность конфигурации
	@echo "$(GREEN)Проверка конфигурации...$(NC)"
	@if command -v go >/dev/null 2>&1; then \
		go run cmd/validate-config/main.go; \
	else \
		echo "$(YELLOW)Go не найден, используем Docker...$(NC)"; \
		$(MAKE) validate-config-docker; \
	fi

validate-config-docker: check-docker ## Проверить конфигурацию через Docker
	@echo "$(GREEN)Проверка конфигурации через Docker...$(NC)"
	docker run --rm \
		-v $(PWD):/app \
		-w /app \
		-e TELEGRAM_BOT_TOKEN \
		-e GEMINI_API_KEY \
		-e DB_HOST \
		-e DB_PORT \
		-e DB_USER \
		-e DB_PASSWORD \
		-e DB_NAME \
		-e LOG_LEVEL \
		-e LOG_OUTPUT \
		-e LOG_FORMAT \
		--env-file .env \
		golang:1.21-alpine \
		sh -c "go mod download && go run cmd/validate-config/main.go"

test-config: ## Протестировать валидацию с разными параметрами
	@echo "$(GREEN)Тестирование валидации конфигурации...$(NC)"
	@echo "\n$(YELLOW)1. Тест с пустыми обязательными параметрами:$(NC)"
	@TELEGRAM_BOT_TOKEN="" GEMINI_API_KEY="" go run cmd/validate-config/main.go || true
	@echo "\n$(YELLOW)2. Тест с неправильным портом БД:$(NC)"
	@DB_PORT="invalid" go run cmd/validate-config/main.go || true
	@echo "\n$(YELLOW)3. Тест с неправильным форматом логов:$(NC)"
	@LOG_FORMAT="xml" go run cmd/validate-config/main.go || true
	@echo "\n$(YELLOW)4. Тест с правильной конфигурацией:$(NC)"
	@go run cmd/validate-config/main.go 