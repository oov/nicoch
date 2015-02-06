package main

import (
	"flag"
	"log"
	"sort"
	"strings"

	"github.com/oov/nicoch/db"
	"github.com/oov/nicoch/db2"

	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbFile := flag.String("db", "nicoch.sqlite3", "database filename")
	flag.Parse()

	dbm, err := db.New("sqlite3", *dbFile, modl.SqliteDialect{})
	if err != nil {
		log.Fatal(err)
	}
	dbm2, err := db2.New("sqlite3", "new"+*dbFile, modl.SqliteDialect{})
	if err != nil {
		log.Fatal(err)
	}

	tx, err := dbm2.Begin()
	if err != nil {
		log.Fatal(err)
	}

	err = MigrateVideo(dbm, tx)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	err = MigrateLog(dbm, tx)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func tagSlice(s string) []string {
	ss := strings.Split(strings.TrimSpace(s), "\n")
	if len(ss) == 1 && ss[0] == "" {
		return make([]string, 0)
	}
	sort.Strings(ss)
	return ss
}

func MigrateVideo(dbm *modl.DbMap, x modl.SqlExecutor) error {
	rows, err := dbm.Dbx.Queryx(`select * from video`)
	if err != nil {
		return err
	}
	var vv db.Video
	for rows.Next() {
		err = rows.StructScan(&vv)
		if err != nil {
			return err
		}

		v := db2.Video{
			Code: vv.ID,
			Name: vv.ID,
		}
		err = x.Insert(&v)
		if err != nil {
			log.Println(vv.ID, err)
			continue
		}

		for _, tag := range tagSlice(vv.Tags) {
			t, err := db2.GetTag(x, tag)
			if err != nil {
				return err
			}

			err = x.Insert(&db2.VideoTag{
				VideoID: v.ID,
				TagID:   t.ID,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func MigrateLog(dbm *modl.DbMap, x modl.SqlExecutor) error {
	rows, err := dbm.Dbx.Queryx(`select * from log`)
	if err != nil {
		return err
	}
	var ll db.Log
	for rows.Next() {
		err = rows.StructScan(&ll)
		if err != nil {
			return err
		}

		v, err := db2.GetVideoByCode(x, ll.VideoID)
		if err != nil {
			return err
		}

		l := db2.Log{
			VideoID: v.ID,
			At:      ll.At,
			View:    int64(ll.View),
			Comment: int64(ll.Comment),
			Mylist:  int64(ll.Mylist),
		}
		err = x.Insert(&l)
		if err != nil {
			log.Println(ll.VideoID, ll.At, err)
			continue
		}

		for _, tagDiff := range tagSlice(ll.TagsDiff) {
			tag := tagDiff[1:]
			t, err := db2.GetTag(x, tag)
			if err != nil {
				return err
			}

			var score int64
			if tagDiff[0] == '+' {
				score = 1
			} else {
				score = -1
			}
			err = x.Insert(&db2.LogTag{
				LogID: l.ID,
				TagID: t.ID,
				Score: score,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
