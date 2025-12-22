#!/bin/bash
# Script to fix Elasticsearch disk watermark issues for single-node setup
# Run this after Elasticsearch container starts

echo "Waiting for Elasticsearch to be ready..."
sleep 10

# Wait for Elasticsearch to be available
until curl -s http://localhost:9200/_cluster/health > /dev/null; do
  echo "Waiting for Elasticsearch..."
  sleep 2
done

echo "Elasticsearch is ready. Configuring disk watermark settings..."

# Disable disk-based allocation for single-node setup
# This prevents shard relocation failures when disk usage is high
curl -X PUT "localhost:9200/_cluster/settings" \
  -H 'Content-Type: application/json' \
  -d '{
    "persistent": {
      "cluster.routing.allocation.disk.threshold_enabled": false
    }
  }'

echo ""
echo "Disk watermark check disabled. Elasticsearch will not relocate shards based on disk usage."
echo "This is safe for single-node setups where shards have nowhere to relocate."

