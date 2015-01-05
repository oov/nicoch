package main

import (
	"database/sql"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
)

type Video struct {
	ID   string
	Tags string
}

type Log struct {
	VideoID  string
	At       time.Time
	View     int32
	Comment  int32
	Mylist   int32
	TagsDiff string
}

func diffTags(oldSet, newSet map[string]struct{}) []string {
	r := make([]string, 0)
	for k, _ := range newSet {
		if _, ok := oldSet[k]; !ok {
			r = append(r, "+"+k)
		}
	}
	for k, _ := range oldSet {
		if _, ok := newSet[k]; !ok {
			r = append(r, "-"+k)
		}
	}
	return r
}

func newDbMap(driver, dsn string, dialect modl.Dialect) (*modl.DbMap, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	dbmap := modl.NewDbMap(db, dialect)
	dbmap.AddTable(Video{}).SetKeys(false, "id")
	dbmap.AddTable(Log{}).SetKeys(false, "videoid", "at")

	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		return nil, err
	}
	return dbmap, nil
}

func write(tx *modl.Transaction, vi *NicoVideoInfo) error {
	var v Video
	err := tx.Get(&v, vi.VideoID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	ots := NewNicoVideoInfoTagSlice(v.Tags)
	nts := vi.TagsByDomain("jp")
	v.ID = vi.VideoID
	v.Tags = nts.String()
	if err == nil {
		_, err = tx.Update(&v)
	} else {
		err = tx.Insert(&v)
	}
	if err != nil {
		return err
	}

	l := Log{
		VideoID:  vi.VideoID,
		At:       time.Now(),
		View:     int32(vi.ViewCounter),
		Comment:  int32(vi.CommentNum),
		Mylist:   int32(vi.MylistCounter),
		TagsDiff: strings.Join(diffTags(ots.StringSet(), nts.StringSet()), "\n"),
	}

	err = tx.Insert(&l)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	dbFile := flag.String("db", "nicoch.sqlite3", "database filename")
	mylistID := flag.Int64("id", 0, "nicovideo mylist id")

	flag.Parse()
	dbmap, err := newDbMap("sqlite3", *dbFile, modl.SqliteDialect{})
	if err != nil {
		log.Fatal(err)
	}

	ml, err := GetNicoVideoMylist(*mylistID)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range ml.Item {
		time.Sleep(3e9)

		vi, err := GetNicoVideoInfo(item.ExtractVideoID())
		if err != nil {
			log.Println(err)
			continue
		}

		tx, err := dbmap.Begin()
		if err != nil {
			log.Println(err)
			continue
		}

		err = write(tx, vi)
		if err != nil {
			log.Println(err)
			err = tx.Rollback()
			if err != nil {
				log.Println(err)
			}
			continue
		}

		err = tx.Commit()
		if err != nil {
			log.Println(err)
		}
	}
}
