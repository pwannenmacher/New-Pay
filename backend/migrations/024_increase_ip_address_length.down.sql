-- Rollback: decrease ip_address column length back to VARCHAR(45)
-- WARNING: This will truncate data if any IP addresses are longer than 45 characters

ALTER TABLE sessions ALTER COLUMN ip_address TYPE VARCHAR(45);
ALTER TABLE audit_logs ALTER COLUMN ip_address TYPE VARCHAR(45);
