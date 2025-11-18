// Dashboard Application
class Dashboard {
    constructor() {
        this.refreshInterval = null;
        this.config = null;
        this.init();
    }

    async init() {
        // Load config
        await this.loadConfig();

        // Initial data load
        await this.refreshData();

        // Set up auto-refresh (every 10 seconds)
        this.refreshInterval = setInterval(() => {
            this.refreshData();
        }, 10000);

        // Set up manual refresh button
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.refreshData();
        });
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/config');
            if (!response.ok) {
                throw new Error('Failed to load config');
            }
            this.config = await response.json();
            this.updateExternalLinks();
        } catch (error) {
            console.error('Error loading config:', error);
        }
    }

    updateExternalLinks() {
        if (!this.config) return;

        const linksGrid = document.querySelector('.links-grid');
        linksGrid.innerHTML = `
            <a href="${this.config.rabbitmq_url}" target="_blank" class="link-card">
                <div class="link-icon">🐰</div>
                <div class="link-name">RabbitMQ</div>
                <div class="link-url">${new URL(this.config.rabbitmq_url).host}</div>
            </a>
            <a href="${this.config.minio_url}" target="_blank" class="link-card">
                <div class="link-icon">📦</div>
                <div class="link-name">MinIO</div>
                <div class="link-url">${new URL(this.config.minio_url).host}</div>
            </a>
        `;
    }

    async refreshData() {
        try {
            // Update timestamp
            this.updateTimestamp();

            // Fetch all metrics
            const response = await fetch('/api/metrics');
            if (!response.ok) {
                throw new Error('Failed to fetch metrics');
            }

            const data = await response.json();

            // Update UI
            this.updateServicesStatus(data.services);
            this.updateQueuesStatus(data.queues);
            this.updateMetrics(data.stats);

        } catch (error) {
            console.error('Error refreshing data:', error);
            this.showError('Failed to refresh data. Retrying...');
        }
    }

    updateTimestamp() {
        const now = new Date();
        const timeString = now.toLocaleTimeString('en-US', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
        document.getElementById('lastUpdate').textContent = timeString;
    }

    updateServicesStatus(services) {
        const container = document.getElementById('servicesStatus');

        if (!services || services.length === 0) {
            container.innerHTML = '<div class="loading-text">No services data available</div>';
            return;
        }

        container.innerHTML = services.map(service => `
            <div class="service-card ${service.healthy ? 'healthy' : 'unhealthy'}">
                <div class="service-name">${service.name}</div>
                <div class="service-status">
                    <span class="status-indicator ${service.status}"></span>
                    <span class="status-text">${service.status}</span>
                </div>
                ${service.checked_at ? `<div class="service-time" style="font-size: 0.75rem; color: #9ca3af; margin-top: 5px;">
                    Checked: ${new Date(service.checked_at).toLocaleTimeString()}
                </div>` : ''}
            </div>
        `).join('');
    }

    updateQueuesStatus(queues) {
        const container = document.getElementById('queuesStatus');

        if (!queues || queues.length === 0) {
            container.innerHTML = '<div class="loading-text">No queue data available</div>';
            return;
        }

        container.innerHTML = queues.map(queue => `
            <div class="queue-item">
                <div class="queue-name">${queue.name}</div>
                <div class="queue-stats">
                    <div class="queue-stat">
                        <span class="queue-stat-label">Messages</span>
                        <span class="queue-stat-value ${queue.messages > 100 ? 'high' : ''}">${queue.messages}</span>
                    </div>
                    <div class="queue-stat">
                        <span class="queue-stat-label">Consumers</span>
                        <span class="queue-stat-value">${queue.consumers}</span>
                    </div>
                </div>
            </div>
        `).join('');
    }

    updateMetrics(stats) {
        if (!stats) {
            return;
        }

        document.getElementById('queuedImages').textContent = stats.queued_images || 0;
        document.getElementById('totalProcessed').textContent = stats.total_processed || 0;
    }

    showError(message) {
        // Check if error message already exists
        let errorDiv = document.querySelector('.error-message');

        if (!errorDiv) {
            errorDiv = document.createElement('div');
            errorDiv.className = 'error-message';
            const main = document.querySelector('main');
            main.insertBefore(errorDiv, main.firstChild);
        }

        errorDiv.textContent = message;

        // Remove error message after 5 seconds
        setTimeout(() => {
            if (errorDiv.parentNode) {
                errorDiv.remove();
            }
        }, 5000);
    }

    destroy() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}

// Initialize dashboard when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new Dashboard();
});

// Clean up on page unload
window.addEventListener('beforeunload', () => {
    if (window.dashboard) {
        window.dashboard.destroy();
    }
});
