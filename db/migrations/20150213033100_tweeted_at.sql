
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

ALTER TABLE video RENAME TO video_;
CREATE TABLE video (
  id INTEGER PRIMARY KEY
  ,code TEXT NOT NULL UNIQUE
  ,name TEXT NOT NULL
  ,postedat datetime NOT NULL
  ,tweetedat datetime NOT NULL
  ,thumb TEXT NOT NULL
);
INSERT INTO video SELECT id, code, name, postedat, "0001-01-01 00:00:00", thumb FROM video_;
DROP TABLE video_;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE video RENAME TO video_;
CREATE TABLE video (
  id INTEGER PRIMARY KEY
  ,code TEXT NOT NULL UNIQUE
  ,name TEXT NOT NULL
  ,postedat datetime NOT NULL
  ,thumb TEXT NOT NULL
);
INSERT INTO video SELECT id, code, name, postedat, thumb FROM video_;
DROP TABLE video_;
