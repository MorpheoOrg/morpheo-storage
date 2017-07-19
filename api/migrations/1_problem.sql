-- +migrate Up
ALTER TABLE problem ALTER COLUMN description TYPE TEXT;
ALTER TABLE problem ALTER COLUMN description SET NOT NULL;

-- +migrate Down
ALTER TABLE problem ALTER COLUMN description TYPE VARCHAR(255) NOT NULL;
