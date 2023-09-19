-- +goose Up

CREATE TABLE donations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sponsor_id TEXT NOT NULL,
  recipient_id TEXT NOT NULL,
  last_ts INTEGER NOT NULL,
  animate_ts INTEGER NOT NULL DEFAULT 0,
  animate_attempt_ts INTEGER NOT NULL DEFAULT 0,
  donable_ts INTEGER NOT NULL DEFAULT 0,
  donate_ts INTEGER NOT NULL DEFAULT 0,
  donate_attempt_ts INTEGER NOT NULL DEFAULT 0,
  UNIQUE (sponsor_id, recipient_id)
);