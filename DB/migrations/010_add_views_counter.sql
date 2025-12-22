-- Migration: Add views counter to system_counters table
-- This migration adds views_count to the existing system_counters table
-- and creates a trigger to automatically maintain the views counter

-- ============================================================================
-- ADD VIEWS COUNTER COLUMN
-- ============================================================================

-- Add views_count column to system_counters table
ALTER TABLE system_counters 
ADD COLUMN IF NOT EXISTS views_count INTEGER DEFAULT 0 NOT NULL;

-- Initialize views_count with actual count from views table
UPDATE system_counters SET
    views_count = (SELECT COUNT(*) FROM views),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- ============================================================================
-- TRIGGER FUNCTION
-- ============================================================================

-- Function to increment/decrement views counter
CREATE OR REPLACE FUNCTION update_views_counter()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE system_counters SET views_count = views_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE system_counters SET views_count = views_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGER
-- ============================================================================

-- Drop existing trigger if it exists (for idempotency)
DROP TRIGGER IF EXISTS trigger_views_counter ON views;

-- Create trigger for views table
CREATE TRIGGER trigger_views_counter
    AFTER INSERT OR DELETE ON views
    FOR EACH ROW
    EXECUTE FUNCTION update_views_counter();

-- ============================================================================
-- NOTES
-- ============================================================================
-- The views_count column is automatically maintained by database triggers
-- Trigger fires AFTER INSERT/DELETE to ensure data consistency
-- Counter is updated atomically within the same transaction as the view change
-- This provides 100% accurate, instant counter reads

