# Deployment Guide

This document provides instructions for deploying the Photo Tags Service to various environments, from local development to production.

## Documentation Links

-   [Main README](../README.md)
-   [Architecture Documentation](architecture.md)
-   [Development Guide](development.md)
-   [Testing Strategy](testing.md)

## Deployment Options

The Photo Tags Service can be deployed in several ways:

1. **Local Development** - Using Docker Compose for testing and development
2. **Raspberry Pi** - Low-cost deployment option for personal use
3. **Cloud Provider** - For production-grade deployments

## System Requirements

### Minimum Requirements (Raspberry Pi)

-   Raspberry Pi 4 with at least 4GB RAM
-   32GB+ microSD card
-   Docker and Docker Compose installed
-   Stable internet connection

### Recommended Production Requirements

-   2+ CPU cores
-   4GB+ RAM
-   20GB+ storage space
-   Docker and Docker Compose
-   Monitoring system (Datadog, Prometheus, etc.)

## Local Deployment

### Prerequisites

-   Docker and Docker Compose installed
-   Git
-   Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
-   OpenRouter API Key with GPT-4o Vision access

### Deployment Steps

1. Clone the repository:

    ```bash
    git clone https://github.com/shabohin/photo-tags.git
    cd photo-tags
    ```

2. Make scripts executable:

    ```bash
    chmod +x scripts/*.sh
    ```

3. Run the setup script which will create environmental files and check dependencies:

    ```bash
    ./scripts/setup.sh
    ```

4. Edit the environment variables in `docker/.env`:

    ```bash
    nano docker/.env
    ```

5. Start the services:

    ```bash
    ./scripts/start.sh
    ```

6. Verify all services are running:

    ```bash
    docker ps
    ```

7. Test the Telegram bot by sending it an image.

## Raspberry Pi Deployment

### Hardware Setup

1. Install Raspberry Pi OS (64-bit recommended)
2. Update system packages:

    ```bash
    sudo apt update && sudo apt upgrade -y
    ```

3. Install Docker and Docker Compose:

    ```bash
    curl -sSL https://get.docker.com | sh
    sudo apt install -y libffi-dev python3-dev python3-pip
    sudo pip3 install docker-compose
    ```

4. Add your user to the docker group:

    ```bash
    sudo usermod -aG docker $(whoami)
    ```

5. Reboot the Raspberry Pi:
    ```bash
    sudo reboot
    ```

### Application Deployment

1. Clone and set up the application as described in Local Deployment section.

2. Configure the environment for Raspberry Pi:

    - Adjust memory limits in Docker Compose file
    - Set appropriate scaling parameters

3. Start the services:

    ```bash
    ./scripts/start.sh
    ```

4. Set up auto-restart using systemd:

    Create a file `/etc/systemd/system/phototags.service`:

    ```ini
    [Unit]
    Description=Photo Tags Service
    After=docker.service
    Requires=docker.service

    [Service]
    Type=oneshot
    RemainAfterExit=yes
    WorkingDirectory=/home/pi/photo-tags
    ExecStart=/home/pi/photo-tags/scripts/start.sh
    ExecStop=/home/pi/photo-tags/scripts/stop.sh
    User=pi

    [Install]
    WantedBy=multi-user.target
    ```

5. Enable and start the service:
    ```bash
    sudo systemctl enable phototags
    sudo systemctl start phototags
    ```

## Production Deployment

For production environments, consider these additional steps:

### Security Considerations

1. **Use Secret Management**:

    - Store API keys and tokens securely
    - Use Docker secrets or environment variables via a secure vault
    - Never commit secrets to source control

2. **Configure Proper Access Controls**:

    - Set up proper authentication for RabbitMQ
    - Configure MinIO with strict access policies
    - Use strong passwords for all admin interfaces

3. **Network Security**:
    - Use internal networks for service communication
    - Expose only necessary ports
    - Use TLS for all exposed endpoints

### Scalability

1. **Horizontal Scaling**:

    - Run multiple instances of each service
    - Configure RabbitMQ for high availability
    - Set up load balancing

2. **Vertical Scaling**:

    - Allocate appropriate CPU and memory resources
    - Monitor resource usage and adjust as needed

3. **Storage Considerations**:
    - Use persistent volumes for MinIO
    - Consider S3 for production deployments
    - Implement regular backups

### High Availability

1. **Redundancy**:

    - Deploy multiple instances across availability zones
    - Configure automatic failover

2. **Backups**:
    - Regularly backup RabbitMQ definitions
    - Backup MinIO data
    - Create system snapshots

## Monitoring and Maintenance

### Basic Monitoring

1. Access RabbitMQ Management UI:

    ```
    http://[your-host]:15672
    ```

    Default credentials: user / password

2. Access MinIO Console:

    ```
    http://[your-host]:9001
    ```

    Default credentials: minioadmin / minioadmin

3. Check service logs:
    ```bash
    docker logs gateway -f
    docker logs analyzer -f
    docker logs processor -f
    ```

### Advanced Monitoring with Datadog

1. Install Datadog Agent:

    ```bash
    DD_API_KEY=your_api_key bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/datadog-agent/master/cmd/agent/install_script.sh)"
    ```

2. Configure Docker integration:

    ```yaml
    init_config:
    instances:
        - url: 'unix://var/run/docker.sock'
          new_tag_names: true
    ```

3. Set up RabbitMQ monitoring:

    ```yaml
    init_config:
    instances:
        - rabbitmq_api_url: http://localhost:15672/api/
          rabbitmq_user: user
          rabbitmq_pass: password
    ```

4. Configure service-specific metrics and logs collection.

### Common Maintenance Tasks

1. **Cleaning Up Old Images**:

    ```bash
    # Remove images older than 30 days
    docker system prune -a
    ```

2. **Database Backup** (if using MongoDB in future):

    ```bash
    mongodump --out /backup/$(date +"%Y-%m-%d")
    ```

3. **Updating Services**:

    ```bash
    # Pull latest code
    git pull

    # Rebuild and restart
    ./scripts/stop.sh
    ./scripts/start.sh
    ```

4. **Rotating Logs**:
   Configure Docker log rotation:
    ```json
    {
        "log-driver": "json-file",
        "log-opts": {
            "max-size": "10m",
            "max-file": "3"
        }
    }
    ```

## Troubleshooting

### Common Issues

1. **Service Won't Start**

    - Check Docker logs for errors
    - Verify environment variables
    - Ensure ports aren't in use by other services

2. **RabbitMQ Connection Issues**

    - Verify RabbitMQ is running
    - Check credentials in .env
    - Inspect RabbitMQ logs

3. **MinIO Access Problems**

    - Ensure MinIO is running
    - Verify bucket permissions
    - Check file access rights

4. **Telegram Bot Not Responding**

    - Verify bot token
    - Check Gateway service logs
    - Ensure internet connectivity

5. **Image Processing Failures**
    - Check OpenRouter API key validity
    - Look for ExifTool errors in Processor logs
    - Verify image formats are supported

### Collecting Diagnostic Information

```bash
# Create diagnostic package
mkdir -p diagnostics
docker ps > diagnostics/containers.txt
docker-compose -f docker/docker-compose.yml logs > diagnostics/compose-logs.txt
docker inspect gateway > diagnostics/gateway-inspect.txt
docker inspect analyzer > diagnostics/analyzer-inspect.txt
docker inspect processor > diagnostics/processor-inspect.txt
docker inspect rabbitmq > diagnostics/rabbitmq-inspect.txt
docker inspect minio > diagnostics/minio-inspect.txt
tar -czf diagnostics.tar.gz diagnostics/
```

## Updating and Upgrading

### Updating the Application

1. Pull the latest changes:

    ```bash
    git pull origin main
    ```

2. Rebuild the services:
    ```bash
    ./scripts/stop.sh
    ./scripts/start.sh
    ```

### Major Version Upgrades

1. Back up all data:

    ```bash
    # Back up MinIO data
    docker exec -it minio sh -c "mc mirror /data/original /backup/original"
    docker exec -it minio sh -c "mc mirror /data/processed /backup/processed"

    # Back up RabbitMQ definitions
    curl -u user:password http://localhost:15672/api/definitions > rabbitmq-backup.json
    ```

2. Follow upgrade instructions specific to the release
