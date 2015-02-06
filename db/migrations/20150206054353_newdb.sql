
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE "video" (
  "id" INTEGER PRIMARY KEY
  ,"code" TEXT NOT NULL UNIQUE
  ,"name" TEXT NOT NULL
);

CREATE TABLE "log" (
  "id" INTEGER PRIMARY KEY
  ,"videoid" bigint NOT NULL
  ,"at" datetime NOT NULL
  ,"view" bigint NOT NULL
  ,"comment" bigint NOT NULL
  ,"mylist" bigint NOT NULL
  ,FOREIGN KEY("videoid") REFERENCES "video"("id") ON DELETE CASCADE
  ,UNIQUE ("videoid", "at")
);
CREATE INDEX "idxlogat" ON "log"("at");

CREATE TABLE "tag" (
  "id" INTEGER PRIMARY KEY
  ,"name" varchar(255) NOT NULL UNIQUE
);

CREATE TABLE "videotag" (
  "videoid" bigint NOT NULL
  ,"tagid" bigint NOT NULL
  ,FOREIGN KEY("videoid") REFERENCES "video"("id") ON DELETE CASCADE
  ,FOREIGN KEY("tagid") REFERENCES "tag"("id") ON DELETE CASCADE
  ,PRIMARY KEY ("videoid", "tagid")
);

CREATE TABLE "logtag" (
  "logid" bigint NOT NULL
  ,"tagid" bigint NOT NULL
  ,"score" INTEGER NOT NULL
  ,FOREIGN KEY("logid") REFERENCES "log"("id") ON DELETE CASCADE
  ,FOREIGN KEY("tagid") REFERENCES "tag"("id") ON DELETE CASCADE
  ,PRIMARY KEY ("logid", "tagid")
);
-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE "logtag";
DROP TABLE "videotag";
DROP TABLE "tag";
DROP INDEX "idxlogat";
DROP TABLE "log";
DROP TABLE "video";
