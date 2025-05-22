# Diabetes Helper Bot

Telegram бот для анализа углеводов в еде с использованием AI (Gemini и OpenAI).

## Функциональность

- Регистрация пользователей на основе данных Telegram
- Анализ количества углеводов в еде по фотографии
- Поддержка указания веса блюда
- Автоматический расчет веса, если не указан
- Использование Gemini AI (основной) и OpenAI (резервный) для анализа

## Требования

- Go 1.21 или выше
- PostgreSQL
- API ключи:
  - Telegram Bot Token
  - Gemini API Key
  - OpenAI API Key

## Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/vladimiradmaev/diabetes-helper.git
cd diabetes-helper
```

2. Установите зависимости:
```bash
go mod download
```

3. Создайте файл `.env` с необходимыми переменными окружения:
```env
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
GEMINI_API_KEY=your_gemini_api_key
OPENAI_API_KEY=your_openai_api_key
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=diabetes_helper
```

4. Создайте базу данных PostgreSQL:
```sql
CREATE DATABASE diabetes_helper;
```

## Запуск

```bash
go run main.go
```

## Использование

1. Найдите бота в Telegram по его username
2. Отправьте команду `/start` для начала работы
3. Отправьте фотографию еды для анализа
4. Опционально укажите вес блюда в граммах в подписи к фото

## Разработка

### Структура проекта

```
.
├── main.go                 # Точка входа
├── internal/
│   ├── bot/               # Telegram бот
│   ├── config/            # Конфигурация
│   ├── database/          # Работа с БД
│   └── services/          # Бизнес-логика
└── README.md
```

### Тестирование

```bash
go test ./...
```

## Лицензия

MIT 