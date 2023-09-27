-- +goose Up

CREATE TABLE kvstore (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
);

CREATE TABLE repos (
  owner_name TEXT NOT NULL,
  repo_name TEXT NOT NULL,
  last_ts INTEGER NOT NULL,
  cursor_manifest TEXT,
  cursor_dep TEXT,
  animate_ts INTEGER NOT NULL DEFAULT 0,
  UNIQUE (owner_name, repo_name)
);

CREATE TABLE donations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sponsor_id TEXT NOT NULL,
  recipient_id TEXT NOT NULL,
  last_ts INTEGER NOT NULL,
  donate_ts INTEGER NOT NULL DEFAULT 0,
  donate_attempt_ts INTEGER NOT NULL DEFAULT 0,
  UNIQUE (sponsor_id, recipient_id)
);