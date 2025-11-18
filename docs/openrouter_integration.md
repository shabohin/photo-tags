# OpenRouter Integration Guide

## Overview

The Analyzer Service integrates with OpenRouter API to provide intelligent, cost-effective image analysis using automatically selected free vision models.

## Key Features

### 🤖 Automatic Model Selection
- Discovers all available models from OpenRouter API
- Filters for free models with vision/multimodal capabilities
- Selects model with highest context length
- Updates selection every 24 hours (configurable)

### 🔄 Rate Limit Handling
- Parses `X-RateLimit-Remaining` and `X-RateLimit-Reset` headers
- Automatic retry after reset time
- Exponential backoff for transient errors (2s, 4s, 8s)

### 🛡️ Error Recovery
- Retry logic for 5xx server errors (up to 3 attempts)
- Fallback to configured model if selection fails
- Thread-safe operations with `sync.RWMutex`

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────┐
│           Analyzer Service                      │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌────────────────┐      ┌──────────────────┐  │
│  │ Model Selector │─────▶│ OpenRouter Client│  │
│  │   (selector/)  │      │  (api/openrouter)│  │
│  └────────────────┘      └──────────────────┘  │
│         │                        │              │
│         │                        │              │
│         └────────┬───────────────┘              │
│                  │                              │
│                  ▼                              │
│         ┌────────────────┐                      │
│         │  OpenRouter    │                      │
│         │      API       │                      │
│         └────────────────┘                      │
│                                                 │
└─────────────────────────────────────────────────┘
```

### Model Selector Service

**File:** `services/analyzer/internal/selector/selector.go`

**Responsibilities:**
1. Fetch available models from OpenRouter
2. Filter and rank free vision models
3. Cache selected model in memory
4. Periodic updates (default: 24h)
5. Thread-safe concurrent access

**Key Methods:**
- `Start(ctx)`: Begin periodic model checking
- `GetCurrentModel()`: Retrieve cached model (thread-safe)
- `updateModels()`: Fetch and select best model
- `Stop()`: Graceful shutdown

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OPENROUTER_API_KEY` | OpenRouter API key | - | ✅ Yes |
| `OPENROUTER_MODEL` | Fallback model name | `openai/gpt-4o` | No |
| `OPENROUTER_MODEL_CHECK_INTERVAL` | Model update interval | `24h` | No |
| `OPENROUTER_MAX_TOKENS` | Max tokens in response | `500` | No |
| `OPENROUTER_TEMPERATURE` | Generation temperature | `0.7` | No |
| `OPENROUTER_PROMPT` | Analysis prompt | See below | No |

### Default Prompt

```
Generate title, description and keywords for this image.
Return strictly in JSON format with fields 'title', 'description' and 'keywords'.
```

### Example Configuration

```bash
# .env file
OPENROUTER_API_KEY=sk-or-v1-...
OPENROUTER_MODEL=openai/gpt-4o  # Fallback only
OPENROUTER_MODEL_CHECK_INTERVAL=24h
OPENROUTER_MAX_TOKENS=500
OPENROUTER_TEMPERATURE=0.7
```

---

## How It Works

### 1. Model Selection Flow

```
┌──────────────┐
│   Startup    │
└──────┬───────┘
       │
       ▼
┌──────────────────────────┐
│ Fetch Available Models   │
│ GET /api/v1/models       │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Filter Free Vision Models│
│ pricing.prompt == "0"    │
│ modality: multimodal     │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Sort by Context Length   │
│ Higher = Better          │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Cache Selected Model     │
│ Thread-Safe Storage      │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Wait 24h (configurable)  │
└──────┬───────────────────┘
       │
       └────────────────────▶ Repeat
```

### 2. Model Selection Criteria

**Filtering:**
```go
// A model is selected if:
1. pricing.prompt == "0" (free)
2. AND (
     modality contains "multimodal"
     OR modality contains "image"
     OR id contains "vision"
     OR name contains "vision"
   )
```

**Ranking:**
```go
// Models sorted by:
1. Context length (descending)
2. First model selected
```

**Example Free Models (as of Nov 2025):**
- `google/gemini-2.0-flash-exp:free` (32,768 context)
- `meta-llama/llama-3.2-11b-vision-instruct:free` (8,192 context)
- `google/gemini-flash-1.5:free` (1,000,000 context)

### 3. Image Analysis Flow

```
┌──────────────┐
│ Receive Image│
└──────┬───────┘
       │
       ▼
┌──────────────────────────┐
│ Get Current Model        │
│ selector.GetCurrentModel()│
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Encode Image to Base64   │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ POST /chat/completions   │
│ With image + prompt      │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Rate Limit Check         │
│ 429? → Retry after reset │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Parse JSON Response      │
│ Extract metadata         │
└──────┬───────────────────┘
       │
       ▼
┌──────────────────────────┐
│ Return Metadata          │
│ {title, description, ...}│
└──────────────────────────┘
```

---

## API Integration

### OpenRouter Endpoints

#### 1. Get Available Models

**Request:**
```http
GET https://openrouter.ai/api/v1/models
Authorization: Bearer {API_KEY}
HTTP-Referer: https://github.com/shabohin/photo-tags
X-Title: Photo Tags Service
```

**Response:**
```json
{
  "data": [
    {
      "id": "google/gemini-2.0-flash-exp:free",
      "name": "Gemini 2.0 Flash (free)",
      "pricing": {
        "prompt": "0",
        "completion": "0"
      },
      "context_length": 32768,
      "architecture": {
        "modality": "multimodal"
      }
    }
  ]
}
```

#### 2. Analyze Image

**Request:**
```http
POST https://openrouter.ai/api/v1/chat/completions
Authorization: Bearer {API_KEY}
HTTP-Referer: https://github.com/shabohin/photo-tags
X-Title: Photo Tags Service
Content-Type: application/json
```

```json
{
  "model": "google/gemini-2.0-flash-exp:free",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "Generate title, description and keywords for this image..."
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
          }
        }
      ]
    }
  ],
  "max_tokens": 500,
  "temperature": 0.7
}
```

**Response:**
```json
{
  "id": "gen-...",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "{\"title\": \"Sunset Over Mountains\", \"description\": \"...\", \"keywords\": [\"sunset\", \"mountain\"]}"
      }
    }
  ]
}
```

---

## Rate Limit Handling

### Rate Limit Headers

OpenRouter returns rate limit information in response headers:

```http
X-RateLimit-Remaining: 9
X-RateLimit-Reset: 1700000000
```

### Retry Strategy

```
Attempt 1: Immediate
  ↓ (failed)
Wait 2s
  ↓
Attempt 2: After 2s
  ↓ (failed)
Wait 4s
  ↓
Attempt 3: After 4s
  ↓ (failed)
Wait 8s
  ↓
Final Attempt: After 8s
  ↓ (failed)
Return Error
```

### Implementation

```go
// Exponential backoff
for attempt := 0; attempt < maxRetries; attempt++ {
    if attempt > 0 {
        delay := initialRetryDelay * (1 << uint(attempt-1))
        time.Sleep(delay)
    }

    resp, err := client.Do(req)

    // Rate limit handling
    if resp.StatusCode == 429 {
        resetTime := parseRateLimitReset(resp.Header)
        time.Sleep(time.Until(resetTime))
        continue
    }

    // Success
    if resp.StatusCode == 200 {
        break
    }
}
```

---

## Error Handling

### Error Types

1. **Rate Limit Exceeded (429)**
   - Parse reset time from headers
   - Wait until reset time
   - Retry request

2. **Server Errors (5xx)**
   - Retry with exponential backoff
   - Up to 3 attempts
   - Log error details

3. **Client Errors (4xx)**
   - No retry (permanent failure)
   - Log error and return

4. **Network Errors**
   - Retry with exponential backoff
   - Up to 3 attempts

### Fallback Behavior

```
Model Selection Failed
  ↓
Use OPENROUTER_MODEL
  ↓
Continue Processing
```

---

## Testing

### Unit Tests

**File:** `services/analyzer/internal/api/openrouter/client_test.go`

Tests include:
- Model selection logic
- Rate limit parsing
- Error handling
- Response parsing

**File:** `services/analyzer/internal/selector/selector_test.go`

Tests include:
- Periodic updates
- Thread safety
- Fallback behavior
- Graceful shutdown

### Running Tests

```bash
cd services/analyzer
go test ./internal/api/openrouter -v
go test ./internal/selector -v
```

---

## Monitoring and Logs

### Key Log Messages

**Model Selection:**
```
INFO  Starting Model Selector              check_interval=24h0m0s
INFO  Updating available models
INFO  Successfully fetched models          models_count=127
INFO  Selected best free vision model
      model_id="google/gemini-2.0-flash-exp:free"
      model_name="Gemini 2.0 Flash (free)"
      context_len=32768
```

**Rate Limiting:**
```
WARN  Rate limit exceeded for AnalyzeImage
      retry_after=30s
      reset_time=2025-11-18T19:00:00Z
```

**Errors:**
```
ERROR Failed to fetch available models    error="connection refused"
WARN  Using fallback model                 model="openai/gpt-4o"
```

### Metrics to Monitor

- Model selection success rate
- Rate limit hits per hour
- Average response time
- Error rate by type (5xx, 4xx, network)
- Currently selected model

---

## Best Practices

### 1. API Key Management
- Store API key securely in environment variables
- Never commit API keys to version control
- Rotate keys periodically

### 2. Rate Limiting
- Monitor rate limit headers
- Adjust `MODEL_CHECK_INTERVAL` based on usage
- Implement backoff strategies

### 3. Error Handling
- Always check for rate limits
- Implement graceful degradation
- Log all errors with context

### 4. Model Selection
- Review selected models periodically
- Monitor model performance
- Keep fallback model updated

---

## Troubleshooting

### Issue: No free models available

**Symptoms:**
```
WARN  No free vision models found
ERROR No free vision models available
```

**Solution:**
- Verify OpenRouter API is accessible
- Check if free models are still offered
- Use fallback model (`OPENROUTER_MODEL`)

### Issue: Rate limit exceeded frequently

**Symptoms:**
```
WARN  Rate limit exceeded
      retry_after=60s
```

**Solution:**
- Increase `WORKER_RETRY_DELAY`
- Reduce `WORKER_CONCURRENCY`
- Consider paid OpenRouter tier

### Issue: Model selection failing

**Symptoms:**
```
ERROR Failed to fetch available models
```

**Solution:**
- Check API key validity
- Verify network connectivity
- Review firewall rules
- Check OpenRouter API status

---

## Future Enhancements

- [ ] Model performance tracking
- [ ] A/B testing between models
- [ ] Custom model ranking algorithm
- [ ] Multi-provider support (beyond OpenRouter)
- [ ] Response caching for repeated images
- [ ] Batch processing support

---

## References

- [OpenRouter API Documentation](https://openrouter.ai/docs)
- [OpenRouter Models List](https://openrouter.ai/models)
- [Analyzer Service Documentation](./analyzer_service.md)
- [Architecture Documentation](./analyzer_architecture.md)

---

**Last Updated:** November 18, 2025
**Author:** Claude AI
**Version:** 1.0
