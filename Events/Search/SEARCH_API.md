# Search API Documentation

This document provides comprehensive API documentation for the Search endpoints in the Hifi backend.

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Endpoints](#endpoints)
  - [Search Users](#1-search-users)
  - [Search Videos](#2-search-videos)
- [Error Responses](#error-responses)
- [Search Behavior](#search-behavior)

---

## Overview

The Search API provides endpoints for searching users and videos using Elasticsearch. These endpoints are publicly accessible and do not require authentication.

**Base Path:** `/search`

**Search Engine:** Elasticsearch

---

## Endpoints

### 1. Search Users

Searches for users by username and profile picture using Elasticsearch.

**Endpoint:** `GET /search/users/{query}`

**Authentication:** Not required

**URL Parameters:**
- `query` (string, required): Search query string to match against username and profile picture

**Query Parameters:**
- `limit` (integer, optional): Maximum number of results to return
  - Default: `10`
  - Maximum: `100`
  - Must be greater than 0

**Search Fields:**
- `username` (weighted 2x)
- `profile_picture`

**Request Example:**

```http
GET /search/users/john?limit=20
```

**Success Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "users": [
      {
        "uid": "user_123",
        "username": "john_doe",
        "profile_picture": "https://example.com/profile.jpg"
      },
      {
        "uid": "user_456",
        "username": "johnny",
        "profile_picture": "https://example.com/profile2.jpg"
      }
    ],
    "query": "john",
    "limit": 20,
    "count": 2
  }
}
```

**Response Fields:**
- `users` (array): Array of user objects matching the search query
  - `uid` (string): Unique identifier for the user
  - `username` (string): Username of the user
  - `profile_picture` (string): URL to the user's profile picture
- `query` (string): The original search query
- `limit` (integer): The limit used for the search
- `count` (integer): Number of results returned

**Error Responses:**

**400 Bad Request:**
```json
{
  "success": false,
  "error": "Query parameter is required"
}
```

**500 Internal Server Error:**
```json
{
  "success": false,
  "error": "Failed to search users"
}
```

---

### 2. Search Videos

Searches for videos by title, description, and tags using Elasticsearch.

**Endpoint:** `GET /search/videos/{query}`

**Authentication:** Not required

**URL Parameters:**
- `query` (string, required): Search query string to match against video title, description, and tags

**Query Parameters:**
- `limit` (integer, optional): Maximum number of results to return
  - Default: `10`
  - Maximum: `100`
  - Must be greater than 0

**Search Fields:**
- `video_title` (weighted 3x)
- `video_description` (weighted 2x)
- `video_tags` (exact term matching)

**Search Behavior:**
- The search uses a `bool` query with `should` clauses
- Matches video title or description using multi-match (fuzzy matching)
- Matches video tags using exact term matching
- At least one condition must match (`minimum_should_match: 1`)
- Results are ranked by relevance score

**Request Example:**

```http
GET /search/videos/gaming?limit=15
```

**Success Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "videos": [
      {
        "video_id": "video_123",
        "video_title": "Gaming Tutorial",
        "video_description": "Learn how to play this amazing game",
        "video_tags": ["gaming", "tutorial", "fun"]
      },
      {
        "video_id": "video_456",
        "video_title": "Best Gaming Setup",
        "video_description": "Review of gaming equipment",
        "video_tags": ["gaming", "review", "equipment"]
      }
    ],
    "query": "gaming",
    "limit": 15,
    "count": 2
  }
}
```

**Response Fields:**
- `videos` (array): Array of video objects matching the search query
  - `video_id` (string): Unique identifier for the video
  - `video_title` (string): Title of the video
  - `video_description` (string): Description of the video
  - `video_tags` (array): Array of tags associated with the video
- `query` (string): The original search query
- `limit` (integer): The limit used for the search
- `count` (integer): Number of results returned

**Error Responses:**

**400 Bad Request:**
```json
{
  "success": false,
  "error": "Query parameter is required"
}
```

**500 Internal Server Error:**
```json
{
  "success": false,
  "error": "Failed to search videos"
}
```

---

## Error Responses

All endpoints follow a standardized error response format:

**Standard Error Response:**
```json
{
  "success": false,
  "error": "<error_message>"
}
```

**Common HTTP Status Codes:**
- `400 Bad Request`: Invalid request parameters
- `500 Internal Server Error`: Server error or Elasticsearch unavailable

---

## Search Behavior

### Elasticsearch Integration

- Both endpoints use Elasticsearch for fast, full-text search capabilities
- If Elasticsearch is not enabled or unavailable, the endpoints will return a `500 Internal Server Error`
- Search operations are non-blocking and optimized for performance

### User Search

- Searches across `username` (weighted 2x) and `profile_picture` fields
- Uses `multi_match` query for fuzzy matching
- Results are ranked by relevance score

### Video Search

- Searches across `video_title` (weighted 3x), `video_description` (weighted 2x), and `video_tags` fields
- Uses a combination of `multi_match` for text fields and `terms` query for exact tag matching
- Title matches are weighted highest (3x), followed by description (2x)
- Tag matching uses exact term matching (case-insensitive)
- Results are ranked by relevance score

### Query Processing

- The search query is processed as-is (no automatic lowercasing for user search)
- For video search, the query is split into terms for tag matching (lowercased)
- Special characters in queries are handled by Elasticsearch's standard analyzer

### Result Limits

- Default limit: `10` results
- Maximum limit: `100` results
- If limit exceeds maximum, it is automatically capped at `100`
- If limit is invalid or less than 1, default limit (`10`) is used

---

## Notes

- Search results are returned in order of relevance (highest score first)
- Empty search queries will return an empty result set
- The search is case-insensitive for most fields
- Elasticsearch must be properly configured and running for these endpoints to function
- Indexed data is automatically updated when users or videos are created, updated, or deleted

