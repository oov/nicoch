
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE video RENAME TO video_;
CREATE TABLE video (
  id INTEGER PRIMARY KEY
  ,code TEXT NOT NULL UNIQUE
  ,name TEXT NOT NULL
  ,postedat datetime NOT NULL
  ,thumb TEXT NOT NULL
);
INSERT INTO video SELECT id, code, name, "2000-01-01 00:00:00", "http://127.0.0.1/" FROM video_;
DROP TABLE video_;
-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE video RENAME TO video_;
CREATE TABLE video (
  id INTEGER PRIMARY KEY
  ,code TEXT NOT NULL UNIQUE
  ,name TEXT NOT NULL
);
INSERT INTO video SELECT id, code, name FROM video_;
DROP TABLE video_;
