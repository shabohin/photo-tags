package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"github.com/shabohin/photo-tags/pkg/database"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
	"github.com/shabohin/photo-tags/services/gateway/internal/monitoring"
)

// Bot represents a Telegram bot
type Bot struct {
	api      *tgbotapi.BotAPI
	logger   *logging.Logger
	minio    storage.MinIOInterface
	rabbitmq messaging.RabbitMQInterface
	repo     database.RepositoryInterface
	cfg      *config.Config
	metrics  *monitoring.Metrics
}

// BotLogger extends the Logger with group ID
type BotLogger struct {
	*logging.Logger
	groupID string
}

// GetGroupID returns the group ID of the logger
func (l *BotLogger) GetGroupID() string {
	return l.groupID
}

// NewBotLogger creates a new bot logger
func NewBotLogger(logger *logging.Logger, groupID string) *BotLogger {
	return &BotLogger{
		Logger:  logger,
		groupID: groupID,
	}
}

// NewBot creates a new Telegram bot
func NewBot(
	cfg *config.Config,
	logger *logging.Logger,
	minio storage.MinIOInterface,
	rabbitmq messaging.RabbitMQInterface,
	repo database.RepositoryInterface,
) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Bot{
		api:      bot,
		logger:   logger,
		minio:    minio,
		rabbitmq: rabbitmq,
		repo:     repo,
		cfg:      cfg,
		metrics:  monitoring.NewMetrics(),
	}, nil
}

// Start starts listening for updates
func (b *Bot) Start(ctx context.Context) error {
	// Ensure MinIO buckets exist
	if err := b.minio.EnsureBucketExists(ctx, storage.BucketOriginal); err != nil {
		return fmt.Errorf("failed to create original bucket: %w", err)
	}
	if err := b.minio.EnsureBucketExists(ctx, storage.BucketProcessed); err != nil {
		return fmt.Errorf("failed to create processed bucket: %w", err)
	}

	// Ensure RabbitMQ queues exist
	if _, err := b.rabbitmq.DeclareQueue(messaging.QueueImageUpload); err != nil {
		return fmt.Errorf("failed to declare image upload queue: %w", err)
	}
	if _, err := b.rabbitmq.DeclareQueue(messaging.QueueImageProcessed); err != nil {
		return fmt.Errorf("failed to declare image processed queue: %w", err)
	}

	// Start consuming processed images
	if err := b.rabbitmq.ConsumeMessages(messaging.QueueImageProcessed, b.handleProcessedImage); err != nil {
		return fmt.Errorf("failed to start consuming processed images: %w", err)
	}

	// Configure updates
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// Start polling for updates
	updates := b.api.GetUpdatesChan(updateConfig)
	b.logger.Info("Telegram bot started", nil)

	// Handle updates
	for update := range updates {
		if update.Message == nil {
			continue
		}

		go b.handleUpdate(ctx, update)
	}

	return nil
}

// handleUpdate handles a single update
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	traceID := uuid.New().String()
	log := b.logger.WithTraceID(traceID)

	// Handle callback queries (for inline buttons)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
		return
	}

	// Check if no message present
	if update.Message == nil {
		return
	}

	// Check if message contains photos or documents
	if len(update.Message.Photo) > 0 {
		// Record metric
		b.metrics.Incr("telegram.messages.received", []string{"type:photo"})

		// Handle photo
		groupID := uuid.New().String()
		botLog := NewBotLogger(log.WithGroupID(groupID), groupID)
		botLog.Info("Received photo", update.Message.From.UserName)

		// Get the largest photo size
		photoSize := update.Message.Photo[len(update.Message.Photo)-1]
		fileID := photoSize.FileID

		// Get file URL
		fileURL, err := b.api.GetFileDirectURL(fileID)
		if err != nil {
			b.metrics.Incr("telegram.messages.errors", []string{"type:photo", "error:get_file_url"})
			botLog.Error("Failed to get file URL", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to get file URL")
			return
		}

		// Process photo
		if err := b.processMedia(ctx, botLog, update.Message, fileID, "photo.jpg", fileURL); err != nil {
			b.metrics.Incr("telegram.messages.errors", []string{"type:photo", "error:process_media"})
			botLog.Error("Failed to process photo", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to process photo")
			return
		}

		b.metrics.Incr("telegram.messages.processed", []string{"type:photo"})
	} else if update.Message.Document != nil {
		// Record metric
		b.metrics.Incr("telegram.messages.received", []string{"type:document"})

		// Handle document
		document := update.Message.Document
		fileName := document.FileName
		fileID := document.FileID

		// Check file extension
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			b.metrics.Incr("telegram.messages.errors", []string{"type:document", "error:unsupported_format"})
			b.sendErrorMessage(update.Message.Chat.ID, "Only JPG and PNG files are supported")
			return
		}

		groupID := uuid.New().String()
		botLog := NewBotLogger(log.WithGroupID(groupID), groupID)
		botLog.Info("Received document", fileName)

		// Get file URL
		fileURL, err := b.api.GetFileDirectURL(fileID)
		if err != nil {
			b.metrics.Incr("telegram.messages.errors", []string{"type:document", "error:get_file_url"})
			botLog.Error("Failed to get file URL", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to get file URL")
			return
		}

		// Process document
		if err := b.processMedia(ctx, botLog, update.Message, fileID, fileName, fileURL); err != nil {
			b.metrics.Incr("telegram.messages.errors", []string{"type:document", "error:process_media"})
			botLog.Error("Failed to process document", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to process document")
			return
		}

		b.metrics.Incr("telegram.messages.processed", []string{"type:document"})
	} else if update.Message.Text != "" {
		// Record metric
		b.metrics.Incr("telegram.messages.received", []string{"type:text"})

		// Handle text message
		b.handleTextMessage(update.Message)

		b.metrics.Incr("telegram.messages.processed", []string{"type:text"})
	}
}

// processMedia processes media files (photos and documents)
func (b *Bot) processMedia(
	ctx context.Context,
	log *BotLogger,
	message *tgbotapi.Message,
	_ string,
	fileName string,
	fileURL string,
) error {
	// Download file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error("Failed to close response body", closeErr)
		}
	}()

	// Determine content type
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Generate trace ID
	traceID := uuid.New().String()
	log = NewBotLogger(log.WithTraceID(traceID), log.GetGroupID())

	// Upload file to MinIO
	minioObjectPath := fmt.Sprintf("%s/%s", traceID, fileName)
	uploadStart := time.Now()
	if err := b.minio.UploadFile(ctx, storage.BucketOriginal, minioObjectPath, resp.Body, contentType); err != nil {
		b.metrics.Incr("image.upload.errors", []string{"error:minio_upload"})
		return fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	// Record metrics
	uploadDuration := time.Since(uploadStart).Milliseconds()
	b.metrics.Timing("image.upload.duration", uploadDuration, []string{})
	b.metrics.Incr("image.uploaded", []string{})
	if resp.ContentLength > 0 {
		b.metrics.Histogram("image.size.bytes", float64(resp.ContentLength), []string{})
	}

	// Send acknowledgement message
	b.sendMessage(message.Chat.ID, "✅ Image received! Processing...")

	// Create upload message
	uploadMessage := models.ImageUpload{
		TraceID:          traceID,
		GroupID:          log.GetGroupID(),
		TelegramID:       message.From.ID,
		TelegramUsername: message.From.UserName,
		OriginalFilename: fileName,
		OriginalPath:     minioObjectPath,
		Timestamp:        time.Now(),
	}

	// Publish upload message
	if err := b.rabbitmq.PublishMessage(messaging.QueueImageUpload, uploadMessage); err != nil {
		b.metrics.Incr("rabbitmq.messages.publish.errors", []string{"queue:image_upload", "error:publish_failed"})
		return fmt.Errorf("failed to publish message: %w", err)
	}

	b.metrics.Incr("rabbitmq.messages.published", []string{"queue:image_upload"})

	// Log image to database if repository is available
	if b.repo != nil {
		username := message.From.UserName
		originalPath := minioObjectPath
		img := &database.Image{
			TraceID:          traceID,
			TelegramID:       message.From.ID,
			TelegramUsername: &username,
			Filename:         fileName,
			OriginalPath:     &originalPath,
			Status:           database.StatusPending,
		}

		if err := b.repo.CreateImage(ctx, img); err != nil {
			log.Error("Failed to log image to database", err)
			// Don't fail the upload if database logging fails
		}
	}

	log.Info("Image uploaded and message published", uploadMessage)

	return nil
}

// handleProcessedImage handles a processed image
func (b *Bot) handleProcessedImage(data []byte) error {
	// Record consumed message
	b.metrics.Incr("rabbitmq.messages.consumed", []string{"queue:image_processed"})

	var message models.ImageProcessed
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	botLog := NewBotLogger(b.logger.WithTraceID(message.TraceID).WithGroupID(message.GroupID), message.GroupID)
	botLog.Info("Received processed image", message)

	ctx := context.Background()

	// Check if processing failed
	if message.Status == "failed" {
		b.sendErrorMessage(message.TelegramID, "Failed to process image: "+message.Error)

		// Update database status if repository is available
		if b.repo != nil {
			errorMsg := message.Error
			if err := b.repo.UpdateImageStatus(ctx, message.TraceID, database.StatusFailed, &errorMsg); err != nil {
				botLog.Error("Failed to update image status in database", err)
			}

			// Log error to errors table
			errRecord := &database.Error{
				TraceID:      &message.TraceID,
				Service:      "processor",
				ErrorType:    "processing_error",
				ErrorMessage: message.Error,
				TelegramID:   &message.TelegramID,
				Filename:     &message.OriginalFilename,
			}
			if logErr := b.repo.LogError(ctx, errRecord); logErr != nil {
				botLog.Error("Failed to log error to database", logErr)
			}
		}

		return nil
	}

	// Download file from MinIO
	obj, err := b.minio.DownloadFile(ctx, storage.BucketProcessed, message.ProcessedPath)
	if err != nil {
		botLog.Error("Failed to download file from MinIO", err)
		b.sendErrorMessage(message.TelegramID, "Failed to download processed image")
		return err
	}
	defer func() {
		if closeErr := obj.Close(); closeErr != nil {
			botLog.Error("Failed to close MinIO object", closeErr)
		}
	}()

	// Read file contents
	fileBytes, err := io.ReadAll(obj)
	if err != nil {
		botLog.Error("Failed to read file contents", err)
		b.sendErrorMessage(message.TelegramID, "Failed to read processed image")
		return err
	}

	// Send image
	fileBytesObj := tgbotapi.FileBytes{
		Name:  message.OriginalFilename,
		Bytes: fileBytes,
	}
	msg := tgbotapi.NewDocument(message.TelegramID, fileBytesObj)
	msg.Caption = "✅ Image processed with AI-generated metadata"

	if _, err := b.api.Send(msg); err != nil {
		botLog.Error("Failed to send image", err)
		b.sendErrorMessage(message.TelegramID, "Failed to send processed image")
		return err
	}

	// Update database status if repository is available
	if b.repo != nil {
		// Convert metadata from message to database format
		var metadata *database.ImageMetadata
		if message.OriginalFilename != "" {
			// Note: We don't have the metadata in ImageProcessed message
			// This could be enhanced in the future
			metadata = nil
		}

		processedPath := message.ProcessedPath
		if err := b.repo.UpdateImageProcessed(ctx, message.TraceID, processedPath, metadata, database.StatusSuccess); err != nil {
			botLog.Error("Failed to update processed image in database", err)
			// Don't fail if database update fails
		}
	}

	botLog.Info("Processed image sent", nil)
	return nil
}

// handleTextMessage handles text messages
func (b *Bot) handleTextMessage(message *tgbotapi.Message) {
	// Handle commands
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			b.handleStartCommand(message)
		case "help":
			b.handleHelpCommand(message)
		case "status":
			b.handleStatusCommand(message)
		default:
			b.sendMessage(message.Chat.ID, "❓ Unknown command. Try /help for available commands.")
		}
		return
	}

	// Handle regular text messages
	b.sendMessage(message.Chat.ID, "Please send me an image to process. Use /help for more information.")
}

// handleStartCommand handles the /start command
func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	welcomeText := "👋 *Welcome to Photo Tags Bot!*\n\n" +
		"I can automatically add AI-generated metadata to your images:\n" +
		"• 📝 Titles\n" +
		"• 📄 Descriptions\n" +
		"• 🏷️ Keywords\n\n" +
		"Just send me a JPG or PNG image, and I'll process it for you!\n\n" +
		"Use /help to see all available commands."

	// Create inline keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📖 Help", "help"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Status", "status"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send start message", err)
	}
}

// handleHelpCommand handles the /help command
func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	helpText := "🤖 *Photo Tags Bot - Help*\n\n" +
		"*Available Commands:*\n" +
		"/start - Welcome message and quick actions\n" +
		"/help - Show this help message\n" +
		"/status - Check processing queue status\n\n" +
		"*How to Use:*\n" +
		"1. Send me a JPG or PNG image (as photo or document)\n" +
		"2. Wait for processing (usually takes a few seconds)\n" +
		"3. Receive your image with AI-generated metadata\n\n" +
		"*Supported Formats:*\n" +
		"• JPG/JPEG\n" +
		"• PNG\n\n" +
		"*Features:*\n" +
		"✅ Automatic title generation\n" +
		"✅ Detailed descriptions\n" +
		"✅ Relevant keywords\n" +
		"✅ EXIF metadata preservation"

	// Create inline keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Check Status", "status"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send help message", err)
	}
}

// handleStatusCommand handles the /status command
func (b *Bot) handleStatusCommand(message *tgbotapi.Message) {
	statusText := b.getQueueStatus()

	// Create inline keyboard
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "status"),
			tgbotapi.NewInlineKeyboardButtonData("📖 Help", "help"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, statusText)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send status message", err)
	}
}

// handleCallbackQuery handles callback queries from inline buttons
func (b *Bot) handleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	// Answer the callback query to remove the loading state
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.api.Request(callback); err != nil {
		b.logger.Error("Failed to answer callback query", err)
	}

	// Handle different callback data
	switch query.Data {
	case "help":
		helpText := "🤖 *Photo Tags Bot - Help*\n\n" +
			"*Available Commands:*\n" +
			"/start - Welcome message and quick actions\n" +
			"/help - Show this help message\n" +
			"/status - Check processing queue status\n\n" +
			"*How to Use:*\n" +
			"1. Send me a JPG or PNG image (as photo or document)\n" +
			"2. Wait for processing (usually takes a few seconds)\n" +
			"3. Receive your image with AI-generated metadata\n\n" +
			"*Supported Formats:*\n" +
			"• JPG/JPEG\n" +
			"• PNG\n\n" +
			"*Features:*\n" +
			"✅ Automatic title generation\n" +
			"✅ Detailed descriptions\n" +
			"✅ Relevant keywords\n" +
			"✅ EXIF metadata preservation"

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Check Status", "status"),
			),
		)

		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, helpText)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			b.logger.Error("Failed to edit message", err)
		}

	case "status":
		statusText := b.getQueueStatus()

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "status"),
				tgbotapi.NewInlineKeyboardButtonData("📖 Help", "help"),
			),
		)

		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, statusText)
		edit.ParseMode = "Markdown"
		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			b.logger.Error("Failed to edit message", err)
		}

	default:
		b.logger.Error("Unknown callback data", fmt.Errorf("data: %s", query.Data))
	}
}

// getQueueStatus returns the current queue status
func (b *Bot) getQueueStatus() string {
	statusText := "📊 *Queue Status*\n\n"
	statusText += "✅ *System Status:* Operational\n\n"
	statusText += "Processing queues are active and ready to handle your images.\n\n"
	statusText += "_Note: Queue statistics require RabbitMQ Management API integration_"
	statusText += fmt.Sprintf("\n\n🕐 *Last Updated:* %s", time.Now().Format("15:04:05"))

	return statusText
}

// sendMessage sends a text message
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		b.logger.Error("Failed to send message", err)
	}
}

// sendErrorMessage sends an error message
func (b *Bot) sendErrorMessage(chatID int64, text string) {
	b.sendMessage(chatID, "❌ "+text)
}

// GetUsername returns the username of the bot
func (b *Bot) GetUsername() string {
	return b.api.Self.UserName
}
