-- 005_make_service_type_nullable.sql
-- Allow appointments without a linked service_type for ad-hoc 'Other' bookings
BEGIN;
ALTER TABLE appointment ALTER COLUMN service_type_id DROP NOT NULL;
COMMIT;
