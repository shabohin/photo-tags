# Photo Tags Dashboard Service

Web-based monitoring dashboard for the Photo Tags system.

## Features

- **Services Status Monitoring**: Real-time health checks for Gateway, Analyzer, and Processor services
- **RabbitMQ Queue Metrics**: Monitor message queues and consumer counts
- **System Metrics**: Track processed images and queue statistics
- **External Links**: Quick access to RabbitMQ Management UI and MinIO Console
- **Auto-refresh**: Dashboard automatically updates every 10 seconds

## Architecture

The dashboard is built with:
- **Backend**: Go HTTP server with REST API
- **Frontend**: Vanilla JavaScript with responsive HTML/CSS
- **Dependencies**:
  - `gorilla/mux` - HTTP router
  - `rabbitmq/amqp091-go` - RabbitMQ client

## API Endpoints

- `GET /api/health` - Dashboard health check
- `GET /api/metrics` - All metrics (services, queues, stats)
- `GET /api/services/status` - Service health statuses only
- `GET /api/rabbitmq/queues` - RabbitMQ queue information
- `GET /api/config` - Frontend configuration (external URLs)

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DASHBOARD_PORT` | `3000` | HTTP server port |
| `RABBITMQ_URL` | `amqp://user:password@rabbitmq:5672/` | RabbitMQ connection URL |
| `GATEWAY_URL` | `http://gateway:8080` | Gateway service URL |
| `ANALYZER_URL` | `http://analyzer:8081` | Analyzer service URL |
| `PROCESSOR_URL` | `http://processor:8082` | Processor service URL |
| `MINIO_URL` | `http://localhost:9001` | MinIO console URL |
| `RABBITMQ_MGMT_URL` | `http://localhost:15672` | RabbitMQ management URL |

## Running Locally

### With Docker Compose

```bash
cd docker
docker-compose up dashboard
```

### Standalone

```bash
cd services/dashboard

# Set environment variables
export RABBITMQ_URL=amqp://user:password@localhost:5672/
export GATEWAY_URL=http://localhost:8080

# Run
go run cmd/main.go
```

Visit http://localhost:3000 in your browser.

## Development

### Building

```bash
go build -o dashboard ./cmd/main.go
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
services/dashboard/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   └── handler.go       # HTTP handlers
│   ├── config/
│   │   └── config.go        # Configuration
│   └── metrics/
│       └── service.go       # Metrics collection
├── static/
│   ├── index.html           # Dashboard UI
│   ├── style.css            # Styles
│   └── app.js               # Frontend logic
├── Dockerfile
├── go.mod
└── README.md
```

## Monitoring

The dashboard provides insights into:

1. **Service Health**
   - Gateway status (up/down)
   - Analyzer status (up/down)
   - Processor status (up/down)
   - RabbitMQ connection status

2. **Queue Metrics**
   - `image_uploaded` - Images waiting for analysis
   - `metadata_generated` - Images with metadata waiting for processing
   - `image_processed` - Processed images ready for delivery
   - Consumer count per queue

3. **Statistics**
   - Total images in queues
   - Total processed images count

## Troubleshooting

### Dashboard shows services as "down"

- Ensure all services are running and accessible
- Check service URLs in environment variables
- Verify services have `/health` endpoints

### Queue metrics not loading

- Verify RabbitMQ connection URL is correct
- Check RabbitMQ is accessible from dashboard container
- Ensure queues exist (they're created by services on startup)

### External links not working

- RabbitMQ Management UI: Ensure port 15672 is exposed
- MinIO Console: Ensure port 9001 is exposed
- Check firewall rules and Docker port mappings
