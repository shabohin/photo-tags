# Руководство по golangci-lint для Photo Tags Service

## Установка

1. **Автоматическая установка** (рекомендуется):
   ```bash
   ./scripts/install-golangci-lint.sh
   ```

2. **Через Makefile**:
   ```bash
   make install-tools
   ```

3. **Вручную**:
   ```bash
   # Via Go
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
   
   # Via Homebrew (macOS)
   brew install golangci-lint
   ```

## Использование

### Основные команды

```bash
# Запуск линтера на всех модулях
make lint

# Запуск линтера с автоисправлением
make lint-fix

# Форматирование кода
make fmt

# Полная проверка (форматирование + линтинг + тесты)
make pre-commit

# Проверка всех модулей (тесты + линтинг)
make check
```

### Работа с отдельными сервисами

```bash
# Линтинг конкретного сервиса
cd services/gateway
golangci-lint run

# С автоисправлением
cd services/gateway
golangci-lint run --fix

# Только определенные линтеры
cd services/gateway
golangci-lint run --enable=errcheck,govet
```

## Настройка IDE

### VS Code

1. Установите расширение Go
2. Настройки уже добавлены в `.vscode/settings.json`
3. Линтер будет запускаться автоматически при сохранении

### GoLand/IntelliJ IDEA

1. Go to: Settings → Tools → Go Linter
2. Enable: golangci-lint
3. Set path to golangci-lint binary

## Git Hooks

Установка pre-commit хука:

```bash
make install-hooks
```

Хук будет запускать:
1. Форматирование кода (gofmt + goimports)
2. Линтинг (golangci-lint)
3. Тесты (go test)

## Конфигурация

Основная конфигурация в `.golangci.yml`:

- **Таймаут**: 5 минут
- **Включенные линтеры**:
  - errcheck (проверка необработанных ошибок)
  - gosimple (упрощение кода)
  - govet (статический анализ)
  - ineffassign (неиспользуемые присваивания)
  - staticcheck (расширенный статический анализ)
  - gofmt/goimports (форматирование)
  - revive (стиль кода)
  - misspell (орфография в комментариях)

## Частые проблемы и решения

### 1. Ошибки errcheck

```go
// Плохо
file.Close()

// Хорошо
if err := file.Close(); err != nil {
    log.Printf("Failed to close file: %v", err)
}

// Или если ошибка не критична
_ = file.Close()
```

### 2. Отсутствие комментариев к экспортируемым функциям

```go
// Плохо
func ProcessImage() {}

// Хорошо
// ProcessImage processes the uploaded image and generates metadata
func ProcessImage() {}
```

### 3. Проблемы с импортами

```bash
# Автоисправление импортов
make fmt

# Или вручную
goimports -w -local github.com/shabohin/photo-tags .
```

### 4. Shadow переменных

```go
// Плохо
if err != nil {
    client, err := NewClient() // переменная err затеняется
}

// Хорошо
if err != nil {
    client, clientErr := NewClient()
    if clientErr != nil {
        // обработка ошибки
    }
}
```

## Интеграция с CI/CD

GitHub Actions автоматически запускает линтинг для каждого PR:
- Проверяет все модули
- Блокирует мерж при наличии ошибок
- Показывает результаты в интерфейсе GitHub

## Полезные команды

```bash
# Показать все доступные линтеры
golangci-lint linters

# Запустить только определенные линтеры
golangci-lint run --enable=errcheck,govet,staticcheck

# Исключить определенные файлы
golangci-lint run --skip-files=".*_test.go"

# Показать версию
golangci-lint version

# Помощь
golangci-lint help
```

## Советы по производительности

1. **Используйте кэш**: golangci-lint автоматически кэширует результаты
2. **Запускайте только измененные файлы**: `golangci-lint run --new-from-rev=HEAD~1`
3. **Настройте исключения**: добавьте правила в `.golangci.yml` для известных проблем
4. **Используйте fast режим**: `golangci-lint run --fast` (для разработки)

## Обновление

```bash
# Проверить текущую версию
golangci-lint version

# Обновить до последней версии
./scripts/install-golangci-lint.sh
```

---

**Полезные ссылки:**
- [Официальная документация golangci-lint](https://golangci-lint.run/)
- [Список всех линтеров](https://golangci-lint.run/usage/linters/)
- [Конфигурация](https://golangci-lint.run/usage/configuration/)
