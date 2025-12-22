# System Counters Table Migration

## Overview

This migration (`008_create_counters_table.sql`) creates a dedicated `system_counters` table for efficient admin statistics. Counters are automatically maintained by database triggers, providing instant, 100% accurate counts without scanning large tables.

## What Changed

### Before
- Counters were calculated using `COUNT(*)` queries (slow on large tables)
- Or used approximate counts from `pg_class.reltuples` (~99% accurate)
- Performance degraded as tables grew

### After
- Dedicated `system_counters` table with a single row (singleton pattern)
- Automatic counter updates via database triggers
- Instant reads regardless of table size
- 100% accurate counters maintained in real-time

## Architecture

### Counter Table Structure

```sql
system_counters (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- Singleton pattern
    users_count INTEGER DEFAULT 0 NOT NULL,
    videos_count INTEGER DEFAULT 0 NOT NULL,
    comments_count INTEGER DEFAULT 0 NOT NULL,
    replies_count INTEGER DEFAULT 0 NOT NULL,
    upvotes_count INTEGER DEFAULT 0 NOT NULL,
    downvotes_count INTEGER DEFAULT 0 NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
)
```

### Trigger System

Database triggers automatically update counters on INSERT/DELETE operations:

- **users table**: Updates `users_count`
- **videos table**: Updates `videos_count`
- **comments table**: Updates `comments_count`
- **replies table**: Updates `replies_count`
- **upvotes table**: Updates `upvotes_count`
- **downvotes table**: Updates `downvotes_count`

### Trigger Functions

Each table has a dedicated trigger function that:
1. Detects INSERT or DELETE operations
2. Increments/decrements the appropriate counter
3. Updates the `updated_at` timestamp
4. Executes atomically within the same transaction

## Benefits

### 1. Performance
- **Instant reads**: Single row SELECT, always fast
- **No table scans**: Counters stored, not calculated
- **Scalable**: Performance doesn't degrade as tables grow

### 2. Accuracy
- **100% accurate**: Counters updated atomically with data changes
- **Real-time**: Counters reflect current state
- **Transaction-safe**: Updates happen within same transaction

### 3. Reliability
- **Automatic**: No manual counter updates needed
- **Consistent**: Triggers ensure counters stay in sync
- **Resync capability**: Manual resync endpoint available if needed

## Usage

### Get Counters (Admin API)

```http
GET /admin/counters
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
  "status": "success",
  "counters": {
    "users": 1250,
    "videos": 5432,
    "comments": 12345,
    "replies": 8765,
    "upvotes": 45678,
    "downvotes": 1234,
    "updated_at": "2024-01-20T14:22:00Z"
  }
}
```

### Resync Counters (Admin API)

If counters get out of sync (e.g., due to direct database operations):

```http
POST /admin/counters/resync
Authorization: Bearer <jwt_token>
```

This recalculates all counters from actual table counts.

## Migration Details

### Initial Sync

The migration includes an initial sync that:
1. Creates the counter table
2. Initializes counters to zero
3. Syncs counters with actual table counts
4. Sets up triggers for future updates

### Idempotency

The migration is idempotent:
- Uses `CREATE TABLE IF NOT EXISTS`
- Uses `CREATE OR REPLACE FUNCTION` for trigger functions
- Drops and recreates triggers to avoid duplicates
- Initial sync only runs if table is empty

## Maintenance

### Automatic Updates

Counters are automatically maintained by triggers. No manual intervention needed.

### Manual Resync

If counters get out of sync:
1. Use the `/admin/counters/resync` endpoint
2. Or run the sync SQL manually:
   ```sql
   UPDATE system_counters SET
       users_count = (SELECT COUNT(*) FROM users),
       videos_count = (SELECT COUNT(*) FROM videos),
       comments_count = (SELECT COUNT(*) FROM comments),
       replies_count = (SELECT COUNT(*) FROM replies),
       upvotes_count = (SELECT COUNT(*) FROM upvotes),
       downvotes_count = (SELECT COUNT(*) FROM downvotes),
       updated_at = CURRENT_TIMESTAMP
   WHERE id = 1;
   ```

### Monitoring

Check counter accuracy periodically:
```sql
SELECT 
    (SELECT COUNT(*) FROM users) as actual_users,
    users_count as counter_users,
    (SELECT COUNT(*) FROM videos) as actual_videos,
    videos_count as counter_videos
FROM system_counters WHERE id = 1;
```

## Performance Comparison

| Approach | Read Speed | Accuracy | Scalability |
|----------|-----------|----------|------------|
| COUNT(*) queries | Slow (varies by table size) | 100% | Degrades with size |
| pg_class.reltuples | Instant | ~99% | Good |
| **Counter table** | **Instant** | **100%** | **Excellent** |

## Notes

- Counters are updated atomically within the same transaction as data changes
- CASCADE deletes automatically trigger counter decrements
- Triggers fire AFTER INSERT/DELETE to ensure data consistency
- The singleton pattern (id=1) ensures only one row exists
- `updated_at` timestamp tracks when counters were last updated

## Future Enhancements

Potential improvements:
1. Add counters for other entities (followers, views, etc.)
2. Add per-user or per-video counters if needed
3. Add counter history/audit trail
4. Add counter drift detection and auto-resync

