# Processor Service - Техническое задание

**Приоритет**: 🔴 КРИТИЧЕСКИЙ
**Статус**: Not Started
**Оценка времени**: 5-7 дней
**Блокирует**: MVP проекта

---

## 📋 Описание

Processor Service - микросервис для записи AI-сгенерированных метаданных (title, description, keywords) в файлы изображений. Это критический компонент, завершающий цепочку обработки изображений в системе Photo Tags.

### Место в архитектуре

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Gateway   │ ───► │   Analyzer   │ ───► │  Processor  │ ───► Gateway
└─────────────┘      └──────────────┘      └─────────────┘
      ▲                                            │
      │                                            │
      └────────────────────────────────────────────┘
```

**Входные данные**: Сообщения из очереди `metadata_generated` с метаданными от Analyzer Service
**Выходные данные**: Обработанные изображения с записанными метаданными в MinIO + сообщения в очередь `image_processed`

---

## 🎯 Функциональные требования

### 1. Обработка сообщений из RabbitMQ

**Consumer для очереди `metadata_generated`:**
- Получение сообщений типа `MetadataGenerated` (см. `pkg/models/messages.go:17-25`)
- Извлечение данных: trace_id, original_path, metadata (title, description, keywords)
- Parallel worker pool для обработки нескольких изображений одновременно

### 2. Работа с MinIO

**Операции с object storage:**
- Скачивание оригинального изображения из bucket `original` по пути `original_path`
- Загрузка обработанного изображения в bucket `processed`
- Именование: `processed/{trace_id}/{original_filename}`
- Timeout для операций: 30 секунд (настраиваемый)
- Retry механизм при временных ошибках

### 3. Запись метаданных в изображение

**Интеграция с ExifTool:**

ExifTool - внешняя утилита для записи метаданных в изображения. Необходимо:

```go
// Псевдокод для интеграции
type ExifToolWrapper interface {
    WriteMetadata(imagePath string, metadata Metadata) error
}

// Записать метаданные в следующие теги:
// - EXIF: ImageDescription, XPTitle, XPKeywords
// - IPTC: Caption-Abstract, Headline, Keywords
// - XMP: dc:title, dc:description, dc:subject
```

**Метаданные для записи:**
- **Title** → EXIF:XPTitle, IPTC:Headline, XMP:dc:title
- **Description** → EXIF:ImageDescription, IPTC:Caption-Abstract, XMP:dc:description
- **Keywords** → EXIF:XPKeywords, IPTC:Keywords, XMP:dc:subject (массив)

**Поддерживаемые форматы:** JPG, JPEG, PNG

**Требования:**
- Сохранение оригинального качества изображения
- Не изменять визуальное содержимое
- Корректная обработка Unicode (русский и английский текст)
- Валидация результата: проверка что метаданные записались

### 4. Публикация результата

**Publisher для очереди `image_processed`:**

Отправка сообщения типа `ImageProcessed` (см. `pkg/models/messages.go:40-50`):

```json
{
  "trace_id": "uuid",
  "group_id": "uuid",
  "telegram_id": 123456789,
  "telegram_username": "username",
  "original_filename": "image.jpg",
  "processed_path": "processed/uuid/image.jpg",
  "status": "completed",
  "error": "",
  "timestamp": "2025-11-18T12:00:00Z"
}
```

**Статусы:**
- `completed` - успешная обработка
- `failed` - ошибка обработки (с описанием в поле `error`)

---

## 🏗️ Архитектура и структура проекта

### Структура директорий

Следовать структуре Analyzer Service:

```
services/processor/
├── cmd/
│   └── main.go                      # Точка входа
├── internal/
│   ├── app/
│   │   └── app.go                   # Инициализация приложения, worker pool
│   ├── config/
│   │   ├── config.go                # Конфигурация из env переменных
│   │   └── config_test.go
│   ├── domain/
│   │   ├── model/
│   │   │   └── message.go           # Внутренние модели (если нужны)
│   │   └── service/
│   │       ├── interfaces.go        # Интерфейсы для DI
│   │       ├── processor.go         # Основная бизнес-логика
│   │       ├── processor_test.go
│   │       └── message_processor.go # Обработчик RabbitMQ сообщений
│   ├── exiftool/
│   │   ├── client.go                # Wrapper для ExifTool
│   │   ├── client_test.go
│   │   └── mock.go                  # Mock для тестов
│   ├── storage/
│   │   └── minio/
│   │       ├── client.go            # MinIO клиент
│   │       └── client_test.go
│   └── transport/
│       └── rabbitmq/
│           ├── consumer.go          # RabbitMQ consumer
│           ├── publisher.go         # RabbitMQ publisher
│           ├── consumer_test.go
│           └── publisher_test.go
├── go.mod
├── go.sum
└── Dockerfile
```

### Компоненты

#### 1. Config (`internal/config/config.go`)

```go
type Config struct {
    Log struct {
        Level  string
        Format string
    }

    RabbitMQ struct {
        URL               string
        ConsumerQueue     string  // "metadata_generated"
        PublisherQueue    string  // "image_processed"
        ReconnectDelay    time.Duration
        ReconnectAttempts int
        PrefetchCount     int
    }

    MinIO struct {
        Endpoint         string
        AccessKey        string
        SecretKey        string
        OriginalBucket   string  // "original"
        ProcessedBucket  string  // "processed"
        DownloadTimeout  time.Duration
        UploadTimeout    time.Duration
        ConnectAttempts  int
        ConnectDelay     time.Duration
        UseSSL           bool
    }

    ExifTool struct {
        BinaryPath      string  // Путь к exiftool binary
        TempDir         string  // Директория для временных файлов
        CommandTimeout  time.Duration
    }

    Worker struct {
        Concurrency int           // Количество параллельных воркеров
        MaxRetries  int           // Максимум попыток обработки
        RetryDelay  time.Duration // Задержка между попытками
    }
}
```

**Переменные окружения:**
```bash
# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
RABBITMQ_CONSUMER_QUEUE=metadata_generated
RABBITMQ_PUBLISHER_QUEUE=image_processed
RABBITMQ_PREFETCH_COUNT=1
RABBITMQ_RECONNECT_ATTEMPTS=5
RABBITMQ_RECONNECT_DELAY=5s

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_ORIGINAL_BUCKET=original
MINIO_PROCESSED_BUCKET=processed
MINIO_DOWNLOAD_TIMEOUT=30s
MINIO_UPLOAD_TIMEOUT=30s
MINIO_CONNECT_ATTEMPTS=5
MINIO_CONNECT_DELAY=3s

# ExifTool
EXIFTOOL_BINARY_PATH=/usr/bin/exiftool
EXIFTOOL_TEMP_DIR=/tmp/processor
EXIFTOOL_COMMAND_TIMEOUT=10s

# Worker
WORKER_CONCURRENCY=3
WORKER_MAX_RETRIES=3
WORKER_RETRY_DELAY=5s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

#### 2. ExifTool Client (`internal/exiftool/client.go`)

```go
package exiftool

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
)

type Client struct {
    binaryPath string
    timeout    time.Duration
    logger     *logrus.Logger
}

type Metadata struct {
    Title       string
    Description string
    Keywords    []string
}

func NewClient(binaryPath string, timeout time.Duration, logger *logrus.Logger) *Client {
    return &Client{
        binaryPath: binaryPath,
        timeout:    timeout,
        logger:     logger,
    }
}

// WriteMetadata записывает метаданные в изображение
// Возвращает путь к обработанному изображению
func (c *Client) WriteMetadata(ctx context.Context, imagePath string, metadata Metadata, traceID string) error {
    // Проверка существования exiftool
    if _, err := exec.LookPath(c.binaryPath); err != nil {
        return fmt.Errorf("exiftool not found: %w", err)
    }

    // Создание команды с таймаутом
    ctx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()

    // Формирование аргументов для exiftool
    args := []string{
        "-overwrite_original",     // Перезаписать оригинал
        "-charset", "utf8",        // UTF-8 для русского текста

        // EXIF теги
        fmt.Sprintf("-XPTitle=%s", metadata.Title),
        fmt.Sprintf("-ImageDescription=%s", metadata.Description),

        // IPTC теги
        fmt.Sprintf("-IPTC:Headline=%s", metadata.Title),
        fmt.Sprintf("-IPTC:Caption-Abstract=%s", metadata.Description),

        // XMP теги
        fmt.Sprintf("-XMP:Title=%s", metadata.Title),
        fmt.Sprintf("-XMP:Description=%s", metadata.Description),

        imagePath,
    }

    // Добавление keywords
    for _, keyword := range metadata.Keywords {
        args = append(args, fmt.Sprintf("-IPTC:Keywords=%s", keyword))
        args = append(args, fmt.Sprintf("-XMP:Subject=%s", keyword))
    }

    // Выполнение команды
    cmd := exec.CommandContext(ctx, c.binaryPath, args...)

    output, err := cmd.CombinedOutput()
    if err != nil {
        c.logger.WithFields(logrus.Fields{
            "trace_id": traceID,
            "output":   string(output),
            "error":    err.Error(),
        }).Error("ExifTool command failed")
        return fmt.Errorf("exiftool failed: %w, output: %s", err, string(output))
    }

    c.logger.WithFields(logrus.Fields{
        "trace_id": traceID,
        "output":   string(output),
    }).Debug("ExifTool command succeeded")

    return nil
}

// VerifyMetadata проверяет что метаданные записались корректно
func (c *Client) VerifyMetadata(ctx context.Context, imagePath string, traceID string) (bool, error) {
    ctx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()

    // Читаем метаданные обратно
    cmd := exec.CommandContext(ctx, c.binaryPath, "-j", "-Title", "-Description", "-Keywords", imagePath)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return false, fmt.Errorf("verification failed: %w", err)
    }

    // Проверяем что output не пустой
    return len(output) > 0 && !strings.Contains(string(output), "error"), nil
}
```

**Тесты:**
- Unit тесты с mock exec.Command
- Интеграционные тесты с реальными изображениями
- Тест обработки Unicode
- Тест обработки спецсимволов в метаданных

#### 3. Image Processor Service (`internal/domain/service/processor.go`)

```go
package service

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
)

type ImageProcessorService struct {
    minioClient  MinioClientInterface
    exifTool     ExifToolInterface
    logger       *logrus.Logger
    tempDir      string
}

func NewImageProcessor(
    minioClient MinioClientInterface,
    exifTool ExifToolInterface,
    tempDir string,
    logger *logrus.Logger,
) *ImageProcessorService {
    return &ImageProcessorService{
        minioClient: minioClient,
        exifTool:    exifTool,
        tempDir:     tempDir,
        logger:      logger,
    }
}

// ProcessImage обрабатывает одно изображение: скачивает, записывает метаданные, загружает
func (s *ImageProcessorService) ProcessImage(
    ctx context.Context,
    originalPath string,
    processedPath string,
    metadata model.Metadata,
    traceID string,
) error {
    s.logger.WithFields(logrus.Fields{
        "trace_id":       traceID,
        "original_path":  originalPath,
        "processed_path": processedPath,
    }).Info("Starting image processing")

    // 1. Скачать изображение из MinIO (bucket: original)
    imageBytes, err := s.minioClient.DownloadImage(ctx, originalPath)
    if err != nil {
        s.logger.WithFields(logrus.Fields{
            "trace_id": traceID,
            "error":    err.Error(),
        }).Error("Failed to download image from MinIO")
        return fmt.Errorf("download failed: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "trace_id":      traceID,
        "image_size_kb": len(imageBytes) / 1024,
    }).Debug("Image downloaded successfully")

    // 2. Сохранить во временную директорию
    tempFilePath := filepath.Join(s.tempDir, fmt.Sprintf("%s_temp.jpg", traceID))
    if err := os.WriteFile(tempFilePath, imageBytes, 0644); err != nil {
        return fmt.Errorf("failed to write temp file: %w", err)
    }
    defer os.Remove(tempFilePath) // Cleanup

    // 3. Записать метаданные с помощью ExifTool
    if err := s.exifTool.WriteMetadata(ctx, tempFilePath, metadata, traceID); err != nil {
        s.logger.WithFields(logrus.Fields{
            "trace_id": traceID,
            "error":    err.Error(),
        }).Error("Failed to write metadata")
        return fmt.Errorf("metadata write failed: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "trace_id": traceID,
        "title":    metadata.Title,
        "keywords": len(metadata.Keywords),
    }).Info("Metadata written successfully")

    // 4. Верифицировать что метаданные записались
    verified, err := s.exifTool.VerifyMetadata(ctx, tempFilePath, traceID)
    if err != nil || !verified {
        s.logger.WithFields(logrus.Fields{
            "trace_id": traceID,
            "verified": verified,
        }).Warn("Metadata verification failed")
    }

    // 5. Загрузить обработанное изображение в MinIO (bucket: processed)
    processedImageBytes, err := os.ReadFile(tempFilePath)
    if err != nil {
        return fmt.Errorf("failed to read processed file: %w", err)
    }

    if err := s.minioClient.UploadImage(ctx, processedPath, processedImageBytes); err != nil {
        s.logger.WithFields(logrus.Fields{
            "trace_id": traceID,
            "error":    err.Error(),
        }).Error("Failed to upload processed image to MinIO")
        return fmt.Errorf("upload failed: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "trace_id":       traceID,
        "processed_path": processedPath,
    }).Info("Image processing completed successfully")

    return nil
}
```

#### 4. Message Processor (`internal/domain/service/message_processor.go`)

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/shabohin/photo-tags/pkg/models"
)

type MessageProcessorService struct {
    imageProcessor ImageProcessorInterface
    publisher      PublisherInterface
    logger         *logrus.Logger
    maxRetries     int
    retryDelay     time.Duration
}

func NewMessageProcessor(
    imageProcessor ImageProcessorInterface,
    publisher PublisherInterface,
    logger *logrus.Logger,
    maxRetries int,
    retryDelay time.Duration,
) *MessageProcessorService {
    return &MessageProcessorService{
        imageProcessor: imageProcessor,
        publisher:      publisher,
        logger:         logger,
        maxRetries:     maxRetries,
        retryDelay:     retryDelay,
    }
}

// Process обрабатывает сообщение из RabbitMQ
func (s *MessageProcessorService) Process(ctx context.Context, messageBody []byte) error {
    // Парсинг сообщения
    var msg models.MetadataGenerated
    if err := json.Unmarshal(messageBody, &msg); err != nil {
        s.logger.WithError(err).Error("Failed to unmarshal message")
        return fmt.Errorf("unmarshal failed: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "trace_id":   msg.TraceID,
        "group_id":   msg.GroupID,
        "telegram_id": msg.TelegramID,
    }).Info("Processing metadata_generated message")

    // Формирование processed_path
    processedPath := fmt.Sprintf("processed/%s/%s", msg.TraceID, msg.OriginalFilename)

    // Обработка с retry логикой
    var lastErr error
    for attempt := 0; attempt < s.maxRetries; attempt++ {
        if attempt > 0 {
            s.logger.WithFields(logrus.Fields{
                "trace_id": msg.TraceID,
                "attempt":  attempt,
            }).Info("Retrying image processing")
            time.Sleep(s.retryDelay)
        }

        err := s.imageProcessor.ProcessImage(
            ctx,
            msg.OriginalPath,
            processedPath,
            msg.Metadata,
            msg.TraceID,
        )

        if err == nil {
            // Успех - отправить completed message
            return s.publishResult(ctx, msg, processedPath, "completed", "")
        }

        lastErr = err
        s.logger.WithFields(logrus.Fields{
            "trace_id": msg.TraceID,
            "attempt":  attempt,
            "error":    err.Error(),
        }).Warn("Image processing attempt failed")
    }

    // Все попытки исчерпаны - отправить failed message
    s.logger.WithFields(logrus.Fields{
        "trace_id": msg.TraceID,
        "error":    lastErr.Error(),
    }).Error("Image processing failed after all retries")

    return s.publishResult(ctx, msg, "", "failed", lastErr.Error())
}

// publishResult отправляет результат в очередь image_processed
func (s *MessageProcessorService) publishResult(
    ctx context.Context,
    originalMsg models.MetadataGenerated,
    processedPath string,
    status string,
    errorMsg string,
) error {
    result := models.ImageProcessed{
        TraceID:          originalMsg.TraceID,
        GroupID:          originalMsg.GroupID,
        TelegramID:       originalMsg.TelegramID,
        TelegramUsername: "", // Will be filled by Gateway
        OriginalFilename: originalMsg.OriginalFilename,
        ProcessedPath:    processedPath,
        Status:           status,
        Error:            errorMsg,
        Timestamp:        time.Now(),
    }

    resultBytes, err := json.Marshal(result)
    if err != nil {
        return fmt.Errorf("failed to marshal result: %w", err)
    }

    if err := s.publisher.Publish(ctx, resultBytes); err != nil {
        s.logger.WithFields(logrus.Fields{
            "trace_id": originalMsg.TraceID,
            "error":    err.Error(),
        }).Error("Failed to publish result")
        return fmt.Errorf("publish failed: %w", err)
    }

    s.logger.WithFields(logrus.Fields{
        "trace_id": originalMsg.TraceID,
        "status":   status,
    }).Info("Result published successfully")

    return nil
}
```

#### 5. App Initialization (`internal/app/app.go`)

Аналогично Analyzer Service (см. выше), но с Processor-специфичными компонентами:
- MinioClient для `original` и `processed` buckets
- ExifTool client
- ImageProcessor service
- MessageProcessor service
- RabbitMQ Consumer/Publisher
- Worker pool

---

## 🧪 Тестирование

### Unit тесты

**Для каждого компонента:**

1. **ExifTool Client** (`exiftool/client_test.go`):
   - Тест записи метаданных
   - Тест обработки Unicode
   - Тест обработки ошибок exec
   - Тест timeout
   - Mock для exec.Command

2. **Image Processor** (`domain/service/processor_test.go`):
   - Тест успешной обработки
   - Тест ошибки скачивания
   - Тест ошибки записи метаданных
   - Тест ошибки загрузки
   - Mock для MinIO и ExifTool

3. **Message Processor** (`domain/service/message_processor_test.go`):
   - Тест успешной обработки сообщения
   - Тест retry логики
   - Тест публикации completed message
   - Тест публикации failed message
   - Mock для всех зависимостей

4. **RabbitMQ Transport** (`transport/rabbitmq/*_test.go`):
   - Тест подключения
   - Тест reconnect логики
   - Тест обработки сообщений
   - Mock для RabbitMQ

### Интеграционные тесты

```go
// integration_test.go
func TestProcessorIntegration(t *testing.T) {
    // Setup: MinIO, RabbitMQ, ExifTool
    // 1. Загрузить тестовое изображение в MinIO (original bucket)
    // 2. Отправить MetadataGenerated message в RabbitMQ
    // 3. Дождаться обработки
    // 4. Проверить что изображение появилось в processed bucket
    // 5. Скачать и проверить метаданные через ExifTool
    // 6. Проверить что ImageProcessed message отправлен
}
```

**Тестовые данные:**
- Набор изображений: JPG (разных размеров), PNG
- Метаданные с Unicode (русский + английский)
- Метаданные со спецсимволами
- Большое количество keywords (49 штук)

### E2E тест (после интеграции с Gateway и Analyzer)

Полный флоу:
1. Gateway получает изображение → отправляет в image_upload
2. Analyzer анализирует → отправляет в metadata_generated
3. **Processor обрабатывает → отправляет в image_processed**
4. Gateway возвращает результат пользователю

---

## 🐳 Docker и деплой

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git

WORKDIR /app

# Копирование go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o processor ./cmd/main.go

# Финальный образ
FROM alpine:latest

# Установка ExifTool и зависимостей
RUN apk add --no-cache \
    exiftool \
    perl \
    ca-certificates \
    tzdata

WORKDIR /app

# Копирование бинарника
COPY --from=builder /app/processor .

# Создание директории для временных файлов
RUN mkdir -p /tmp/processor && chmod 777 /tmp/processor

# Запуск приложения
CMD ["./processor"]
```

### Docker Compose интеграция

Добавить в `docker/docker-compose.yml`:

```yaml
processor:
  build:
    context: ../services/processor
    dockerfile: Dockerfile
  container_name: processor
  environment:
    - RABBITMQ_URL=amqp://user:password@rabbitmq:5672/
    - RABBITMQ_CONSUMER_QUEUE=metadata_generated
    - RABBITMQ_PUBLISHER_QUEUE=image_processed
    - MINIO_ENDPOINT=minio:9000
    - MINIO_ACCESS_KEY=minioadmin
    - MINIO_SECRET_KEY=minioadmin
    - MINIO_ORIGINAL_BUCKET=original
    - MINIO_PROCESSED_BUCKET=processed
    - EXIFTOOL_BINARY_PATH=/usr/bin/exiftool
    - EXIFTOOL_TEMP_DIR=/tmp/processor
    - WORKER_CONCURRENCY=3
    - LOG_LEVEL=info
    - LOG_FORMAT=json
  depends_on:
    - rabbitmq
    - minio
  networks:
    - photo-tags
  restart: unless-stopped
  volumes:
    - processor-temp:/tmp/processor

volumes:
  processor-temp:
```

---

## 🔍 Обработка ошибок

### Типы ошибок и стратегии

1. **Transient errors** (retry):
   - Timeout подключения к MinIO
   - Timeout подключения к RabbitMQ
   - Временная недоступность ExifTool
   - **Стратегия**: Retry с exponential backoff

2. **Permanent errors** (fail fast):
   - Невалидное сообщение (JSON parse error)
   - Изображение не найдено в MinIO
   - Неподдерживаемый формат изображения
   - ExifTool не установлен
   - **Стратегия**: Немедленно отправить failed message

3. **Partial failures**:
   - Метаданные записаны, но верификация failed
   - **Стратегия**: Считать успешным, но залогировать warning

### Логирование

Все операции логировать с:
- `trace_id` - для трейсинга
- `worker_id` - идентификатор воркера
- Timestamp
- Error details
- Context (что именно делалось)

**Уровни логов:**
- DEBUG: Детали выполнения команд
- INFO: Начало/завершение обработки, успехи
- WARN: Retry попытки, частичные failures
- ERROR: Критические ошибки, failed processing

---

## ✅ Acceptance Criteria

**Processor Service считается завершенным когда:**

### Функциональность
- [ ] Consumer получает сообщения из очереди `metadata_generated`
- [ ] Изображения скачиваются из MinIO bucket `original`
- [ ] Метаданные (title, description, keywords) корректно записываются в EXIF/IPTC/XMP
- [ ] Обработанные изображения загружаются в MinIO bucket `processed`
- [ ] Publisher отправляет сообщения в очередь `image_processed` со статусом `completed` или `failed`
- [ ] Поддерживаются форматы: JPG, JPEG, PNG
- [ ] Корректная обработка Unicode (русский + английский текст)

### Надежность
- [ ] Retry механизм работает для временных ошибок (минимум 3 попытки)
- [ ] Graceful shutdown при получении SIGTERM/SIGINT
- [ ] Worker pool с настраиваемым количеством воркеров
- [ ] Timeout для всех внешних операций (MinIO, ExifTool)
- [ ] Temporary файлы очищаются после обработки

### Тестирование
- [ ] Unit тесты для всех компонентов (coverage > 80%)
- [ ] Интеграционные тесты с реальными MinIO и RabbitMQ
- [ ] Тесты обработки Unicode
- [ ] Тесты error scenarios
- [ ] Все тесты проходят: `make test`

### Code Quality
- [ ] Код проходит линтер: `make lint`
- [ ] Следует структуре Analyzer Service
- [ ] Все публичные функции документированы
- [ ] Интерфейсы определены для DI и тестирования

### Deployment
- [ ] Dockerfile собирается без ошибок
- [ ] Docker Compose конфигурация обновлена
- [ ] Environment variables документированы
- [ ] Сервис запускается и работает в Docker окружении

### Documentation
- [ ] README с описанием сервиса
- [ ] Комментарии к ключевым функциям
- [ ] Примеры использования в тестах

---

## 📚 Полезные ресурсы

### ExifTool документация
- [Official ExifTool documentation](https://exiftool.org/)
- [Tag Names](https://exiftool.org/TagNames/index.html)
- [EXIF Tags](https://exiftool.org/TagNames/EXIF.html)
- [IPTC Tags](https://exiftool.org/TagNames/IPTC.html)
- [XMP Tags](https://exiftool.org/TagNames/XMP.html)

### Go библиотеки
- [MinIO Go Client](https://github.com/minio/minio-go)
- [AMQP (RabbitMQ) for Go](https://github.com/rabbitmq/amqp091-go)
- [Logrus](https://github.com/sirupsen/logrus)

### Аналогичный код
- Analyzer Service: `/services/analyzer`
- Gateway Service: `/services/gateway`
- Shared packages: `/pkg`

---

## 🎯 Порядок реализации

### День 1-2: Базовая структура
1. Создать структуру директорий
2. Реализовать Config
3. Реализовать ExifTool client + тесты
4. Настроить Dockerfile с ExifTool

### День 3-4: Core логика
5. Реализовать MinIO client (скачивание + загрузка)
6. Реализовать ImageProcessor service
7. Реализовать MessageProcessor service
8. Unit тесты для всех компонентов

### День 5: Интеграция
9. RabbitMQ Consumer/Publisher
10. App initialization с worker pool
11. Docker Compose интеграция

### День 6-7: Тестирование и финализация
12. Интеграционные тесты
13. E2E тестирование с полным флоу
14. Исправление багов
15. Code review и рефакторинг
16. Документация

---

## 🚀 После завершения

Когда Processor Service будет готов:
1. ✅ **MVP системы будет завершен!**
2. Запустить full E2E тест всей системы
3. Performance тестирование
4. Подготовка к production деплою

**Следующие шаги после Processor:**
- CI/CD pipeline
- Мониторинг и метрики
- Production deployment
