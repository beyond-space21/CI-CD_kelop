# Elasticsearch Docker Setup

## Issue: High Disk Watermark

If you're seeing errors like:
- `high disk watermark [90%] exceeded`
- `NoShardAvailableActionException`
- `all shards failed`

This happens because Elasticsearch tries to relocate shards when disk usage exceeds 90%, but in a single-node setup, there's nowhere to relocate them to.

## Quick Fix

### Option 1: Disable Disk-Based Allocation (Recommended for Single-Node)

After Elasticsearch starts, run:

```bash
curl -X PUT "localhost:9200/_cluster/settings" \
  -H 'Content-Type: application/json' \
  -d '{
    "persistent": {
      "cluster.routing.allocation.disk.threshold_enabled": false
    }
  }'
```

Or use the provided script:
```bash
./fix-disk-watermark.sh
```

### Option 2: Adjust Disk Watermarks

If you want to keep disk-based allocation but with higher thresholds:

```bash
curl -X PUT "localhost:9200/_cluster/settings" \
  -H 'Content-Type: application/json' \
  -d '{
    "persistent": {
      "cluster.routing.allocation.disk.watermark.low": "92%",
      "cluster.routing.allocation.disk.watermark.high": "95%",
      "cluster.routing.allocation.disk.watermark.flood_stage": "98%"
    }
  }'
```

## Starting Elasticsearch

```bash
cd ES
docker-compose up -d
```

Wait for Elasticsearch to be ready (check logs):
```bash
docker-compose logs -f elasticsearch
```

Then apply the fix using Option 1 above.

## Verifying the Fix

Check cluster settings:
```bash
curl http://localhost:9200/_cluster/settings?include_defaults=true | grep -A 5 "disk.threshold"
```

Check cluster health:
```bash
curl http://localhost:9200/_cluster/health?pretty
```

## Notes

- For production multi-node setups, keep disk-based allocation enabled
- For single-node development setups, disabling it is safe and prevents shard relocation failures
- The setting persists across restarts (it's stored as `persistent`)

