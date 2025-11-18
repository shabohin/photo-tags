# Анализ требований к ресурсам - Photo Tags Service

**Дата анализа**: 18 ноября 2025

---

## 📊 Текущая архитектура

### Компоненты в Production:
1. **Gateway Service** - HTTP + Telegram Bot (Go)
2. **Analyzer Service** - AI анализ через OpenRouter (Go)
3. **Processor Service** - ExifTool обработка (Go + ExifTool)
4. **RabbitMQ** - Message broker
5. **MinIO** - Object storage (2 buckets: original, processed)
6. **Monitoring** - Prometheus + Grafana (в разработке)

### Планируемые компоненты:
7. **PostgreSQL** - База данных для статистики
8. **File Watcher Service** - Мониторинг папок (Go)
9. **Dashboard Service** - Web UI (Go/Node.js)
10. **Redis** - Кеширование (опционально)

---

## 💻 Оценка ресурсов по нагрузке

### 🟢 Сценарий 1: Малая нагрузка (MVP / Personal Use)

**Пользователи**: 1-10 активных пользователей
**Нагрузка**: 50-100 изображений в день
**Пиковая нагрузка**: 10 изображений одновременно

#### Минимальные требования:

**CPU:**
- Gateway: 0.5 core (легкая нагрузка, в основном I/O)
- Analyzer: 1 core (ожидание API responses)
- Processor: 1 core (ExifTool + I/O)
- RabbitMQ: 0.5 core
- MinIO: 0.5 core
- PostgreSQL: 0.5 core
- Monitoring: 0.5 core
- **ИТОГО: 4.5 cores**

**RAM:**
- Gateway: 256 MB
- Analyzer: 512 MB (worker pool 3)
- Processor: 512 MB (worker pool 3 + temp files)
- RabbitMQ: 512 MB
- MinIO: 512 MB
- PostgreSQL: 512 MB
- Prometheus: 512 MB
- Grafana: 256 MB
- **ИТОГО: ~3.5 GB**

**Disk:**
- OS + Apps: 10 GB
- MinIO storage: 50 GB (хранение ~5000 изображений original + processed)
- PostgreSQL: 5 GB
- Logs: 5 GB
- **ИТОГО: 70 GB SSD**

**Network:**
- Входящий трафик: ~50 MB/день (изображения от пользователей)
- OpenRouter API: ~30 MB/день
- Исходящий трафик: ~50 MB/день (обработанные изображения)
- **ИТОГО: ~150 MB/день, пики до 10 Mbps**

#### Рекомендуемое железо:

**Вариант A: Raspberry Pi 4/5**
- ✅ Raspberry Pi 5 (8GB RAM)
- ✅ 4 CPU cores (ARM64)
- ✅ 256 GB microSD или USB SSD
- ✅ Gigabit Ethernet
- 💰 **Стоимость: ~$100-150**
- ⚡ **Энергопотребление: 5-10W**

**Вариант B: Mini PC**
- ✅ Intel N100 или аналог
- ✅ 8 GB RAM
- ✅ 256 GB NVMe SSD
- ✅ Gigabit LAN
- 💰 **Стоимость: ~$150-250**
- ⚡ **Энергопотребление: 10-15W**

**Вариант C: VPS Cloud**
- ✅ 2 vCPU
- ✅ 4 GB RAM
- ✅ 80 GB SSD
- 💰 **Стоимость: $10-20/месяц** (DigitalOcean, Hetzner, Linode)

---

### 🟡 Сценарий 2: Средняя нагрузка (Small Business)

**Пользователи**: 50-100 активных пользователей
**Нагрузка**: 500-1000 изображений в день
**Пиковая нагрузка**: 50 изображений одновременно

#### Рекомендуемые требования:

**CPU:**
- Gateway: 1 core
- Analyzer: 2 cores (worker pool 5-7)
- Processor: 2 cores (worker pool 5-7)
- RabbitMQ: 1 core
- MinIO: 1 core
- PostgreSQL: 1 core
- Monitoring: 1 core
- **ИТОГО: 9 cores (можно 8 с HT)**

**RAM:**
- Gateway: 512 MB
- Analyzer: 2 GB (больше workers)
- Processor: 2 GB (параллельная обработка)
- RabbitMQ: 1 GB
- MinIO: 2 GB
- PostgreSQL: 2 GB
- Prometheus: 1 GB
- Grafana: 512 MB
- Redis: 512 MB (опционально)
- **ИТОГО: ~11-12 GB**

**Disk:**
- OS + Apps: 20 GB
- MinIO storage: 500 GB (50,000 изображений)
- PostgreSQL: 20 GB
- Logs: 10 GB
- Backups: 100 GB
- **ИТОГО: 650 GB SSD**

**Network:**
- Трафик: ~1-2 GB/день
- Пики: 100 Mbps
- **Желательно: 1 Gbps**

#### Рекомендуемое железо:

**Вариант A: Desktop Server**
- ✅ Intel i5-12400 или AMD Ryzen 5 5600
- ✅ 16 GB DDR4 RAM
- ✅ 1 TB NVMe SSD
- ✅ Gigabit Ethernet
- 💰 **Стоимость: ~$500-700**
- ⚡ **Энергопотребление: 30-50W idle, 100W load**

**Вариант B: Cloud VPS**
- ✅ 4-6 vCPU
- ✅ 16 GB RAM
- ✅ 500 GB SSD
- 💰 **Стоимость: $40-80/месяц** (Hetzner CPX41, DO)

**Вариант C: Dedicated Server**
- ✅ Entry-level dedicated (Intel Xeon E-2xxx)
- ✅ 32 GB RAM
- ✅ 2x 1TB SSD (RAID1)
- 💰 **Стоимость: $50-100/месяц** (Hetzner, OVH)

---

### 🔴 Сценарий 3: Высокая нагрузка (Enterprise / SaaS)

**Пользователи**: 500-1000+ активных пользователей
**Нагрузка**: 5000-10000 изображений в день
**Пиковая нагрузка**: 200+ изображений одновременно

#### Требования (с горизонтальным масштабированием):

**CPU:**
- Gateway: 2 cores (2 replicas = 4 cores)
- Analyzer: 4 cores (3 replicas = 12 cores)
- Processor: 4 cores (3 replicas = 12 cores)
- RabbitMQ: 2 cores (cluster 3 nodes = 6 cores)
- MinIO: 4 cores (cluster 4 nodes = 16 cores)
- PostgreSQL: 4 cores (master + 2 replicas = 12 cores)
- Monitoring: 2 cores
- Load Balancer: 2 cores
- **ИТОГО: ~66 cores**

**RAM:**
- Gateway: 1 GB x2 = 2 GB
- Analyzer: 4 GB x3 = 12 GB
- Processor: 4 GB x3 = 12 GB
- RabbitMQ: 4 GB x3 = 12 GB
- MinIO: 8 GB x4 = 32 GB
- PostgreSQL: 8 GB x3 = 24 GB
- Prometheus: 4 GB
- Grafana: 2 GB
- Redis: 4 GB
- **ИТОГО: ~104 GB**

**Disk:**
- MinIO storage: 5-10 TB (500k+ изображений)
- PostgreSQL: 100-200 GB
- Logs + Monitoring: 100 GB
- Backups: 2-3 TB
- **ИТОГО: 7-13 TB**

**Network:**
- Трафик: 20-50 GB/день
- Пики: 1-10 Gbps
- **Требуется: 10 Gbps**

#### Рекомендуемое решение:

**Kubernetes Cluster:**
- ✅ 6-10 worker nodes (8 cores, 16-32 GB RAM each)
- ✅ Managed Kubernetes (GKE, EKS, AKS) или self-hosted
- ✅ Managed PostgreSQL
- ✅ S3-compatible storage (AWS S3, DigitalOcean Spaces)
- ✅ CDN для output images
- ✅ Load Balancer
- 💰 **Стоимость: $500-2000/месяц**

**Или Bare Metal:**
- ✅ 3-5 серверов (AMD EPYC 7xx2, 64-128 GB RAM each)
- ✅ Ceph или MinIO cluster для storage
- ✅ HA PostgreSQL cluster
- 💰 **Стоимость: $300-800/месяц** (Hetzner Dedicated)

---

## 📈 Масштабирование по компонентам

### Gateway Service
- **Stateless** - легко масштабируется горизонтально
- **Bottleneck**: Telegram API rate limits (30 req/sec)
- **Scaling strategy**: Nginx Load Balancer → N replicas

### Analyzer Service
- **CPU-bound** на ожидании OpenRouter API
- **Bottleneck**: OpenRouter rate limits и стоимость
- **Scaling strategy**: Увеличить worker pool + replicas

### Processor Service
- **CPU + I/O bound** (ExifTool + MinIO)
- **Bottleneck**: Disk I/O для temp files
- **Scaling strategy**: SSD + worker pool + replicas

### RabbitMQ
- **Memory-bound** при большой очереди
- **Bottleneck**: Single node throughput ~10k msg/sec
- **Scaling strategy**: RabbitMQ cluster (3+ nodes)

### MinIO
- **Storage + Network bound**
- **Bottleneck**: Disk I/O и network bandwidth
- **Scaling strategy**: Distributed cluster (4+ nodes, erasure coding)

### PostgreSQL
- **Memory + Disk I/O bound**
- **Bottleneck**: Write-heavy workload
- **Scaling strategy**: Read replicas + connection pooling (PgBouncer)

---

## 💰 Оценка стоимости

### Развертывание на собственном железе (one-time):

| Сценарий | Железо | Стоимость | Энергопотребление/месяц |
|----------|--------|-----------|-------------------------|
| **Малая нагрузка** | Raspberry Pi 5 | $150 | ~$2 (10W) |
| **Малая нагрузка** | Mini PC | $250 | ~$3 (15W) |
| **Средняя нагрузка** | Desktop Server | $700 | ~$15 (50W) |
| **Высокая нагрузка** | 3x Servers | $3000-6000 | ~$150 (500W) |

### Cloud Hosting (recurring):

| Сценарий | Provider | vCPU | RAM | Storage | Стоимость/месяц |
|----------|----------|------|-----|---------|-----------------|
| **Малая** | DigitalOcean | 2 | 4 GB | 80 GB | $24 |
| **Малая** | Hetzner Cloud | 2 | 4 GB | 40 GB | $7 |
| **Средняя** | Hetzner Cloud | 4 | 16 GB | 160 GB | $28 |
| **Средняя** | DigitalOcean | 6 | 16 GB | 320 GB | $84 |
| **Высокая** | Hetzner Dedicated | 8 cores | 32 GB | 2x1TB | $60 |
| **Высокая** | AWS/GCP/Azure | Custom | Custom | Custom | $500-2000 |

---

## 🎯 Рекомендации по выбору

### Для MVP / Personal Use:
✅ **Raspberry Pi 5 (8GB)** или **Hetzner Cloud CX21** ($7/мес)
- Достаточно для 10-50 пользователей
- Низкая стоимость
- Легко апгрейдить при росте

### Для Small Business:
✅ **Desktop Server (i5/Ryzen 5, 16GB)** или **Hetzner Cloud CPX31** ($28/мес)
- Обрабатывает 100-500 пользователей
- Хорошее соотношение цена/производительность
- Запас для роста

### Для Enterprise / SaaS:
✅ **Kubernetes на Hetzner Dedicated** или **Managed K8s (GKE/EKS)**
- Высокая доступность (HA)
- Горизонтальное масштабирование
- Auto-scaling по нагрузке

---

## 📊 Оценка по OpenRouter API costs

### Стоимость AI обработки:

**Предположения:**
- Средний размер изображения: 2 MB
- OpenRouter free models: $0 (но с rate limits)
- Fallback на GPT-4 Vision: ~$0.01 per image
- Claude 3.5 Sonnet: ~$0.003 per image

| Изображений/день | Free models (95%) | Paid models (5%) | Стоимость AI/месяц |
|------------------|-------------------|------------------|---------------------|
| 100 | 95 | 5 | ~$1.5 |
| 1000 | 950 | 50 | ~$15 |
| 10000 | 9500 | 500 | ~$150 |

**Оптимизация:**
- Используй free OpenRouter models максимально
- Кешируй результаты для похожих изображений (perceptual hash)
- Compress изображения перед отправкой в API
- Batch processing для снижения overhead

---

## ⚡ Оптимизация производительности

### Узкие места (Bottlenecks):

1. **OpenRouter API rate limits**
   - Решение: Multi-provider fallback, caching, batch requests

2. **ExifTool CPU usage**
   - Решение: Worker pool, async processing, SSD для temp files

3. **MinIO storage I/O**
   - Решение: SSD, distributed MinIO cluster, CDN для output

4. **RabbitMQ queue buildup**
   - Решение: Увеличить workers, monitoring для queue depth

5. **Network bandwidth**
   - Решение: Compression, CDN, региональные clusters

### Recommended optimizations:

1. **Use SSD everywhere** - критично для temp files и database
2. **Enable compression** - для MinIO и network transfer
3. **Implement caching** - Redis для metadata, image hashes
4. **Monitor queue depths** - RabbitMQ alerts на >1000 messages
5. **Auto-scaling** - HPA в Kubernetes на CPU/Memory metrics

---

## 🔧 Мониторинг ресурсов

### Key Metrics to track:

**Infrastructure:**
- CPU usage per service (target: <70%)
- Memory usage per service (target: <80%)
- Disk I/O (IOPS, latency)
- Network throughput (Mbps)
- Queue depth (RabbitMQ)

**Application:**
- Images processed/hour
- Average processing time (Gateway→User)
- Error rate per service
- API costs (OpenRouter/month)
- Storage usage (MinIO buckets)

**Business:**
- Active users
- Daily/Monthly active users (DAU/MAU)
- Conversion rate (free→paid)
- Churn rate

---

## 📝 Выводы

### Для текущего MVP (90% готовности):

**Минимально для запуска:**
- 4 CPU cores
- 4 GB RAM
- 100 GB SSD
- 10 Mbps network
- **Cost: $7-24/месяц (cloud) или $150 (Raspberry Pi one-time)**

**Рекомендуется:**
- 8 CPU cores
- 16 GB RAM
- 500 GB SSD
- 100 Mbps network
- **Cost: $28-84/месяц (cloud) или $500-700 (own hardware)**

### Scaling path:
1. **0-100 users**: Single server (RPi5 или VPS)
2. **100-500 users**: Dedicated server или managed VPS
3. **500-1000 users**: Начать кластеризацию (RabbitMQ, MinIO)
4. **1000+ users**: Kubernetes cluster, managed services

**Break-even analysis:**
- Own hardware окупается за ~6-12 месяцев vs cloud
- Cloud выгоднее для непредсказуемой нагрузки (auto-scaling)
- Dedicated servers - sweet spot для стабильной средней нагрузки

---

**Дата**: 18 ноября 2025
**Следующий review**: При достижении 100 активных пользователей
