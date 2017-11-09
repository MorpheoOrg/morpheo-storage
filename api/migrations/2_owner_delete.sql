-- +migrate Up
ALTER TABLE algo
DROP COLUMN owner;

ALTER TABLE data
DROP COLUMN owner;

ALTER TABLE model
DROP COLUMN owner;

ALTER TABLE problem
DROP COLUMN owner;

-- +migrate Down
ALTER TABLE algo
ADD owner UUID;

ALTER TABLE data
ADD owner UUID;

ALTER TABLE model
ADD owner UUID;

ALTER TABLE problem
ADD owner UUID;
