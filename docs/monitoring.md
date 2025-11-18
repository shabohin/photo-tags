# Monitoring with Datadog

This document describes how to set up and use Datadog monitoring for the Photo Tags Service.

## Overview

The Photo Tags Service integrates with Datadog to provide comprehensive monitoring, including:

- **APM (Application Performance Monitoring)**: Distributed tracing across services
- **Custom Metrics**: Business and technical metrics via DogStatsD
- **Log Management**: Centralized logging from all Docker containers
- **Infrastructure Monitoring**: Docker and system-level metrics

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Datadog Platform                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │   APM    │ │ Metrics  │ │   Logs   │ │  Infra   │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
└───────────────────────▲──────────────────────────────────────┘
                        │
                        │ API Key
                        │
┌───────────────────────▼──────────────────────────────────────┐
│                    Datadog Agent                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                     │
│  │  Tracer  │ │ StatsD   │ │   Logs   │                     │
│  │  :8126   │ │  :8125   │ │Collector │                     │
│  └──────────┘ └──────────┘ └──────────┘                     │
└───────┬───────────────┬────────────────┬─────────────────────┘
        │               │                │
    Traces          Metrics            Logs
        │               │                │
┌───────┴───────────────┴────────────────┴─────────────────────┐
│                   Application Services                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                     │
│  │ Gateway  │ │ Analyzer │ │Processor │                     │
│  └──────────┘ └──────────┘ └──────────┘                     │
└───────────────────────────────────────────────────────────────┘
```

## Setup

### 1. Get Datadog API Key

1. Sign up for a free Datadog account at [datadoghq.com](https://www.datadoghq.com/)
2. Navigate to **Organization Settings** → **API Keys**
3. Create a new API key or copy an existing one

### 2. Configure Environment Variables

Edit `docker/.env`:

```bash
# Datadog monitoring
DD_API_KEY=your_api_key_here
DD_SITE=datadoghq.com  # or datadoghq.eu for EU region
DD_ENV=development  # or production, staging, etc.
```

**Note**: If `DD_API_KEY` is not set, the services will run without Datadog monitoring.

### 3. Start Services

```bash
./scripts/start.sh
```

The Datadog Agent will automatically start and begin collecting data.

### 4. Verify Setup

1. **Check Agent Status**:
   ```bash
   docker logs datadog-agent
   ```

2. **Check Service Logs**:
   ```bash
   docker logs gateway | grep -i datadog
   docker logs analyzer | grep -i datadog
   ```

3. **Visit Datadog Dashboard**:
   - Go to [app.datadoghq.com](https://app.datadoghq.com/)
   - Navigate to **APM** → **Services** to see your services
   - Navigate to **Metrics** → **Explorer** to see custom metrics

## Available Metrics

### Gateway Service

#### Telegram Bot Metrics

- `photo_tags.telegram.messages.received` (count)
  - Tags: `type:photo|document|text`
  - Total messages received from Telegram

- `photo_tags.telegram.messages.processed` (count)
  - Tags: `type:photo|document|text`
  - Successfully processed messages

- `photo_tags.telegram.messages.errors` (count)
  - Tags: `type:photo|document|text`, `error:get_file_url|process_media|unsupported_format`
  - Failed message processing

#### Image Processing Metrics

- `photo_tags.image.uploaded` (count)
  - Successfully uploaded images to MinIO

- `photo_tags.image.upload.duration` (timing, ms)
  - Time taken to upload image to MinIO

- `photo_tags.image.size.bytes` (histogram)
  - Size distribution of uploaded images

- `photo_tags.image.upload.errors` (count)
  - Tags: `error:minio_upload`
  - Failed image uploads

#### RabbitMQ Metrics

- `photo_tags.rabbitmq.messages.published` (count)
  - Tags: `queue:image_upload|image_processed`
  - Messages published to RabbitMQ

- `photo_tags.rabbitmq.messages.consumed` (count)
  - Tags: `queue:image_upload|image_processed`
  - Messages consumed from RabbitMQ

- `photo_tags.rabbitmq.messages.publish.errors` (count)
  - Tags: `queue:...`, `error:publish_failed`
  - Failed message publishes

### Analyzer Service

(Analyzer-specific metrics will be added when implemented in the service)

## Dashboards

### Creating a Dashboard

1. Go to **Dashboards** → **New Dashboard**
2. Add widgets for key metrics:
   - **Timeseries**: Message throughput (`photo_tags.telegram.messages.received`)
   - **Query Value**: Error rate (`photo_tags.telegram.messages.errors`)
   - **Heatmap**: Image upload duration (`photo_tags.image.upload.duration`)
   - **Top List**: Most common error types

### Recommended Widgets

#### Message Processing Overview

```
sum:photo_tags.telegram.messages.received{*} by {type}.as_count()
sum:photo_tags.telegram.messages.processed{*} by {type}.as_count()
sum:photo_tags.telegram.messages.errors{*} by {error}.as_count()
```

#### Image Processing Performance

```
avg:photo_tags.image.upload.duration{*}
p95:photo_tags.image.upload.duration{*}
p99:photo_tags.image.upload.duration{*}
```

#### Queue Health

```
sum:photo_tags.rabbitmq.messages.published{*} by {queue}.as_rate()
sum:photo_tags.rabbitmq.messages.consumed{*} by {queue}.as_rate()
```

## Alerts

### Example Alerts

#### High Error Rate

```
Alert when: sum:photo_tags.telegram.messages.errors{*}.as_rate() > 0.1
Timeframe: last 5 minutes
Message: "High error rate in Gateway service"
```

#### Slow Image Upload

```
Alert when: p95:photo_tags.image.upload.duration{*} > 5000
Timeframe: last 10 minutes
Message: "Image upload taking longer than 5 seconds"
```

#### Queue Backlog

```
Alert when: sum:photo_tags.rabbitmq.messages.published{queue:image_upload}.as_rate() -
            sum:photo_tags.rabbitmq.messages.consumed{queue:image_upload}.as_rate() > 10
Timeframe: last 5 minutes
Message: "Queue backlog growing"
```

## APM (Traces)

Datadog APM automatically traces:
- HTTP requests
- RabbitMQ operations
- External API calls (OpenRouter)
- Database queries

### View Traces

1. Go to **APM** → **Traces**
2. Filter by service: `gateway`, `analyzer`, or `processor`
3. Click on a trace to see the full request flow

### Service Map

The service map shows:
- Service dependencies
- Request flow
- Latency between services
- Error rates

Access it at **APM** → **Service Map**

## Logs

All Docker container logs are automatically collected by the Datadog Agent.

### View Logs

1. Go to **Logs** → **Explorer**
2. Filter by:
   - Service: `service:gateway`, `service:analyzer`
   - Status: `status:error`
   - Custom fields: `trace_id`, `group_id`

### Log Correlation

Logs are automatically correlated with traces using `trace_id`. Click on a log entry to see the related trace.

## Troubleshooting

### Agent Not Connecting

**Problem**: Datadog Agent shows connection errors

**Solution**:
1. Verify API key is correct
2. Check network connectivity:
   ```bash
   docker exec datadog-agent agent status
   ```
3. Ensure DD_SITE is correct for your region

### No Metrics Appearing

**Problem**: Custom metrics not showing in Datadog

**Solution**:
1. Verify DD_API_KEY is set in `.env`
2. Check service logs for monitoring initialization:
   ```bash
   docker logs gateway | grep "Datadog monitoring"
   ```
3. Wait a few minutes for metrics to appear (first push can take time)

### Traces Not Appearing

**Problem**: No APM traces in Datadog

**Solution**:
1. Verify DD_APM_ENABLED=true in docker-compose.yml
2. Check Agent is receiving traces:
   ```bash
   docker logs datadog-agent | grep -i trace
   ```
3. Ensure services can reach Agent on port 8126

## Cost Optimization

### Free Tier Limits

Datadog free tier includes:
- 5 hosts
- 1-day metric retention
- 15-day trace retention
- 3-day log retention

### Reducing Costs

1. **Sample Traces**: Reduce trace ingestion
   ```go
   tracer.Start(
       tracer.WithSampler(tracer.NewRateSampler(0.5)), // 50% sampling
   )
   ```

2. **Filter Logs**: Only collect error logs
   ```yaml
   DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL: false
   ```

3. **Reduce Metric Cardinality**: Use fewer tags

## Best Practices

1. **Use Consistent Tags**:
   - Always include `env`, `service`, `version`
   - Use standard tag names across services

2. **Instrument Critical Paths**:
   - Focus on user-facing operations
   - Track error rates and latencies

3. **Set Up Alerts Early**:
   - Monitor error rates
   - Track performance degradation
   - Alert on queue backlogs

4. **Regular Dashboard Reviews**:
   - Weekly review of key metrics
   - Monthly capacity planning
   - Quarterly alert tuning

## Additional Resources

- [Datadog Documentation](https://docs.datadoghq.com/)
- [Go Tracer Documentation](https://docs.datadoghq.com/tracing/setup_overview/setup/go/)
- [DogStatsD Documentation](https://docs.datadoghq.com/developers/dogstatsd/)
- [Docker Integration](https://docs.datadoghq.com/agent/docker/)
