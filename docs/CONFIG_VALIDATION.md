# Валидация конфигурации

## Обзор

Приложение автоматически валидирует все параметры конфигурации при запуске. Если конфигурация содержит ошибки, приложение завершится с детальным описанием проблем.

## Команды для проверки

```bash
make validate-config  # Проверить текущую конфигурацию
make test-config      # Протестировать валидацию с разными параметрами
```

## Обязательные параметры

### TELEGRAM_BOT_TOKEN
- **Обязательный**: Да
- **Формат**: `bot_id:auth_token`
- **Пример**: `123456789:AAHdqTcvfc1vasJxaSeoaSAs1K5PALDsaw`
- **Валидация**:
  - Длина: 35-50 символов
  - Содержит двоеточие (:)
  - bot_id: 8-12 цифр
  - auth_token: 25-40 символов

### GEMINI_API_KEY
- **Обязательный**: Да
- **Формат**: Начинается с `AIza`
- **Пример**: `AlzaayDakmmKa4JszZ-HjGw47_X2-19_21rqbOE`
- **Валидация**:
  - Длина: 35-45 символов
  - Начинается с "AIza"

## Параметры базы данных

### DB_HOST
- **Обязательный**: Да (по умолчанию: `localhost`)
- **Валидация**:
  - Не может быть пустым
  - Должен быть валидным IP-адресом или hostname
  - Исключения: `localhost`, `db` (для Docker)

### DB_PORT
- **Обязательный**: Да (по умолчанию: `5432`)
- **Валидация**:
  - Должен быть числом
  - Диапазон: 1-65535

### DB_USER
- **Обязательный**: Да (по умолчанию: `postgres`)
- **Валидация**: Не может быть пустым

### DB_PASSWORD
- **Обязательный**: Нет (по умолчанию: `postgres`)
- **Валидация**: Нет ограничений

### DB_NAME
- **Обязательный**: Да (по умолчанию: `diabetes_helper`)
- **Валидация**: Не может быть пустым

## Параметры логирования

### LOG_OUTPUT
- **Обязательный**: Да (по умолчанию: `logs/app.log`)
- **Валидация**: Не может быть пустым
- **Допустимые значения**: Любой путь к файлу или `stdout`

### LOG_FORMAT
- **Обязательный**: Да (по умолчанию: `json`)
- **Валидация**: Должен быть `json` или `text`

### LOG_LEVEL
- **Обязательный**: Нет (по умолчанию: `info`)
- **Допустимые значения**: `debug`, `info`, `warn`, `warning`, `error`

## Примеры ошибок валидации

### Пустые обязательные параметры
```
❌ Ошибка валидации конфигурации:
configuration validation failed: 
- config validation failed for field 'TELEGRAM_BOT_TOKEN' (value: ''): telegram bot token is required
- config validation failed for field 'GEMINI_API_KEY' (value: ''): gemini API key is required
```

### Неправильный формат токена
```
❌ Ошибка валидации конфигурации:
configuration validation failed: 
- config validation failed for field 'TELEGRAM_BOT_TOKEN' (value: 'inva...oken'): telegram bot token format is invalid (should be like '123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11')
```

### Неправильный порт БД
```
❌ Ошибка валидации конфигурации:
configuration validation failed: 
- config validation failed for field 'DB_PORT' (value: 'invalid'): database port must be a valid number
```

### Неправильный формат логов
```
❌ Ошибка валидации конфигурации:
configuration validation failed: 
- config validation failed for field 'LOG_FORMAT' (value: 'xml'): log format must be 'json' or 'text'
```

## Безопасность

- **Маскирование**: Чувствительные данные (токены, ключи) маскируются в сообщениях об ошибках
- **Формат маскирования**: `abcd...xyz` (первые 4 и последние 4 символа)
- **Логирование**: Валидационные ошибки не содержат полных значений секретов

## Интеграция с CI/CD

Для проверки конфигурации в CI/CD пайплайнах:

```bash
# В GitHub Actions, GitLab CI и т.д.
make validate-config
```

Команда завершится с кодом выхода:
- `0` - конфигурация валидна
- `1` - есть ошибки валидации

## Разработка

При добавлении новых параметров конфигурации:

1. Добавьте поле в соответствующую структуру (`Config`, `DBConfig`, `LoggerConfig`)
2. Добавьте валидацию в метод `Validate()`
3. Обновите `.env.example`
4. Добавьте тест в `make test-config`
5. Обновите документацию

## Примеры валидных конфигураций

### Минимальная конфигурация
```bash
TELEGRAM_BOT_TOKEN=123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw
GEMINI_API_KEY=AIzaSyDaGmWKa4JsXZ-HjGw47_X2-19_E1rZbOE
```

### Полная конфигурация
```bash
TELEGRAM_BOT_TOKEN=123456789:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsaw
GEMINI_API_KEY=AIzaSyDaGmWKa4JsXZ-HjGw47_X2-19_E1rZbOE
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=mypassword
DB_NAME=diabetes_helper
LOG_LEVEL=info
LOG_OUTPUT=logs/app.log
LOG_FORMAT=json
``` 