# Schema Improvements: Foreign Key Constraints

## Overview

This migration (`007_add_foreign_key_constraints.sql`) adds comprehensive foreign key constraints to enforce referential integrity across all tables in the database.

## What Changed

### Before
- Tables had relationships stored as VARCHAR fields (e.g., `user_uid`, `video_id`)
- No database-level enforcement of relationships
- Orphaned records could exist when users/videos were deleted
- Manual cleanup required in application code

### After
- All relationships are enforced with foreign key constraints
- Automatic cascading deletes when parent records are deleted
- Database-level referential integrity
- No orphaned records possible

## Foreign Key Relationships Added

### Videos Tables

1. **videos.user_uid** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all their videos are automatically deleted
   - `ON UPDATE CASCADE`: If user UID changes, all references are updated

2. **video_on_upload.user_uid** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, incomplete uploads are deleted

### Social Tables

3. **followers.followed_by** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all relationships where they are following are deleted

4. **followers.followed_to** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all relationships where they are followed are deleted

5. **blocklists.blocked_by** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all blocks they created are deleted

6. **blocklists.blocked_to** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all blocks against them are deleted

### Engagement Tables

7. **upvotes.upvoted_by** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all their upvotes are deleted

8. **upvotes.upvoted_to** → `videos.video_id`
   - `ON DELETE CASCADE`: When a video is deleted, all its upvotes are deleted

9. **downvotes.downvoted_by** → `users.uid`
   - `ON DELETE CASCADE`: When a user is deleted, all their downvotes are deleted

10. **downvotes.downvoted_to** → `videos.video_id`
    - `ON DELETE CASCADE`: When a video is deleted, all its downvotes are deleted

11. **comments.commented_by** → `users.uid`
    - `ON DELETE CASCADE`: When a user is deleted, all their comments are deleted

12. **comments.commented_to** → `videos.video_id`
    - `ON DELETE CASCADE`: When a video is deleted, all its comments are deleted

13. **replies.replied_by** → `users.uid`
    - `ON DELETE CASCADE`: When a user is deleted, all their replies are deleted

14. **replies.replied_to** → `comments.comment_id`
    - `ON DELETE CASCADE`: When a comment is deleted, all its replies are deleted

15. **views.viewed_by** → `users.uid`
    - `ON DELETE CASCADE`: When a user is deleted, all their view records are deleted

16. **views.viewed_to** → `videos.video_id`
    - `ON DELETE CASCADE`: When a video is deleted, all its view records are deleted

## Cascading Delete Behavior

### When a User is Deleted

The following are automatically deleted (in order):
1. All videos owned by the user
   - Which triggers deletion of:
     - All upvotes on those videos
     - All downvotes on those videos
     - All comments on those videos
       - Which triggers deletion of all replies to those comments
     - All views of those videos
2. All incomplete uploads (`video_on_upload`)
3. All follower relationships (both directions)
4. All blocklist relationships (both directions)
5. All upvotes/downvotes/comments/replies/views created by the user

### When a Video is Deleted

The following are automatically deleted:
1. All upvotes on the video
2. All downvotes on the video
3. All comments on the video
   - Which triggers deletion of all replies to those comments
4. All views of the video

### When a Comment is Deleted

The following are automatically deleted:
1. All replies to the comment

## Application Code Changes

### DeleteUser Function

The `DeleteUser` function in `Events/Users/Users.go` has been updated to:

1. **Query videos before deletion**: Fetches all video IDs for Qdrant cleanup
2. **Remove from Qdrant**: Deletes videos from vector search database (if configured)
3. **Archive user**: Moves user to `deleted_users` table
4. **Delete user**: Removes from `users` table, which triggers CASCADE deletes

**Note**: The database now handles all relationship cleanup automatically. The application only needs to:
- Clean up external services (Qdrant)
- Archive the user record

## Benefits

1. **Data Integrity**: Impossible to have orphaned records
2. **Simplified Code**: No need to manually delete related records
3. **Performance**: Database handles cascading deletes efficiently
4. **Consistency**: All deletions follow the same rules
5. **Safety**: Prevents accidental data corruption

## Migration Notes

- This migration is **safe to run** on existing databases
- It will fail if there are existing orphaned records (which is good - it means data integrity issues)
- If migration fails due to orphaned records, you'll need to clean them up first
- The migration uses `IF NOT EXISTS` patterns where possible, but foreign keys don't support this, so it will fail if constraints already exist

## Testing

After running this migration, test:

1. **User Deletion**: Delete a user and verify all related data is removed
2. **Video Deletion**: Delete a video and verify all engagement data is removed
3. **Comment Deletion**: Delete a comment and verify all replies are removed
4. **Orphan Prevention**: Try to insert a video with invalid `user_uid` - should fail

## Rollback

If you need to rollback this migration:

```sql
-- Remove all foreign key constraints
ALTER TABLE videos DROP CONSTRAINT IF EXISTS fk_videos_user_uid;
ALTER TABLE video_on_upload DROP CONSTRAINT IF EXISTS fk_video_on_upload_user_uid;
ALTER TABLE followers DROP CONSTRAINT IF EXISTS fk_followers_followed_by;
ALTER TABLE followers DROP CONSTRAINT IF EXISTS fk_followers_followed_to;
ALTER TABLE blocklists DROP CONSTRAINT IF EXISTS fk_blocklists_blocked_by;
ALTER TABLE blocklists DROP CONSTRAINT IF EXISTS fk_blocklists_blocked_to;
ALTER TABLE upvotes DROP CONSTRAINT IF EXISTS fk_upvotes_upvoted_by;
ALTER TABLE upvotes DROP CONSTRAINT IF EXISTS fk_upvotes_upvoted_to;
ALTER TABLE downvotes DROP CONSTRAINT IF EXISTS fk_downvotes_downvoted_by;
ALTER TABLE downvotes DROP CONSTRAINT IF EXISTS fk_downvotes_downvoted_to;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_commented_by;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_commented_to;
ALTER TABLE replies DROP CONSTRAINT IF EXISTS fk_replies_replied_by;
ALTER TABLE replies DROP CONSTRAINT IF EXISTS fk_replies_replied_to;
ALTER TABLE views DROP CONSTRAINT IF EXISTS fk_views_viewed_by;
ALTER TABLE views DROP CONSTRAINT IF EXISTS fk_views_viewed_to;
```

## Future Considerations

1. **Soft Deletes**: If you want to keep videos when users are deleted, consider:
   - Changing `ON DELETE CASCADE` to `ON DELETE SET NULL` for videos
   - Making `user_uid` nullable in videos table
   - Updating application logic to handle NULL user references

2. **Audit Trail**: Consider adding audit tables to track deletions if needed for compliance

3. **Performance**: Monitor cascading delete performance with large datasets

