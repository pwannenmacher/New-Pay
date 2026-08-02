-- Track when the last draft reminder was sent per self-assessment.
-- Enables interval-based reminders that actually fire, replacing a broken
-- modulo check that only matched when the age was an exact multiple of the interval.

ALTER TABLE self_assessments ADD COLUMN last_reminder_sent_at TIMESTAMP;
