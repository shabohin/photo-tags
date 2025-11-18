# Image Processing Package

Пакет для автоматической оптимизации изображений перед отправкой в OpenRouter API.

## Возможности

- **Валидация изображений**: Проверка формата и корректности изображения
- **Автоматическое изменение размера**: Уменьшение разрешения до 2048px по большей стороне
- **Сжатие без потери качества**: JPEG с quality=85, PNG с максимальным сжатием
- **Конвертация PNG→JPEG**: Автоматическая конвертация больших PNG файлов (>500KB) в JPEG
- **Оптимизация размера файла**: Сжатие файлов размером >2MB

## Использование

```go
import (
    "github.com/shabohin/photo-tags/pkg/imageprocessing"
    "github.com/sirupsen/logrus"
)

// Создание оптимизатора
logger := logrus.New()
optimizer := imageprocessing.NewOptimizer(logger)

// Оптимизация изображения
imageData := []byte{...} // исходные данные изображения
result, err := optimizer.Optimize(imageData, "trace-id-123")
if err != nil {
    log.Fatal(err)
}

// Использование оптимизированных данных
optimizedData := result.Data

// Информация об оптимизации
fmt.Printf("Original size: %d KB\n", result.OriginalSize/1024)
fmt.Printf("Optimized size: %d KB\n", result.OptimizedSize/1024)
fmt.Printf("Compression ratio: %.2f\n", result.CompressionRatio)
fmt.Printf("Was resized: %v\n", result.WasResized)
fmt.Printf("Was converted: %v\n", result.WasConverted)
```

## Константы

- `MaxImageSize` = 2MB - максимальный размер файла без оптимизации
- `MaxImageDimension` = 2048px - максимальное разрешение по большей стороне
- `JPEGQuality` = 85 - качество JPEG сжатия
- `PNGToJPEGThreshold` = 500KB - порог конвертации PNG в JPEG

## Типы

### OptimizationResult

```go
type OptimizationResult struct {
    Data              []byte      // Оптимизированные данные изображения
    OriginalSize      int         // Исходный размер в байтах
    OptimizedSize     int         // Размер после оптимизации в байтах
    OriginalFormat    ImageFormat // Исходный формат (jpeg/png)
    OptimizedFormat   ImageFormat // Формат после оптимизации
    WasResized        bool        // Было ли изменено разрешение
    WasCompressed     bool        // Было ли сжато
    WasConverted      bool        // Была ли конвертация формата
    CompressionRatio  float64     // Коэффициент сжатия (OptimizedSize/OriginalSize)
}
```

## Функции

### NewOptimizer

```go
func NewOptimizer(logger *logrus.Logger) *Optimizer
```

Создает новый экземпляр оптимизатора изображений.

### Validate

```go
func (o *Optimizer) Validate(data []byte) error
```

Проверяет валидность данных изображения. Поддерживаются форматы: JPEG, PNG.

### Optimize

```go
func (o *Optimizer) Optimize(data []byte, traceID string) (*OptimizationResult, error)
```

Оптимизирует изображение:
1. Валидация формата
2. Изменение размера (если >2048px по любой стороне)
3. Конвертация PNG→JPEG (если PNG >500KB)
4. Сжатие изображения

Возвращает оригинальные данные, если изображение не требует оптимизации.

### OptimizeReader

```go
func (o *Optimizer) OptimizeReader(r io.Reader, traceID string) (*OptimizationResult, error)
```

Удобный метод для оптимизации изображения из io.Reader.

## Пример интеграции

Пакет интегрирован в сервис Analyzer для оптимизации изображений перед отправкой в OpenRouter:

```go
// В services/analyzer/internal/domain/service/analyzer.go
func (s *ImageAnalyzerService) AnalyzeImage(ctx context.Context, msg model.ImageUploadMessage) (model.Metadata, error) {
    // Скачать изображение
    imageBytes, err := s.minioClient.DownloadImage(ctx, msg.OriginalPath)
    if err != nil {
        return model.Metadata{}, err
    }

    // Оптимизировать изображение
    optimizationResult, err := s.imageOptimizer.Optimize(imageBytes, msg.TraceID)
    if err != nil {
        return model.Metadata{}, err
    }

    // Отправить оптимизированное изображение в OpenRouter
    metadata, err := s.openRouterClient.AnalyzeImage(ctx, optimizationResult.Data, msg.TraceID)
    return metadata, err
}
```

## Преимущества

1. **Снижение затрат на API**: Меньшие изображения = меньше данных для передачи
2. **Ускорение обработки**: Меньше времени на загрузку и обработку
3. **Оптимизация трафика**: Снижение использования пропускной способности
4. **Прозрачность**: Детальное логирование всех операций оптимизации

## Тестирование

Запуск тестов:

```bash
cd pkg/imageprocessing
go test -v
```

Покрытие тестами включает:
- Валидацию изображений
- Изменение размера
- Сжатие в JPEG и PNG
- Конвертацию PNG→JPEG
- Обработку маленьких изображений (без оптимизации)
- Обработку больших изображений (с оптимизацией)
