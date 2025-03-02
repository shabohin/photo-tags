package constants

// Supported file formats
var SupportedFormats = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

// MIME types for supported formats
var MimeTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

// Maximum file size in bytes (10 MB)
const MaxFileSize = 10 * 1024 * 1024

// Maximum number of concurrent uploads
const MaxConcurrentUploads = 5

// Status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)
