# PostgreSQL Migration Guide

This document describes the migration from MongoDB to PostgreSQL for the Hifi backend.

## Migration Files

All migration files are located in `DB/migrations/`:

1. **001_create_users_table.sql** - Creates users table
2. **002_create_deleted_users_table.sql** - Creates deleted_users table
3. **003_create_videos_table.sql** - Creates videos and video_on_upload tables
4. **004_create_social_tables.sql** - Creates followers, blocklists, upvotes, downvotes, comments, replies, and views tables
5. **005_create_servers_tables.sql** - Creates servers, rooms, sessions, streams, and stream_sessions tables
6. **006_add_password_to_users.sql** - Adds password column to users table
7. **007_add_foreign_key_constraints.sql** - Adds foreign key constraints with CASCADE deletes
8. **008_create_counters_table.sql** - Creates system_counters table with automatic trigger-based counter maintenance
9. **009_add_bio_email_to_users.sql** - Adds bio and email fields to users and deleted_users tables
10. **010_add_views_counter.sql** - Adds views_count to system_counters table with automatic trigger-based maintenance
11. **011_add_videos_created_at_index.sql** - Adds index on videos.created_at for efficient ordering queries
12. **012_add_composite_indexes.sql** - Adds composite indexes for optimized query patterns (ordering, filtering)

## Running Migrations

To run migrations, set the `RUN_MIGRATIONS=true` environment variable before starting the server:

```bash
export RUN_MIGRATIONS=true
go run main.go
```

Or add it to your `.env` file:
```
RUN_MIGRATIONS=true
```

## Database Connection

The PostgreSQL connection is configured in `Services/Mdb/Mdb.go` using the following environment variables:

- `POSTGRES_HOST` (default: localhost)
- `POSTGRES_PORT` (default: 5432)
- `POSTGRES_USER` (default: hiffi)
- `POSTGRES_PASSWORD` (default: dataofhiffiofsuperlabs)
- `POSTGRES_DB` (default: hiffi)

These defaults match the docker-compose.yml configuration.

## Migration Status

### ✅ Completed

1. **Database Service** (`Services/Mdb/Mdb.go`)
   - Replaced MongoDB client with PostgreSQL connection
   - Added migration runner function

2. **Schema Files**
   - Updated all schema files to use SQL `db` tags instead of `bson` tags
   - Added proper PostgreSQL types (StringArray for text arrays)

3. **Users Package** (`Events/Users/`)
   - All CRUD operations migrated to PostgreSQL
   - Functions: CreateUser, GetUser, GetSelf, DeleteUser, UpdateUser, UsernameAvailability, ListUser

4. **Servers Package** (`Events/Servers/`)
   - All CRUD operations migrated to PostgreSQL
   - Functions: CreateServer, ListServer, UpdateServer, DeleteServer, GetServer
   - Room operations: CreateRoom, ListRoom, UpdateRoom, DeleteRoom, HoldVacantRoom

5. **Main Application** (`main.go`)
   - Updated to use PostgreSQL initialization
   - Added migration runner support

### ⏳ Remaining

The following packages still need to be migrated from MongoDB to PostgreSQL:

1. **Videos Package** (`Events/Videos/`)
   - Functions: Upload, UploadACK, Delete, GetVideo, ListVideo, ListVideoSelf, VectorSearch
   - Collections: videos, video_on_upload

2. **Social Package** (`Events/Social/`)
   - SocialUsers: Follow, Unfollow, ListFollowers, ListFollowing
   - SocialVideos: Upvote, Downvote, Comment, Reply, View
   - Collections: followers, upvotes, downvotes, comments, replies, views

3. **Streaming Package** (`Events/Streaming/`)
   - May need updates if it uses database operations
   - Collections: streams, stream_sessions

4. **Admin Package** (`Events/Admin/`)
   - ✅ All CRUD operations migrated to PostgreSQL
   - Functions: ListUsers, ListVideos, ListComments, ListReplies, ListFollowers, GetCounters, ResyncCounters
   - Delete operations: DeleteUser, DeleteVideo, DeleteComment, DeleteReply

## Migration Pattern

When migrating remaining packages, follow this pattern:

### 1. Replace MongoDB imports
```go
// Remove:
import "go.mongodb.org/mongo-driver/bson"
import "go.mongodb.org/mongo-driver/mongo"

// Keep:
import "database/sql"
import Mdb "hifi/Services/Mdb"
```

### 2. Replace Collection access
```go
// Before:
collection := Mdb.DB.Collection("users")
err := collection.FindOne(ctx, bson.M{"uid": uid}).Decode(&user)

// After:
var user User
err := Mdb.DB.QueryRow(
    "SELECT id, uid, username, ... FROM users WHERE uid = $1",
    uid,
).Scan(&user.ID, &user.UID, &user.Username, ...)
```

### 3. Replace InsertOne
```go
// Before:
result, err := collection.InsertOne(ctx, data)
docID := result.InsertedID.(primitive.ObjectID).Hex()

// After:
var id int
err := Mdb.DB.QueryRow(
    "INSERT INTO table (col1, col2, ...) VALUES ($1, $2, ...) RETURNING id",
    data.Col1, data.Col2, ...,
).Scan(&id)
```

### 4. Replace UpdateOne
```go
// Before:
_, err = collection.UpdateOne(ctx, bson.M{"uid": uid}, bson.M{"$set": update})

// After:
_, err = Mdb.DB.Exec(
    "UPDATE table SET col1 = $1, col2 = $2 WHERE uid = $3",
    update.Col1, update.Col2, uid,
)
```

### 5. Replace Find with cursor
```go
// Before:
cursor, err := collection.Find(ctx, bson.M{})
defer cursor.Close(ctx)
for cursor.Next(ctx) {
    var item Item
    cursor.Decode(&item)
    items = append(items, item)
}

// After:
rows, err := Mdb.DB.Query("SELECT ... FROM table")
defer rows.Close()
for rows.Next() {
    var item Item
    rows.Scan(&item.ID, &item.Field1, ...)
    items = append(items, item)
}
```

### 6. Handle errors
```go
// Before:
if errors.Is(err, mongo.ErrNoDocuments) {
    // handle not found
}

// After:
if err == sql.ErrNoRows {
    // handle not found
}
```

## Notes

- All timestamps use PostgreSQL's `TIMESTAMP` type
- Arrays (like video_tags) use PostgreSQL's `TEXT[]` type with a custom `StringArray` type for scanning
- Foreign key relationships are enforced with CASCADE deletes (migration 007)
- Indexes have been created on commonly queried fields
- The `deleted_users` table stores soft-deleted users
- The `system_counters` table (migration 008) provides instant, accurate counts via database triggers
  - Counters are automatically maintained on INSERT/DELETE operations
  - Provides `/admin/counters` endpoint for efficient admin statistics
  - See `008_COUNTERS_TABLE.md` for detailed documentation

## Testing

After migration:
1. Run migrations: `RUN_MIGRATIONS=true go run main.go`
2. Test all API endpoints
3. Verify data integrity
4. Check for any remaining MongoDB references

