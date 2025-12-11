-- Drop trigger first
DROP TRIGGER IF EXISTS enforce_encrypted_records_append_only ON encrypted_records;

-- Drop function
DROP FUNCTION IF EXISTS prevent_encrypted_record_modifications();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS encrypted_records;
DROP TABLE IF EXISTS process_keys;
DROP TABLE IF EXISTS user_keys;
