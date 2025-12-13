-- Add weight field to categories table for weighted average calculation
ALTER TABLE categories ADD COLUMN weight DECIMAL(5,4) CHECK (weight >= 0.00 AND weight <= 1.00);
