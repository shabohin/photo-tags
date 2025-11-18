package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockTelegramServer is a mock Telegram Bot API server
type MockTelegramServer struct {
	server          *httptest.Server
	mu              sync.RWMutex
	updates         []TelegramUpdate
	sentMessages    []SentMessage
	sentPhotos      []SentPhoto
	updateOffset    int
	botToken        string
	simulateError   bool
	simulateTimeout bool
	rateLimitCount  int
}

// TelegramUpdate represents a Telegram update
type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	MessageID int          `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      *TelegramChat `json:"chat,omitempty"`
	Text      string       `json:"text,omitempty"`
	Photo     []TelegramPhoto `json:"photo,omitempty"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// TelegramPhoto represents a photo size
type TelegramPhoto struct {
	FileID   string `json:"file_id"`
	FileSize int    `json:"file_size"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// SentMessage represents a sent message
type SentMessage struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

// SentPhoto represents a sent photo
type SentPhoto struct {
	ChatID  int64  `json:"chat_id"`
	Photo   string `json:"photo"`
	Caption string `json:"caption"`
}

// NewMockTelegramServer creates a new mock Telegram server
func NewMockTelegramServer(botToken string) *MockTelegramServer {
	mock := &MockTelegramServer{
		botToken: botToken,
		updates:  make([]TelegramUpdate, 0),
		sentMessages: make([]SentMessage, 0),
		sentPhotos: make([]SentPhoto, 0),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", mock.handleRequest)

	mock.server = httptest.NewServer(mux)
	return mock
}

// Close closes the mock server
func (m *MockTelegramServer) Close() {
	m.server.Close()
}

// URL returns the server URL
func (m *MockTelegramServer) URL() string {
	return m.server.URL
}

// AddUpdate adds a new update to the queue
func (m *MockTelegramServer) AddUpdate(update TelegramUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updates = append(m.updates, update)
}

// AddPhotoUpdate adds a photo update
func (m *MockTelegramServer) AddPhotoUpdate(chatID int64, fileID string, fileSize, width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	update := TelegramUpdate{
		UpdateID: len(m.updates) + 1,
		Message: &TelegramMessage{
			MessageID: len(m.updates) + 1,
			From: &TelegramUser{
				ID:        chatID,
				FirstName: "Test",
				Username:  "testuser",
			},
			Chat: &TelegramChat{
				ID:   chatID,
				Type: "private",
			},
			Photo: []TelegramPhoto{
				{
					FileID:   fileID,
					FileSize: fileSize,
					Width:    width,
					Height:   height,
				},
			},
		},
	}

	m.updates = append(m.updates, update)
}

// GetSentMessages returns all sent messages
func (m *MockTelegramServer) GetSentMessages() []SentMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sentMessages
}

// GetSentPhotos returns all sent photos
func (m *MockTelegramServer) GetSentPhotos() []SentPhoto {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sentPhotos
}

// SimulateError enables error simulation
func (m *MockTelegramServer) SimulateError(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateError = enable
}

// SimulateTimeout enables timeout simulation
func (m *MockTelegramServer) SimulateTimeout(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateTimeout = enable
}

// SimulateRateLimit simulates rate limiting
func (m *MockTelegramServer) SimulateRateLimit(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitCount = count
}

// handleRequest handles incoming requests
func (m *MockTelegramServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	simulateError := m.simulateError
	simulateTimeout := m.simulateTimeout
	rateLimitCount := m.rateLimitCount
	m.mu.RUnlock()

	// Simulate timeout
	if simulateTimeout {
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}

	// Simulate rate limit
	if rateLimitCount > 0 {
		m.mu.Lock()
		m.rateLimitCount--
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          false,
			"error_code":  429,
			"description": "Too Many Requests: retry after 1",
			"parameters": map[string]int{
				"retry_after": 1,
			},
		})
		return
	}

	// Simulate error
	if simulateError {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          false,
			"error_code":  400,
			"description": "Bad Request: simulated error",
		})
		return
	}

	// Route based on path
	switch {
	case r.URL.Path == fmt.Sprintf("/bot%s/getUpdates", m.botToken):
		m.handleGetUpdates(w, r)
	case r.URL.Path == fmt.Sprintf("/bot%s/sendMessage", m.botToken):
		m.handleSendMessage(w, r)
	case r.URL.Path == fmt.Sprintf("/bot%s/sendPhoto", m.botToken):
		m.handleSendPhoto(w, r)
	case r.URL.Path == fmt.Sprintf("/bot%s/getFile", m.botToken):
		m.handleGetFile(w, r)
	case r.URL.Path == fmt.Sprintf("/bot%s/getMe", m.botToken):
		m.handleGetMe(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleGetUpdates handles getUpdates requests
func (m *MockTelegramServer) handleGetUpdates(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var updates []TelegramUpdate
	for i := m.updateOffset; i < len(m.updates); i++ {
		updates = append(updates, m.updates[i])
	}

	if len(updates) > 0 {
		m.updateOffset = m.updates[len(m.updates)-1].UpdateID + 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"result": updates,
	})
}

// handleSendMessage handles sendMessage requests
func (m *MockTelegramServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var msg SentMessage
	json.Unmarshal(body, &msg)

	m.mu.Lock()
	m.sentMessages = append(m.sentMessages, msg)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"message_id": len(m.sentMessages),
			"chat": map[string]interface{}{
				"id": msg.ChatID,
			},
		},
	})
}

// handleSendPhoto handles sendPhoto requests
func (m *MockTelegramServer) handleSendPhoto(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	photo := SentPhoto{
		Caption: r.FormValue("caption"),
	}

	m.mu.Lock()
	m.sentPhotos = append(m.sentPhotos, photo)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"message_id": len(m.sentPhotos),
		},
	})
}

// handleGetFile handles getFile requests
func (m *MockTelegramServer) handleGetFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"file_id":   "test_file_id",
			"file_size": 1024,
			"file_path": "photos/test.jpg",
		},
	})
}

// handleGetMe handles getMe requests
func (m *MockTelegramServer) handleGetMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"id":         123456789,
			"is_bot":     true,
			"first_name": "Test Bot",
			"username":   "test_bot",
		},
	})
}
