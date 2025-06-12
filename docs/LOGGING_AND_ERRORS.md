# Система логирования и обработки ошибок

## Структурированное логирование

Приложение использует встроенный пакет `log/slog` из Go 1.21+ для структурированного логирования.

### Конфигурация

Логирование настраивается через переменные окружения:

```bash
LOG_LEVEL=info          # debug, info, warn, error
LOG_OUTPUT=logs/app.log # путь к файлу или "stdout"
LOG_FORMAT=json         # json или text
```

### Использование

```go
import "github.com/vladimiradmaev/diabetes-helper/internal/logger"

// Простое логирование
logger.Info("User logged in", "user_id", 123, "username", "john")
logger.Error("Database error", "error", err, "query", "SELECT * FROM users")

// С контекстом
logger.InfoContext(ctx, "Processing request", "request_id", reqID)

// Получение логгера для сервиса
type MyService struct {
    logger *slog.Logger
}

func NewMyService() *MyService {
    return &MyService{
        logger: logger.GetLogger(),
    }
}

func (s *MyService) DoSomething(ctx context.Context) {
    s.logger.InfoContext(ctx, "Doing something", "operation", "important")
}
```

### Уровни логирования

- **Debug**: Детальная отладочная информация
- **Info**: Общая информация о работе приложения
- **Warn**: Предупреждения, не критичные ошибки
- **Error**: Ошибки, требующие внимания

## Система обработки ошибок

### Типы ошибок

```go
import apperrors "github.com/vladimiradmaev/diabetes-helper/internal/errors"

// Типы ошибок
ErrorTypeValidation   // Ошибки валидации
ErrorTypeDatabase     // Ошибки базы данных
ErrorTypeExternal     // Ошибки внешних API
ErrorTypeInternal     // Внутренние ошибки
ErrorTypePermission   // Ошибки доступа
ErrorTypeRateLimit    // Превышение лимитов
ErrorTypeTimeout      // Таймауты
```

### Создание ошибок

```go
// Новая ошибка
err := apperrors.New(apperrors.ErrorTypeValidation, "INVALID_EMAIL", "Invalid email format")

// Обертка существующей ошибки
err := apperrors.Wrap(originalErr, apperrors.ErrorTypeDatabase, "DB_QUERY", "Failed to query users")

// Добавление контекста
err = err.WithContext("user_id", 123).WithContext("table", "users")

// Готовые конструкторы
err := apperrors.NewValidationError("Email is required")
err := apperrors.NewDatabaseError(sqlErr)
err := apperrors.NewExternalAPIError(httpErr, "Gemini")
err := apperrors.NewTimeoutError("database_query")
```

### Обработка ошибок

```go
// Создание обработчика
errorHandler := apperrors.NewHandler(logger.GetLogger())

// Обработка ошибки (автоматическое логирование)
errorHandler.Handle(ctx, err)

// Логирование и возврат ошибки
return errorHandler.LogAndReturn(ctx, err)

// Проверка типа ошибки
var appErr *apperrors.AppError
if errors.As(err, &appErr) {
    switch appErr.Type {
    case apperrors.ErrorTypeValidation:
        // Обработка ошибки валидации
    case apperrors.ErrorTypeDatabase:
        // Обработка ошибки БД
    }
}
```

### Структура AppError

```go
type AppError struct {
    Type     ErrorType                  // Тип ошибки
    Message  string                     // Сообщение для пользователя
    Code     string                     // Код ошибки
    Internal error                      // Исходная ошибка
    Context  map[string]interface{}     // Дополнительный контекст
    Source   string                     // Место возникновения ошибки
}
```

## Примеры использования

### В сервисах

```go
func (s *AIService) AnalyzeImage(ctx context.Context, imageURL string) (*Result, error) {
    s.logger.InfoContext(ctx, "Starting image analysis", "url", imageURL)
    
    resp, err := http.Get(imageURL)
    if err != nil {
        return nil, apperrors.NewExternalAPIError(err, "HTTP").
            WithContext("url", imageURL).
            WithContext("operation", "download_image")
    }
    
    s.logger.InfoContext(ctx, "Image downloaded successfully", "size", resp.ContentLength)
    return result, nil
}
```

### В обработчиках

```go
func (h *Handler) HandleRequest(ctx context.Context, req *Request) error {
    if err := h.validateRequest(req); err != nil {
        return apperrors.NewValidationError("Invalid request format").
            WithContext("request_id", req.ID)
    }
    
    result, err := h.service.Process(ctx, req)
    if err != nil {
        h.errorHandler.Handle(ctx, err)
        return err
    }
    
    h.logger.InfoContext(ctx, "Request processed successfully", 
        "request_id", req.ID,
        "duration", time.Since(start))
    return nil
}
```

## Преимущества новой системы

1. **Структурированные логи**: Легко парсятся и анализируются
2. **Контекстная информация**: Каждая ошибка содержит полный контекст
3. **Типизированные ошибки**: Разные стратегии обработки для разных типов
4. **Централизованная обработка**: Единая точка логирования ошибок
5. **Трассировка**: Автоматическое определение места возникновения ошибки
6. **Производительность**: Использование slog для высокой производительности

## Миграция со старой системы

1. Замените `logger.Infof()` на `logger.Info()` с структурированными полями
2. Замените `fmt.Errorf()` на `apperrors.New()` или `apperrors.Wrap()`
3. Добавьте контекст к ошибкам через `WithContext()`
4. Используйте `errorHandler.Handle()` для централизованной обработки
5. Передавайте `context.Context` в методы логирования 