-- +migrate Up
ALTER TABLE algo ADD name VARCHAR(255) NOT NULL DEFAULT '';


-- +migrate Down
ALTER TABLE algo DROP COLUMN IF EXISTS name;
