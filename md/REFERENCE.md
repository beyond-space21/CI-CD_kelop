Of course, here is the API documentation for the provided Flask application.

# Superfan API Documentation

This documentation provides details on the available endpoints for the Superfan application.
### Base URLs

  - **Application:** `https://superfan.alterwork.in/api/`
  - **Janus Proxy:** `https://superfan.alterwork.in/api/streams/media`

-----

### Authentication

All requests to the application's backend services require a Firebase authentication token passed in the `Authorization` header.

**Header:**
`Authorization: Bearer <firebase_auth_token>`

-----

## Table of Contents

  - User APIs
  - Social APIs
  - Moderation APIs
  - Streaming APIs
  - Recording APIs
  - Admin APIs

-----

## User APIs

### Get User Profile

  - **Endpoint:** `GET /users/<username>`
  - **Description:** Retrieves public profile information for a specific user.
  - **Auth:** `false`
  - **URL Parameters:**
      - `username` (string, required): The display name of the user to retrieve.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "user": {
            "display_name": "testuser",
            "status": "notlive",
            "followers": 10,
            "following": 5,
            "sessions": 2
        }
    }
    ```
  - **Error Response (404 Not Found):**
    ```json
    {
        "status": "error",
        "message": "User not found"
    }
    ```

### Create New User

  - **Endpoint:** `POST /users`
  - **Description:** Creates a new user profile.
  - **Auth:** `true` (Requires a valid Firebase JWT in the Authorization header)
  - **Headers:**
      - `Authorization`: `Bearer <Firebase-ID-Token>`
  - **Request Body:**
    ```json
    {
        "username": "newuser"
    }
    ```
  - **Success Response (201 Created):**
    ```json
    {
        "status": "ok",
        "user": {
            "UID": "firebase_uid_123",
            "display_name": "newuser",
            "email": "user@example.com",
            "role": "user",
            "sessions": 0,
            "followers": 0,
            "following": 0,
            "status": "notlive",
            "created_at": "2025-06-28T12:00:00Z"
        }
    }
    ```
  - **Error Response (404 Not Found):**
    ```json
    {
        "status": "error",
        "message": "user not found"
    }
    ```

### Check Username Availability

  - **Endpoint:** `GET /users/availability`
  - **Description:** Checks if a username is available.
  - **Auth:** `false`
  - **Query Parameters:**
      - `username` (string, required): The username to check.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "Username is available"
    }
    ```
  - **Error Response (400 Bad Request):**
    ```json
    {
        "status": "error",
        "message": "Username already exists"
    }
    ```

### Edit User Profile

  - **Endpoint:** `PUT /users`
  - **Description:** Updates the profile information for the authenticated user.
  - **Auth:** `true`
  - **Headers:**
      - `Authorization`: `Bearer <Firebase-ID-Token>`
  - **Request Body (form-data):**
      - `payload` (string, required): A JSON string containing the profile data.
      - `profile_picture` (file, optional): An image file for the user's profile picture.
  - **Payload JSON Structure:**
    ```json
    {
        "payload": {
            "name": "User Full Name",
            "bio": "This is my bio.",
            "email": "contact@example.com",
            "channel_category": "Gaming",
            "stream_Language": "English",
            "twitter_link": "https://twitter.com/user",
            "youtube_link": "https://youtube.com/user",
            "instagram_link": "https://instagram.com/user"
        }
    }
    ```
  - **Success Response (200 OK):**
    ```json
    {
        "status": "success",
        "message": "Form submitted",
        "data": {
            "display_name": "currentuser",
            "name": "User Full Name",
            "bio": "This is my bio.",
            "email": "contact@example.com",
            "channel_category": "Gaming",
            "stream_Language": "English",
            "twitter_link": "https://twitter.com/user",
            "youtube_link": "https://youtube.com/user",
            "instagram_link": "https://instagram.com/user"
        }
    }
    ```
  - **Error Response (400 Bad Request):**
    ```json
    {
        "status": "error",
        "message": "Missing payload"
    }
    ```

### Get User About Page

  - **Endpoint:** `GET /users/<username>/about`
  - **Description:** Retrieves the 'about' information for a specific user.
  - **Auth:** `false`
  - **URL Parameters:**
      - `username` (string, required): The display name of the user.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "about": {
            "display_name": "testuser",
            "name": "Test User",
            "bio": "A test bio.",
            "email": "test@example.com",
            "channel_category": "Just Chatting",
            "stream_Language": "English",
            "twitter_link": "",
            "youtube_link": "",
            "instagram_link": ""
        }
    }
    ```
  - **Error Response (404 Not Found):**
    ```json
    {
        "status": "error",
        "message": "User not found"
    }
    ```

### Find Users

  - **Endpoint:** `GET /users`
  - **Description:** Searches for users or lists top users.
  - **Auth:** `false`
  - **Query Parameters:**
      - `search` (string, optional): A search term to filter users by display name.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "users": [
            {
                "display_name": "searcheduser"
            }
        ]
    }
    ```

-----

## Social APIs

### Follow a User

  - **Endpoint:** `POST /users/<username>/follow`
  - **Description:** Allows the authenticated user to follow another user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to follow.
  - **Success Response (201 Created):**
    ```json
    {
        "status": "ok"
    }
    ```

### Unfollow a User

  - **Endpoint:** `DELETE /users/<username>/follow`
  - **Description:** Allows the authenticated user to unfollow another user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to unfollow.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "Unfollowed successfully"
    }
    ```

### Check Follow Status

  - **Endpoint:** `GET /users/<username>/follow/status`
  - **Description:** Checks if the authenticated user is following another user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to check the follow status against.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "followed": true
    }
    ```

### Get Followers

  - **Endpoint:** `GET /users/followers`
  - **Description:** Retrieves the list of users who follow the authenticated user.
  - **Auth:** `true`
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "followers": [
            {
                "followed_by": "user_a"
            },
            {
                "followed_by": "user_b"
            }
        ]
    }
    ```

### Get Following

  - **Endpoint:** `GET /users/following`
  - **Description:** Retrieves the list of users the authenticated user is following.
  - **Auth:** `true`
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "following": [
            {
                "follow": "user_c"
            },
            {
                "follow": "user_d"
            }
        ]
    }
    ```

-----

## Moderation APIs

### Block a User

  - **Endpoint:** `POST /blocks/<username>`
  - **Description:** Blocks a user, preventing them from interacting with the authenticated user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to block.
  - **Success Response (201 Created):**
    ```json
    {
        "status": "ok"
    }
    ```

### Unblock a User

  - **Endpoint:** `DELETE /blocks/<username>`
  - **Description:** Unblocks a previously blocked user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to unblock.
  - **Success Response (201 Created):**
    ```json
    {
        "status": "ok"
    }
    ```

### Get Blocked Users

  - **Endpoint:** `GET /blocks`
  - **Description:** Retrieves the authenticated user's blocklist.
  - **Auth:** `true`
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "blocklist": [
            {
                "blocklist": "blocked_user"
            }
        ]
    }
    ```

### Check if User is Blocked

  - **Endpoint:** `GET /blocks/<username>/status`
  - **Description:** Checks if the authenticated user has blocked a specific user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to check.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "blocked": true
    }
    ```

### Check if I am Blocked

  - **Endpoint:** `GET /blocks/<username>/blocking-me`
  - **Description:** Checks if a specific user has blocked the authenticated user.
  - **Auth:** `true`
  - **URL Parameters:**
      - `username` (string, required): The user to check against.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "blocked": false
    }
    ```

-----

## Streaming APIs

### Create a Stream

  - **Endpoint:** `POST /streams`
  - **Description:** Creates a new live stream session.
  - **Auth:** `true`
  - **Request Body:**
    ```json
    {
        "payload": {
            "session_id": "unique_session_id",
            "username": "streamer_username",
            "title": "My Awesome Stream",
            "description": "Come watch me play!",
            "chatEnabled": true,
            "type": "new",
            "test": false
        }
    }
    ```
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "response": {
            "UID": "firebase_uid_123",
            "hookId": "unique_session_id",
            "sessions": ["unique_session_id"],
            "likes": 0,
            "views": 0,
            "maxviews": 0,
            "name": "streamer_username",
            "title": "My Awesome Stream",
            "description": "Come watch me play!",
            "start": "2025-06-28T12:30:00Z",
            "chatEnabled": true
        }
    }
    ```

### Get Live Streams

  - **Endpoint:** `GET /streams/live`
  - **Description:** Retrieves a list of all currently live streams.
  - **Auth:** `false`
  - **Success Response (200 OK):**
    ```json
    {
        "live": {
            "streamer_uid_1": {
                "UID": "streamer_uid_1",
                "hookId": "session_id_1",
                ...
            },
            "streamer_uid_2": {
                "UID": "streamer_uid_2",
                "hookId": "session_id_2",
                ...
            }
        }
    }
    ```

### Get Stream Details

  - **Endpoint:** `GET /streams/<roomId>`
  - **Description:** Retrieves details for a specific live stream.
  - **Auth:** `false`
  - **URL Parameters:**
      - `roomId` (string, required): The unique ID of the stream room.
  - **Success Response (200 OK):**
    ```json
    {
        "UID": "firebase_uid_123",
        "hookId": "unique_session_id",
        "sessions": ["unique_session_id"],
        "likes": 50,
        "views": 100,
        "maxviews": 150,
        "name": "streamer_username",
        "title": "My Awesome Stream",
        "description": "Come watch me play!",
        "start": "2025-06-28T12:30:00Z",
        "chatEnabled": true
    }
    ```
  - **Error Response (404 Not Found):**
    ```json
    {
        "status": "error",
        "message": "Room not found"
    }
    ```

### Janus Media Proxy

  - **Endpoint:** `POST /streams/media`
  - **Description:** A proxy endpoint to communicate with the Janus WebRTC server.
  - **Auth:** `false`
  - **Request Body:**
    ```json
    {
        "method": "POST",
        "path": "/<session_id>",
        "payload": {
            "janus": "message",
            "body": {
                "request": "join",
                "room": 1234,
                "ptype": "publisher",
                "display": "username"
            },
            "jsep": {
                "type": "offer",
                "sdp": "..."
            }
        }
    }
    ```
  - **Success Response:** The response from the Janus server is forwarded directly.

### Get Stream Views

  - **Endpoint:** `GET /streams/<roomId>/views`
  - **Description:** Get the current number of views for a stream.
  - **Auth:** `false`
  - **URL Parameters:**
      - `roomId` (string, required): The ID of the stream room.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "views": 125
    }
    ```

### Increment Stream View

  - **Endpoint:** `POST /streams/<roomId>/<sessionId>/views`
  - **Description:** Increments the view count for a stream. Each unique session ID is counted once.
  - **Auth:** `true`
  - **URL Parameters:**
      - `roomId` (string, required): The ID of the stream room.
      - `sessionId` (string, required): The unique session ID of the viewer.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok"
    }
    ```

-----

## Recording APIs

### Get User Recordings

  - **Endpoint:** `GET /recordings/<username>`
  - **Description:** Retrieves a list of past stream recordings for a user.
  - **Auth:** `false`
  - **URL Parameters:**
      - `username` (string, required): The username to fetch recordings for.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "user": [
            {
                "hookId": "past_session_id_1",
                "name": "testuser",
                "title": "Past Stream Title 1",
                ...
            },
            {
                "hookId": "past_session_id_2",
                "name": "testuser",
                "title": "Past Stream Title 2",
                ...
            }
        ]
    }
    ```

-----

## Admin APIs

### Get All Users

  - **Endpoint:** `GET /admin/users`
  - **Description:** Retrieves a paginated list of all users in the system.
  - **Auth:** `true` (Admin only)
  - **Query Parameters:**
      - `page` (integer, optional): Page number (default: 1)
      - `per_page` (integer, optional): Number of users per page (default: 20)
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "users": [
            {
                "display_name": "user1",
                "followers": 15,
                "following": 3,
                "status": "notlive"
            },
            {
                "display_name": "user2",
                "followers": 20,
                "following": 10,
                "status": "live_session_id"
            }
        ],
        "total": 100,
        "page": 1,
        "per_page": 20
    }
    ```

### Get All Streams

  - **Endpoint:** `GET /admin/streams`
  - **Description:** Retrieves a paginated list of all streams in the system.
  - **Auth:** `true` (Admin only)
  - **Query Parameters:**
      - `page` (integer, optional): Page number (default: 1)
      - `per_page` (integer, optional): Number of streams per page (default: 20)
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "streams": [
            {
                "hookId": "session_id_1",
                "name": "streamer_username",
                "title": "Stream Title",
                "description": "...",
                "start": "2025-06-28T12:30:00Z",
                ...
            },
            {
                "hookId": "session_id_2",
                "name": "another_streamer",
                ...
            }
        ],
        "total": 50,
        "page": 1,
        "per_page": 20
    }
    ```

### Delete a User

  - **Endpoint:** `DELETE /admin/users/<username>`
  - **Description:** Deletes a user and all their associated data.
  - **Auth:** `true` (Admin only)
  - **URL Parameters:**
      - `username` (string, required): The user to delete.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "User deleted successfully"
    }
    ```

### Delete a Stream

  - **Endpoint:** `DELETE /admin/streams/<roomId>`
  - **Description:** Deletes a stream recording and decrements the sessions count for the user with UID equal to roomId.
  - **Auth:** `true` (Admin only)
  - **URL Parameters:**
      - `roomId` (string, required): The ID of the stream to delete (also the UID of the user whose sessions will be decremented).
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "Stream deleted successfully"
    }
    ```

### Ban a User

  - **Endpoint:** `POST /admin/ban/<username>`
  - **Description:** Bans a user, preventing them from using the service.
  - **Auth:** `true` (Admin only)
  - **URL Parameters:**
      - `username` (string, required): The user to ban.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "User banned successfully"
    }
    ```

### Unban a User

  - **Endpoint:** `POST /admin/unban/<username>`
  - **Description:** Lifts a ban on a user.
  - **Auth:** `true` (Admin only)
  - **URL Parameters:**
      - `username` (string, required): The user to unban.
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "message": "User unbanned successfully"
    }
    ```

### Check Server Storage

  - **Endpoint:** `GET /admin/storage`
  - **Description:** Checks the available disk space on the server.
  - **Auth:** `true` (Admin only)
  - **Success Response (200 OK):**
    ```json
    {
        "status": "ok",
        "total": "100.0 GB",
        "used": "50.0 GB",
        "free": "50.0 GB"
    }
    ```
  - **Error Response (500 Internal Server Error):**
    ```json
    {
        "status": "error",
        "message": "Error getting storage info"
    }
    ```