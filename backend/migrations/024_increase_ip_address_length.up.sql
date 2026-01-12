-- Increase ip_address column length to handle IPv6, ports, and X-Forwarded-For lists
-- VARCHAR(45) is too short for IPv6 addresses with ports or forwarded IP chains

ALTER TABLE sessions ALTER COLUMN ip_address TYPE VARCHAR(255);
ALTER TABLE audit_logs ALTER COLUMN ip_address TYPE VARCHAR(255);
