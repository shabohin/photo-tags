# Backup and Disaster Recovery

This guide covers backup and disaster recovery procedures for the Photo Tags system.

## Table of Contents

- [Overview](#overview)
- [What Gets Backed Up](#what-gets-backed-up)
- [Automated Backups](#automated-backups)
- [Manual Backup](#manual-backup)
- [Restore Procedures](#restore-procedures)
- [Configuration](#configuration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The Photo Tags system includes automated backup and restore scripts for all critical data:

- **MinIO Buckets**: Original and processed images
- **RabbitMQ**: Queue definitions, exchanges, and bindings
- **Backup Retention**: Last 7 days of backups (configurable)
- **Backup Format**: Compressed tar.gz archives with timestamps

## What Gets Backed Up

### MinIO Storage

- `original` bucket: Original uploaded images
- `processed` bucket: Processed images with metadata

### RabbitMQ

- Queue definitions and configuration
- Exchange definitions and bindings
- Virtual host settings
- User permissions (if configured)
- Queue metadata and statistics

## Automated Backups

### Setup Automated Backups

Use the setup script to configure daily automated backups:

```bash
# Install cron job for daily backups at 2:00 AM
./scripts/setup-backup-cron.sh

# Test backup script without installing cron
./scripts/setup-backup-cron.sh --test-only

# Remove automated backups
./scripts/setup-backup-cron.sh --remove
```

### Manual Cron Configuration

Alternatively, you can manually configure cron:

```bash
# Edit crontab
crontab -e

# Add this line for daily backups at 2:00 AM
0 2 * * * cd /path/to/photo-tags && ./scripts/backup.sh >> /var/log/photo-tags-backup.log 2>&1
```

### Verify Cron Setup

```bash
# List current cron jobs
crontab -l

# View backup logs
tail -f /var/log/photo-tags-backup.log
```

## Manual Backup

### Basic Usage

Create a backup manually at any time:

```bash
# Create backup in default location (./backups)
./scripts/backup.sh

# Create backup in custom location
BACKUP_DIR=/path/to/backups ./scripts/backup.sh
```

### Configuration Options

The backup script supports several environment variables:

```bash
# Backup directory (default: ./backups)
export BACKUP_DIR=/path/to/backups

# Retention period in days (default: 7)
export RETENTION_DAYS=14

# MinIO configuration (uses defaults from docker-compose.yml)
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin

# RabbitMQ configuration
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=15672
export RABBITMQ_USER=user
export RABBITMQ_PASSWORD=password

# Run backup
./scripts/backup.sh
```

### Backup Process

The backup script performs the following steps:

1. Verifies Docker containers are running
2. Creates timestamped backup directory
3. Backs up MinIO buckets using MinIO Client (mc)
4. Exports RabbitMQ definitions via Management API
5. Creates compressed tar.gz archive
6. Removes backups older than retention period

### Backup Output

Backups are stored as compressed archives:

```
backups/
├── backup_20240101_020000.tar.gz
├── backup_20240102_020000.tar.gz
├── backup_20240103_020000.tar.gz
└── ...
```

Each backup archive contains:

```
backup_20240101_020000/
├── backup.info              # Backup metadata
├── minio/
│   ├── original/           # Original images bucket
│   └── processed/          # Processed images bucket
└── rabbitmq/
    ├── definitions.json    # RabbitMQ definitions
    ├── queues.json         # Queue information
    └── overview.json       # System overview
```

## Restore Procedures

### Interactive Restore

The easiest way to restore is using interactive mode:

```bash
./scripts/restore.sh
```

This will:
1. Display available backups
2. Let you select which backup to restore
3. Show backup information
4. Ask for confirmation before restoring

### Restore Specific Backup

To restore a specific backup file:

```bash
# Restore from specific backup file
./scripts/restore.sh /path/to/backup_20240101_020000.tar.gz

# Force restore without confirmation
./scripts/restore.sh --force backup_20240101_020000.tar.gz
```

### Configuration Options

```bash
# Custom backup directory
BACKUP_DIR=/path/to/backups ./scripts/restore.sh

# MinIO configuration
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin

# RabbitMQ configuration
export RABBITMQ_HOST=localhost
export RABBITMQ_PORT=15672
export RABBITMQ_USER=user
export RABBITMQ_PASSWORD=password

# Run restore
./scripts/restore.sh
```

### Restore Process

The restore script performs the following steps:

1. Verifies Docker containers are running
2. Extracts backup archive to temporary directory
3. Displays backup information and asks for confirmation
4. Restores MinIO buckets using MinIO Client (mc)
5. Imports RabbitMQ definitions via Management API
6. Cleans up temporary files

### Partial Restore

To restore only specific components, you can manually extract the backup:

```bash
# Extract backup
tar -xzf backup_20240101_020000.tar.gz

# Restore only MinIO
cd backup_20240101_020000/minio
# Use mc mirror or cp to restore specific buckets

# Restore only RabbitMQ
curl -u user:password \
  -H "Content-Type: application/json" \
  -X POST \
  --data-binary @backup_20240101_020000/rabbitmq/definitions.json \
  http://localhost:15672/api/definitions
```

## Configuration

### Environment Variables

Create a `.env.backup` file for persistent configuration:

```bash
# Backup configuration
BACKUP_DIR=/data/backups
RETENTION_DAYS=7

# MinIO configuration
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# RabbitMQ configuration
RABBITMQ_HOST=localhost
RABBITMQ_PORT=15672
RABBITMQ_USER=user
RABBITMQ_PASSWORD=password
```

Load configuration before running scripts:

```bash
source .env.backup
./scripts/backup.sh
```

### Docker Compose Integration

The backup scripts automatically detect running containers from docker-compose.yml.

If using custom container names, update the scripts:

```bash
# In backup.sh and restore.sh
MINIO_CONTAINER="my-custom-minio"
RABBITMQ_CONTAINER="my-custom-rabbitmq"
```

## Best Practices

### Backup Strategy

1. **Regular Backups**: Schedule daily automated backups
2. **Off-site Storage**: Copy backups to remote storage periodically
3. **Verify Backups**: Periodically test restore procedures
4. **Monitor Logs**: Check backup logs for errors
5. **Retention Policy**: Keep enough backups for your recovery needs

### Storage Recommendations

```bash
# Example: Sync backups to remote storage
rsync -av ./backups/ user@backup-server:/backups/photo-tags/

# Or use cloud storage (S3, Google Cloud Storage, etc.)
aws s3 sync ./backups/ s3://my-backup-bucket/photo-tags/
```

### Security

1. **Encrypt Backups**: Consider encrypting sensitive data
2. **Secure Credentials**: Store MinIO and RabbitMQ credentials securely
3. **Access Control**: Limit access to backup files
4. **Network Security**: Use VPN or secure channels for remote backups

### Monitoring

Add monitoring to backup process:

```bash
# Check backup success
if ./scripts/backup.sh; then
  echo "Backup successful" | mail -s "Photo Tags Backup OK" admin@example.com
else
  echo "Backup failed" | mail -s "Photo Tags Backup FAILED" admin@example.com
fi
```

## Troubleshooting

### Backup Issues

**Problem**: Backup script fails with "Container not running"

```bash
# Check if containers are running
docker ps

# Start containers if needed
cd docker && docker-compose up -d
```

**Problem**: MinIO backup is empty

```bash
# Verify buckets exist and contain data
docker exec minio mc ls backup/original
docker exec minio mc ls backup/processed

# Check MinIO credentials
echo $MINIO_ACCESS_KEY
echo $MINIO_SECRET_KEY
```

**Problem**: RabbitMQ backup fails

```bash
# Verify RabbitMQ management API is accessible
curl -u user:password http://localhost:15672/api/overview

# Check RabbitMQ container logs
docker logs rabbitmq
```

**Problem**: Disk space issues

```bash
# Check available disk space
df -h

# Remove old backups manually
find ./backups -name "backup_*.tar.gz" -mtime +7 -delete

# Or adjust retention period
RETENTION_DAYS=3 ./scripts/backup.sh
```

### Restore Issues

**Problem**: Restore fails with permission errors

```bash
# Run restore script with appropriate permissions
sudo ./scripts/restore.sh

# Or fix file permissions
chmod -R 755 backups/
```

**Problem**: Data not appearing after restore

```bash
# Restart containers after restore
cd docker && docker-compose restart

# Verify data was restored
docker exec minio mc ls restore/original
docker exec rabbitmq rabbitmqctl list_queues
```

**Problem**: RabbitMQ restore conflicts with existing queues

```bash
# Option 1: Delete existing queues before restore
docker exec rabbitmq rabbitmqctl purge_queue queue_name

# Option 2: Stop services, restore, then start services
cd docker
docker-compose stop gateway analyzer processor
./scripts/restore.sh --force backup.tar.gz
docker-compose start gateway analyzer processor
```

### Verification

After backup or restore, verify data integrity:

```bash
# Check backup archive
tar -tzf backups/backup_20240101_020000.tar.gz

# Verify MinIO data
docker exec minio mc ls restore/original --recursive
docker exec minio mc ls restore/processed --recursive

# Verify RabbitMQ
curl -u user:password http://localhost:15672/api/queues
curl -u user:password http://localhost:15672/api/exchanges
```

## Disaster Recovery Scenarios

### Scenario 1: Complete System Failure

```bash
# 1. Reinstall Docker and docker-compose
# 2. Clone repository
git clone https://github.com/shabohin/photo-tags.git
cd photo-tags

# 3. Start containers
cd docker
docker-compose up -d

# 4. Restore from backup
cd ..
./scripts/restore.sh

# 5. Verify services
docker ps
curl http://localhost:8080/health
```

### Scenario 2: Data Corruption

```bash
# 1. Stop affected services
cd docker
docker-compose stop

# 2. Restore from last known good backup
cd ..
./scripts/restore.sh

# 3. Start services
cd docker
docker-compose start

# 4. Verify data integrity
```

### Scenario 3: Accidental Data Deletion

```bash
# Restore specific bucket or queue
./scripts/restore.sh

# Or manually restore specific components
tar -xzf backup.tar.gz
# Use mc or curl to restore specific items
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [MinIO Client Guide](https://min.io/docs/minio/linux/reference/minio-mc.html)
- [RabbitMQ Management API](https://www.rabbitmq.com/management.html)
- [Cron Documentation](https://man7.org/linux/man-pages/man5/crontab.5.html)

## Support

For issues or questions about backup and recovery:

1. Check logs: `/var/log/photo-tags-backup.log`
2. Review Docker logs: `docker logs <container_name>`
3. Open an issue on GitHub
4. Contact system administrator
