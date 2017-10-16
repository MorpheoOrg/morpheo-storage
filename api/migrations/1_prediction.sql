-- +migrate Up
CREATE TABLE IF NOT EXISTS prediction (
  uuid UUID PRIMARY KEY,
  timestamp_upload INTEGER NOT NULL
);

-- +migrate Down
DROP TABLE prediction;
