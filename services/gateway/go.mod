module github.com/shabohin/photo-tags/services/gateway

go 1.24.0

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/google/uuid v1.6.0
	github.com/shabohin/photo-tags/pkg v0.0.0
)

replace github.com/shabohin/photo-tags/pkg => ../../pkg