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

logs: check-docker ## Показать логи приложения
	$(DOCKER_COMPOSE_CMD) logs -f app

logs-db: check-docker ## Показать логи базы данных
	$(DOCKER_COMPOSE_CMD) logs -f db

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