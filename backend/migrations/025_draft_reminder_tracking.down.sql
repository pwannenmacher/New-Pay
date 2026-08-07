-- Rollback: remove draft reminder tracking column.

ALTER TABLE self_assessments DROP COLUMN last_reminder_sent_at;
