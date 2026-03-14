-- Migration: Parent-child linking via WhatsApp OTP (indexes)
-- Safe to run multiple times.

CREATE INDEX IF NOT EXISTS idx_children_parent_whatsapp_number
    ON children(parent_whatsapp_number)
    WHERE parent_whatsapp_number IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_child_link_otps_pair
    ON child_link_otps(child_id, parent_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_child_link_otps_active
    ON child_link_otps(child_id, parent_id, expires_at)
    WHERE consumed_at IS NULL;

