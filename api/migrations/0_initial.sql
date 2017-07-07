-- +migrate Up
CREATE TABLE IF NOT EXISTS problem (
  uuid UUID PRIMARY KEY,
  timestamp_upload INTEGER NOT NULL,
  name VARCHAR(255) NOT NULL,
  description VARCHAR(255) NOT NULL,
  owner UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS algo (
  uuid UUID PRIMARY KEY,
  timestamp_upload INTEGER NOT NULL,
  name VARCHAR(255) NOT NULL,
  owner UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS model (
  uuid UUID PRIMARY KEY,
  algo UUID REFERENCES algo(uuid),
  timestamp_upload INTEGER NOT NULL,
  owner UUID NOT NULL
);

CREATE TABLE IF NOT EXISTS data (
  uuid UUID PRIMARY KEY,
  timestamp_upload INTEGER NOT NULL,
  owner UUID NOT NULL
);

-- +migrate Down
DROP TABLE problem;
DROP TABLE algo;
DROP TABLE model;
DROP TABLE data;
