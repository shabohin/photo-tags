#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Ensure script is called from project root
ROOT_DIR=$(dirname "$(dirname "$0")")
cd "$ROOT_DIR/docker"

echo -e "${YELLOW}Starting monitoring stack...${NC}"

# Create network if it doesn't exist
docker network create photo-tags-network 2>/dev/null || true

# Start monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Monitoring stack started successfully!${NC}"
    echo -e "- Jaeger UI: ${YELLOW}http://localhost:9101${NC} - Distributed tracing"
    echo -e "- Prometheus: ${YELLOW}http://localhost:9109${NC} - Metrics collection"  
    echo -e "- Grafana: ${YELLOW}http://localhost:9110${NC} - Dashboards (admin/admin)"
    echo -e "- OTEL Collector: ${YELLOW}http://localhost:9108/metrics${NC} - OpenTelemetry metrics"
    
    echo ""
    echo -e "${GREEN}To view logs:${NC}"
    echo -e "  docker logs jaeger -f         ${YELLOW}# Jaeger logs${NC}"
    echo -e "  docker logs prometheus -f     ${YELLOW}# Prometheus logs${NC}"
    echo -e "  docker logs grafana -f        ${YELLOW}# Grafana logs${NC}"
    echo -e "  docker logs otel-collector -f ${YELLOW}# OTEL Collector logs${NC}"
    
    echo ""
    echo -e "${GREEN}To stop monitoring:${NC}"
    echo -e "  ./scripts/stop-monitoring.sh"
else
    echo -e "${RED}Failed to start monitoring stack.${NC}"
    exit 1
fi
