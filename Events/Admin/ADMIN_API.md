# Admin API Documentation

This document provides comprehensive API documentation for the Admin endpoints in the Hifi backend.

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Authorization](#authorization)
- [Endpoints](#endpoints)
  - [Get Counters](#1-get-counters)
  - [Resync Counters](#2-resync-counters)
  - [List Users](#3-list-users)
  - [List Videos](#3-list-videos)
  - [List Comments](#4-list-comments)
  - [List Replies](#5-list-replies)
  - [List Followers](#6-list-followers)
  - [Delete User](#7-delete-user)
  - [Delete Video](#8-delete-video)
  - [Delete Comment](#9-delete-comment)
  - [Delete Reply](#10-delete-reply)
- [Error Responses](#error-responses)

---

## Overview

The Admin API provides endpoints for administrative operations including listing and deleting users, videos, comments, and replies. All endpoints require admin role authentication.

**Base Path:** `/admin`

---

## Authentication

All endpoints require authentication via JWT token. The token should be included in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

The token is validated using the `Auth.GetClaims` function, which extracts and validates the JWT claims from the request.

---

## Authorization

All admin endpoints require the authenticated user to have the `admin` role. Users with roles `user` or `creator` will receive a `403 Forbidden` response.

**Required Role:** `admin`

---

## Endpoints

### 1. Get Counters

Retrieves aggregated counters for all entities in the system. Uses a dedicated `system_counters` table maintained automatically by database triggers. Provides instant, 100% accurate counts without scanning large tables.

**Endpoint:** `GET /admin/counters`

**Authentication:** Required (Admin only)

**Request Example:**
```http
GET /admin/counters
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
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

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch counters

**Notes:**
- Counters are maintained automatically by database triggers on INSERT/DELETE operations
- Provides instant reads (single row SELECT) regardless of table size
- 100% accurate - counters are updated atomically within the same transaction as data changes
- No performance degradation as tables grow - always instant
- Counters are updated in real-time as data changes
- The `updated_at` timestamp shows when counters were last updated

---

### 2. Resync Counters

Manually resyncs counters with actual table counts. Useful if counters get out of sync due to direct database operations or trigger failures.

**Endpoint:** `POST /admin/counters/resync`

**Authentication:** Required (Admin only)

**Request Example:**
```http
POST /admin/counters/resync
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "message": "Counters resynced successfully"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to resync counters

**Notes:**
- This operation recalculates all counters from actual table counts
- Use this if you suspect counters are out of sync
- The operation uses a transaction to ensure atomicity
- May take longer on large tables as it performs COUNT(*) operations

---

### 3. List Users

Retrieves a paginated list of all users with optional filtering.

**Endpoint:** `GET /admin/users`

**Authentication:** Required (Admin only)

**Query Parameters:**

**Pagination:**
- `limit` (integer, optional): Number of users to return per page
  - Default: `20`
  - Maximum: `100`
  - Must be greater than 0
- `offset` (integer, optional): Number of users to skip
  - Default: `0`
  - Must be greater than or equal to 0

**Text Filters:**
- `username` (string, optional): Filter by username (case-insensitive partial match)
- `name` (string, optional): Filter by name (case-insensitive partial match)
- `role` (string, optional): Filter by role (exact match, case-insensitive)
  - Valid values: `user`, `creator`, `admin`
- `uid` (string, optional): Filter by user UID (exact match)

**Numeric Range Filters:**
- `followers_min` (integer, optional): Minimum number of followers
- `followers_max` (integer, optional): Maximum number of followers
- `following_min` (integer, optional): Minimum number of following
- `following_max` (integer, optional): Maximum number of following
- `total_videos_min` (integer, optional): Minimum number of total videos
- `total_videos_max` (integer, optional): Maximum number of total videos

**Date Range Filters:**
- `created_after` (string, optional): Filter users created after this date (ISO 8601 format, e.g., `2024-01-01T00:00:00Z`)
- `created_before` (string, optional): Filter users created before this date (ISO 8601 format)
- `updated_after` (string, optional): Filter users updated after this date (ISO 8601 format)
- `updated_before` (string, optional): Filter users updated before this date (ISO 8601 format)

**Request Examples:**

Filter by username:
```http
GET /admin/users?limit=20&offset=0&username=john
Authorization: Bearer <jwt_token>
```

Filter by role:
```http
GET /admin/users?limit=20&offset=0&role=creator
Authorization: Bearer <jwt_token>
```

Filter by followers range:
```http
GET /admin/users?limit=20&offset=0&followers_min=100&followers_max=1000
Authorization: Bearer <jwt_token>
```

Filter by creation date:
```http
GET /admin/users?limit=20&offset=0&created_after=2024-01-01T00:00:00Z
Authorization: Bearer <jwt_token>
```

Combined filters:
```http
GET /admin/users?limit=20&offset=0&role=user&followers_min=50&total_videos_min=5
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "users": [
    {
      "id": 1,
      "uid": "abc123def456...",
      "username": "johndoe",
      "name": "John Doe",
      "role": "user",
      "profile_picture": "https://example.com/pic.jpg",
      "followers": 150,
      "following": 75,
      "total_streams": 42,
      "total_videos": 10,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-20T14:22:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 1,
  "filters": {
    "username": "john",
    "role": "user",
    "followers": {
      "min": "100",
      "max": "1000"
    },
    "created_at": {
      "after": "2024-01-01T00:00:00Z",
      "before": ""
    }
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch users

**Notes:**
- Results are ordered by `created_at` in descending order (newest first)
- Multiple filters can be combined using AND logic
- Text filters (`username`, `name`) use case-insensitive partial matching (LIKE)
- Role filter uses exact match (case-insensitive)
- UID filter uses exact match
- Numeric range filters can be used independently or together (min and/or max)
- Date range filters can be used independently or together (after and/or before)
- Invalid date formats are ignored
- Invalid numeric values are ignored
- The `filters` object in the response shows all applied filters
- The `count` field represents the number of users returned in the current page

---

### 4. List Videos

Retrieves a paginated list of all videos with optional filtering.

**Endpoint:** `GET /admin/videos`

**Authentication:** Required (Admin only)

**Query Parameters:**

**Pagination:**
- `limit` (integer, optional): Number of videos to return per page
  - Default: `20`
  - Maximum: `100`
  - Must be greater than 0
- `offset` (integer, optional): Number of videos to skip
  - Default: `0`
  - Must be greater than or equal to 0

**Text Filters:**
- `video_id` (string, optional): Filter by video ID (exact match)
- `video_title` (string, optional): Filter by video title (case-insensitive partial match)
- `video_description` (string, optional): Filter by video description (case-insensitive partial match)
- `user_username` (string, optional): Filter by uploader username (case-insensitive partial match)
- `user_uid` (string, optional): Filter by uploader UID (exact match)
- `video_tag` (string, optional): Filter by video tag (case-insensitive partial match, searches within video_tags array)

**Numeric Range Filters:**
- `video_views_min` (integer, optional): Minimum number of views
- `video_views_max` (integer, optional): Maximum number of views
- `video_upvotes_min` (integer, optional): Minimum number of upvotes
- `video_upvotes_max` (integer, optional): Maximum number of upvotes
- `video_downvotes_min` (integer, optional): Minimum number of downvotes
- `video_downvotes_max` (integer, optional): Maximum number of downvotes
- `video_comments_min` (integer, optional): Minimum number of comments
- `video_comments_max` (integer, optional): Maximum number of comments

**Date Range Filters:**
- `created_after` (string, optional): Filter videos created after this date (ISO 8601 format, e.g., `2024-01-01T00:00:00Z`)
- `created_before` (string, optional): Filter videos created before this date (ISO 8601 format)
- `updated_after` (string, optional): Filter videos updated after this date (ISO 8601 format)
- `updated_before` (string, optional): Filter videos updated before this date (ISO 8601 format)

**Request Examples:**

Filter by video title:
```http
GET /admin/videos?limit=20&offset=0&video_title=gaming
Authorization: Bearer <jwt_token>
```

Filter by user username:
```http
GET /admin/videos?limit=20&offset=0&user_username=johndoe
Authorization: Bearer <jwt_token>
```

Filter by video tag:
```http
GET /admin/videos?limit=20&offset=0&video_tag=funny
Authorization: Bearer <jwt_token>
```

Filter by views range:
```http
GET /admin/videos?limit=20&offset=0&video_views_min=100&video_views_max=10000
Authorization: Bearer <jwt_token>
```

Filter by upvotes and comments:
```http
GET /admin/videos?limit=20&offset=0&video_upvotes_min=50&video_comments_min=10
Authorization: Bearer <jwt_token>
```

Filter by creation date:
```http
GET /admin/videos?limit=20&offset=0&created_after=2024-01-01T00:00:00Z
Authorization: Bearer <jwt_token>
```

Combined filters:
```http
GET /admin/videos?limit=20&offset=0&user_username=john&video_views_min=100&video_upvotes_min=10&created_after=2024-01-01T00:00:00Z
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "videos": [
    {
      "video_id": "abc123...",
      "video_url": "videos/abc123...",
      "video_thumbnail": "thumbnails/videos/abc123....jpg",
      "video_title": "Gaming Highlights",
      "video_description": "Best gaming moments",
      "video_tags": ["gaming", "funny"],
      "video_views": 150,
      "video_upvotes": 42,
      "video_downvotes": 2,
      "video_comments": 10,
      "user_uid": "user123...",
      "user_username": "johndoe",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-20T14:22:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 1,
  "filters": {
    "video_title": "gaming",
    "user_username": "john",
    "video_views": {
      "min": "100",
      "max": "10000"
    },
    "video_upvotes": {
      "min": "10",
      "max": ""
    },
    "created_at": {
      "after": "2024-01-01T00:00:00Z",
      "before": ""
    }
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch videos

**Notes:**
- Results are ordered by `created_at` in descending order (newest first)
- Multiple filters can be combined using AND logic
- Text filters (`video_title`, `video_description`, `user_username`) use case-insensitive partial matching (LIKE)
- `video_id` and `user_uid` filters use exact match
- `video_tag` filter searches within the `video_tags` array (case-insensitive partial match)
- Numeric range filters can be used independently or together (min and/or max)
- Date range filters can be used independently or together (after and/or before)
- Invalid date formats are ignored
- Invalid numeric values are ignored
- The `filters` object in the response shows all applied filters

---

### 5. List Comments

Retrieves a paginated list of all comments with optional filtering.

**Endpoint:** `GET /admin/comments`

**Authentication:** Required (Admin only)

**Query Parameters:**
- `limit` (integer, optional): Number of comments to return per page
  - Default: `20`
  - Maximum: `100`
  - Must be greater than 0
- `offset` (integer, optional): Number of comments to skip
  - Default: `0`
  - Must be greater than or equal to 0
- `filter` (string, optional): Filter by comment_id, comment text, comment_by_username, or commented_to (video_id) (case-insensitive partial match)

**Request Example:**
```http
GET /admin/comments?limit=20&offset=0&filter=great
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "comments": [
    {
      "comment_id": "xyz789abc123...",
      "commented_by": "user_uid_123",
      "commented_to": "abc123def456...",
      "commented_at": "2024-01-01T00:00:00Z",
      "comment": "Great video!",
      "comment_by_username": "johndoe",
      "total_replies": 5
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 1,
  "filter": "great"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch comments

**Notes:**
- Results are ordered by `commented_at` in descending order (newest first)
- The `filter` parameter searches across comment_id, comment text, comment_by_username, and commented_to fields

---

### 6. List Replies

Retrieves a paginated list of all replies with optional filtering.

**Endpoint:** `GET /admin/replies`

**Authentication:** Required (Admin only)

**Query Parameters:**
- `limit` (integer, optional): Number of replies to return per page
  - Default: `20`
  - Maximum: `100`
  - Must be greater than 0
- `offset` (integer, optional): Number of replies to skip
  - Default: `0`
  - Must be greater than or equal to 0
- `filter` (string, optional): Filter by reply_id, reply text, reply_by_username, or replied_to (comment_id) (case-insensitive partial match)

**Request Example:**
```http
GET /admin/replies?limit=20&offset=0&filter=agree
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "replies": [
    {
      "reply_id": "def456ghi789...",
      "replied_by": "user_uid_456",
      "replied_to": "xyz789abc123...",
      "replied_at": "2024-01-01T12:00:00Z",
      "reply": "I agree!",
      "reply_by_username": "janedoe"
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 1,
  "filter": "agree"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch replies

**Notes:**
- Results are ordered by `replied_at` in descending order (newest first)
- The `filter` parameter searches across reply_id, reply text, reply_by_username, and replied_to fields

---

### 7. List Followers

Retrieves a paginated list of all follower relationships with optional filtering.

**Endpoint:** `GET /admin/followers`

**Authentication:** Required (Admin only)

**Query Parameters:**

**Pagination:**
- `limit` (integer, optional): Number of follower relationships to return per page
  - Default: `20`
  - Maximum: `100`
  - Must be greater than 0
- `offset` (integer, optional): Number of follower relationships to skip
  - Default: `0`
  - Must be greater than or equal to 0

**Text Filters:**
- `followed_by_username` (string, optional): Filter by username of the user who is following (case-insensitive partial match)
- `followed_to_username` (string, optional): Filter by username of the user being followed (case-insensitive partial match)
- `followed_by` (string, optional): Filter by UID of the user who is following (exact match)
- `followed_to` (string, optional): Filter by UID of the user being followed (exact match)

**Date Range Filters:**
- `followed_after` (string, optional): Filter follower relationships created after this date (ISO 8601 format, e.g., `2024-01-01T00:00:00Z`)
- `followed_before` (string, optional): Filter follower relationships created before this date (ISO 8601 format)

**Request Examples:**

Filter by followed_by username:
```http
GET /admin/followers?limit=20&offset=0&followed_by_username=user1
Authorization: Bearer <jwt_token>
```

Filter by followed_to username:
```http
GET /admin/followers?limit=20&offset=0&followed_to_username=johndoe
Authorization: Bearer <jwt_token>
```

Filter by followed_by UID:
```http
GET /admin/followers?limit=20&offset=0&followed_by=user_uid_123
Authorization: Bearer <jwt_token>
```

Filter by date range:
```http
GET /admin/followers?limit=20&offset=0&followed_after=2024-01-01T00:00:00Z
Authorization: Bearer <jwt_token>
```

Combined filters:
```http
GET /admin/followers?limit=20&offset=0&followed_by_username=user1&followed_to_username=johndoe
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "followers": [
    {
      "followed_by": "user_uid_123",
      "followed_by_username": "user1",
      "followed_to": "user_uid_456",
      "followed_to_username": "johndoe",
      "followed_at": "2024-01-01T00:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "count": 1,
  "filters": {
    "followed_by_username": "user1",
    "followed_to_username": "johndoe",
    "followed_at": {
      "after": "2024-01-01T00:00:00Z",
      "before": ""
    }
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `500 Internal Server Error`: Failed to fetch followers

**Notes:**
- Results are ordered by `followed_at` in descending order (newest first)
- Multiple filters can be combined using AND logic
- Username filters (`followed_by_username`, `followed_to_username`) use case-insensitive partial matching (LIKE)
- UID filters (`followed_by`, `followed_to`) use exact match
- Date range filters can be used independently or together (after and/or before)
- Invalid date formats are ignored
- The `filters` object in the response shows all applied filters
- Usernames are included in the response for better readability

---

### 8. Delete User

Deletes a user account by UID. This is a soft delete that moves the user to the `deleted_users` table.

**Endpoint:** `DELETE /admin/users/{uid}`

**Authentication:** Required (Admin only)

**URL Parameters:**
- `uid` (string, required): The UID of the user to delete

**Request Example:**
```http
DELETE /admin/users/abc123def456...
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "message": "User deleted successfully"
}
```

**Error Responses:**
- `400 Bad Request`: User UID is required
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: User not found
- `500 Internal Server Error`: 
  - Failed to load user
  - Failed to start transaction
  - Failed to archive deleted user
  - Failed to delete user
  - Failed to complete deletion

**Notes:**
- Deletion is performed using a database transaction to ensure atomicity
- User data is moved to the `deleted_users` table before deletion
- The operation includes a `deleted_at` timestamp in the archived record
- **Elasticsearch Integration**: Automatically deletes the user from Elasticsearch index (non-blocking operation)
  - If Elasticsearch deletion fails, the operation logs an error but does not fail the deletion
- Foreign key CASCADE will automatically delete:
  - All videos (videos table)
  - All video_on_upload records
  - All followers relationships (both followed_by and followed_to)
  - All blocklists relationships (both blocked_by and blocked_to)
  - All upvotes, downvotes, comments, replies, views

---

### 9. Delete Video

Deletes a video by videoID. This permanently removes the video and all associated data.

**Endpoint:** `DELETE /admin/videos/{videoID}`

**Authentication:** Required (Admin only)

**URL Parameters:**
- `videoID` (string, required): The video ID to delete

**Request Example:**
```http
DELETE /admin/videos/abc123def456...
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "message": "Video deleted successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Video ID is required
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: Video not found
- `500 Internal Server Error`: 
  - Failed to load video
  - Failed to start transaction
  - Failed to delete video
  - Failed to update user video count
  - Failed to complete deletion

**Notes:**
- Deletion is performed using a database transaction to ensure atomicity
- **Elasticsearch Integration**: Automatically deletes the video from Elasticsearch index (non-blocking operation)
  - If Elasticsearch deletion fails, the operation logs an error but does not fail the deletion
- Foreign key CASCADE will automatically delete:
  - All upvotes, downvotes, comments (and their replies), views
- The user's `total_videos` count is decremented
- The video file and thumbnail remain in storage (consider adding cleanup if needed)

---

### 10. Delete Comment

Deletes a comment by commentID. This permanently removes the comment and all associated replies.

**Endpoint:** `DELETE /admin/comments/{commentID}`

**Authentication:** Required (Admin only)

**URL Parameters:**
- `commentID` (string, required): The comment ID to delete

**Request Example:**
```http
DELETE /admin/comments/xyz789abc123...
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "message": "Comment deleted successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Comment ID is required
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: Comment not found
- `500 Internal Server Error`: 
  - Failed to load comment
  - Failed to start transaction
  - Failed to delete comment
  - Failed to update video comment count
  - Failed to complete deletion

**Notes:**
- Deletion is performed using a database transaction to ensure atomicity
- Foreign key CASCADE will automatically delete all replies to this comment
- The video's `video_comments` count is decremented

---

### 11. Delete Reply

Deletes a reply by replyID.

**Endpoint:** `DELETE /admin/replies/{replyID}`

**Authentication:** Required (Admin only)

**URL Parameters:**
- `replyID` (string, required): The reply ID to delete

**Request Example:**
```http
DELETE /admin/replies/def456ghi789...
Authorization: Bearer <jwt_token>
```

**Success Response (200 OK):**
```json
{
  "status": "success",
  "message": "Reply deleted successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Reply ID is required
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: Reply not found
- `500 Internal Server Error`: 
  - Failed to load reply
  - Failed to start transaction
  - Failed to delete reply
  - Failed to update comment reply count
  - Failed to complete deletion

**Notes:**
- Deletion is performed using a database transaction to ensure atomicity
- The comment's `total_replies` count is decremented

---


## Error Responses

All error responses follow a consistent format:

```json
{
  "status": "error",
  "message": "Error message description"
}
```

**Common HTTP Status Codes:**
- `400 Bad Request`: Invalid request parameters or body
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: User does not have admin role
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

---

## Implementation Notes

### Admin Authorization

All admin endpoints use the `requireAdmin` function which:
1. Validates the JWT token
2. Fetches the user from the database
3. Checks if the user's role is `"admin"`
4. Returns `403 Forbidden` if the user is not an admin

### Transaction Safety

All delete operations use database transactions to ensure atomicity:
- If any step fails, the transaction is rolled back
- This prevents partial deletions and data inconsistency

### Foreign Key CASCADE

The database schema uses foreign key constraints with `ON DELETE CASCADE`:
- Deleting a user automatically deletes all related data (videos, followers, comments, etc.)
- Deleting a video automatically deletes all related data (upvotes, downvotes, comments, replies, views)
- Deleting a comment automatically deletes all replies to that comment

### Filtering

All list endpoints support optional filtering via the `filter` query parameter:
- Filters are case-insensitive
- Filters use SQL `LIKE` with `%filter%` pattern matching
- Filters search across multiple relevant fields for each resource type

### Pagination

All list endpoints support pagination:
- `limit`: Number of results per page (default: 20, max: 100)
- `offset`: Number of results to skip (default: 0)
- Results are ordered by creation timestamp in descending order (newest first)

---

## Changelog

- Initial API documentation created
- Added Elasticsearch integration for deletion operations
  - Users are automatically removed from Elasticsearch index when deleted via Admin Delete User endpoint
  - Videos are automatically removed from Elasticsearch index when deleted via Admin Delete Video endpoint

