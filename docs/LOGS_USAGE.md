# Использование системы логирования

## Команды для просмотра логов

### Основные команды
```bash
make logs          # Универсальная команда - показывает Docker + файловые логи
make logs-file     # Только файловые логи (для разработки)
make logs-docker   # Только Docker логи (для продакшена)
make logs-db       # Логи базы данных
```

### Переключение режимов
```bash
make logs-to-file    # Переключить в файловый режим (разработка)
make logs-to-stdout  # Переключить в stdout режим (продакшен)
```

## Режимы логирования

### 📁 Файловый режим (разработка)
- **Логи записываются в**: `./logs/app.log`
- **Формат**: JSON
- **Просмотр**: `make logs-file`
- **Преимущества**: 
  - Логи сохраняются между перезапусками
  - Удобно для отладки
  - Можно анализировать историю

### 📺 Stdout режим (продакшен)
- **Логи выводятся в**: stdout контейнера
- **Формат**: JSON
- **Просмотр**: `make logs-docker`
- **Преимущества**:
  - Соответствует 12-Factor App
  - Интеграция с системами мониторинга
  - Автоматический сбор логов

## Примеры использования

### Разработка
```bash
# Переключиться в режим разработки
make logs-to-file

# Смотреть логи в реальном времени
make logs-file

# Или универсальная команда
make logs
```

### Продакшен
```bash
# Переключиться в продакшен режим
make logs-to-stdout

# Смотреть логи через Docker
make logs-docker

# Интеграция с внешними системами
docker compose logs app | your-log-aggregator
```

### Отладка
```bash
# Посмотреть последние 50 строк
docker compose logs --tail=50 app

# Логи с временными метками
docker compose logs -t app

# Логи конкретного сервиса
make logs-db  # База данных
```

## Конфигурация

Настройки логирования хранятся в `.env.logging`:
```bash
LOG_OUTPUT=logs/app.log  # или stdout
LOG_FORMAT=json          # или text
LOG_LEVEL=info          # debug, info, warn, error
```

## Структура логов

Все логи в JSON формате содержат:
```json
{
  "time": "2025-06-07T12:13:16.085742923Z",
  "level": "INFO",
  "source": {
    "function": "github.com/vladimiradmaev/diabetes-helper/internal/services.AnalyzeFood",
    "file": "/app/internal/services/ai_service.go",
    "line": 118
  },
  "msg": "Food analysis completed successfully",
  "carbs": 75,
  "confidence": "medium",
  "food_items_count": 1
}
```

## Мониторинг в продакшене

Для продакшена рекомендуется:
1. Использовать `make logs-to-stdout`
2. Настроить сбор логов через ELK Stack, Grafana Loki или аналоги
3. Настроить алерты на ERROR уровень
4. Мониторить метрики через структурированные поля 