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
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
)

// Bot represents a Telegram bot
type Bot struct {
	api      *tgbotapi.BotAPI
	logger   *logging.Logger
	minio    storage.MinIOInterface
	rabbitmq messaging.RabbitMQInterface
	cfg      *config.Config
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
func NewBot(cfg *config.Config, logger *logging.Logger, minio storage.MinIOInterface, rabbitmq messaging.RabbitMQInterface) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	return &Bot{
		api:      bot,
		logger:   logger,
		minio:    minio,
		rabbitmq: rabbitmq,
		cfg:      cfg,
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

	// Check if message contains photos or documents
	if update.Message.Photo != nil && len(update.Message.Photo) > 0 {
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
			botLog.Error("Failed to get file URL", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to get file URL")
			return
		}

		// Process photo
		if err := b.processMedia(ctx, botLog, update.Message, fileID, "photo.jpg", fileURL); err != nil {
			botLog.Error("Failed to process photo", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to process photo")
			return
		}

	} else if update.Message.Document != nil {
		// Handle document
		document := update.Message.Document
		fileName := document.FileName
		fileID := document.FileID

		// Check file extension
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			b.sendErrorMessage(update.Message.Chat.ID, "Only JPG and PNG files are supported")
			return
		}

		groupID := uuid.New().String()
		botLog := NewBotLogger(log.WithGroupID(groupID), groupID)
		botLog.Info("Received document", fileName)

		// Get file URL
		fileURL, err := b.api.GetFileDirectURL(fileID)
		if err != nil {
			botLog.Error("Failed to get file URL", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to get file URL")
			return
		}

		// Process document
		if err := b.processMedia(ctx, botLog, update.Message, fileID, fileName, fileURL); err != nil {
			botLog.Error("Failed to process document", err)
			b.sendErrorMessage(update.Message.Chat.ID, "Failed to process document")
			return
		}
	} else if update.Message.Text != "" {
		// Handle text message
		b.handleTextMessage(update.Message)
	}
}

// processMedia processes media files (photos and documents)
func (b *Bot) processMedia(ctx context.Context, log *BotLogger, message *tgbotapi.Message, fileID, fileName, fileURL string) error {
	// Download file
	resp, err := http.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Determine content type
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Generate trace ID
	traceID := uuid.New().String()
	log = NewBotLogger(log.Logger.WithTraceID(traceID), log.GetGroupID())

	// Upload file to MinIO
	minioObjectPath := fmt.Sprintf("%s/%s", traceID, fileName)
	if err := b.minio.UploadFile(ctx, storage.BucketOriginal, minioObjectPath, resp.Body, contentType); err != nil {
		return fmt.Errorf("failed to upload file to MinIO: %w", err)
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
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Info("Image uploaded and message published", uploadMessage)

	return nil
}

// handleProcessedImage handles a processed image
func (b *Bot) handleProcessedImage(data []byte) error {
	var message models.ImageProcessed
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	botLog := NewBotLogger(b.logger.WithTraceID(message.TraceID).WithGroupID(message.GroupID), message.GroupID)
	botLog.Info("Received processed image", message)

	// Check if processing failed
	if message.Status == "failed" {
		b.sendErrorMessage(message.TelegramID, fmt.Sprintf("Failed to process image: %s", message.Error))
		return nil
	}

	// Download file from MinIO
	ctx := context.Background()
	obj, err := b.minio.DownloadFile(ctx, storage.BucketProcessed, message.ProcessedPath)
	if err != nil {
		botLog.Error("Failed to download file from MinIO", err)
		b.sendErrorMessage(message.TelegramID, "Failed to download processed image")
		return err
	}
	defer obj.Close()

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

	botLog.Info("Processed image sent", nil)
	return nil
}

// handleTextMessage handles text messages
func (b *Bot) handleTextMessage(message *tgbotapi.Message) {
	// Handle commands
	if message.IsCommand() {
		switch message.Command() {
		case "start":
			b.sendMessage(message.Chat.ID, "Welcome to Photo Tags Bot! Send me an image, and I'll add AI-generated metadata to it.")
		case "help":
			b.sendMessage(message.Chat.ID, "This bot automatically adds titles, descriptions, and keywords to your images using AI.\n\nJust send me a JPG or PNG image, and I'll process it for you!")
		default:
			b.sendMessage(message.Chat.ID, "Unknown command. Try /help for available commands.")
		}
		return
	}

	// Handle regular text messages
	b.sendMessage(message.Chat.ID, "Please send me an image to process. Use /help for more information.")
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
