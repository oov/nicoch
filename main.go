package main

import (
	"database/sql"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/oov/nicoch/db"

	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
)

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

func write(tx *modl.Transaction, vi *NicoVideoInfo) error {
	var v db.Video
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

	l := db.Log{
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
	dbmap, err := db.New("sqlite3", *dbFile, modl.SqliteDialect{})
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
