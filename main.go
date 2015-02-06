package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"time"

	"github.com/oov/nicoch/db2"

	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
)

func diffTags(oldSet, newSet map[string]struct{}) (added, removed []string) {
	for k, _ := range newSet {
		if _, ok := oldSet[k]; !ok {
			added = append(added, k)
		}
	}
	for k, _ := range oldSet {
		if _, ok := newSet[k]; !ok {
			removed = append(removed, k)
		}
	}
	return
}

func getVideo(x modl.SqlExecutor, vi *NicoVideoInfo) (db2.Video, error) {
	v, err := db2.GetVideoByCode(x, vi.VideoID)
	switch {
	case err == nil:
		v.Code = vi.VideoID
		v.Name = vi.Title
		_, err = x.Update(&v)
		return v, err
	case err != sql.ErrNoRows:
		return v, err
	case err == sql.ErrNoRows:
		v.Code = vi.VideoID
		v.Name = vi.Title
		err = x.Insert(&v)
		return v, err
	}
	panic("unreachable")
}

func write(x modl.SqlExecutor, vi *NicoVideoInfo) error {
	v, err := getVideo(x, vi)
	if err != nil {
		return err
	}

	tags, err := v.Tags(x)
	if err != nil {
		return err
	}

	ots := NewNicoVideoInfoTagSlice(tags.StringSlice())
	nts := vi.TagsByDomain("jp")
	err = v.UpdateTags(x, nts.StringSlice())
	if err != nil {
		return err
	}

	l := db2.Log{
		VideoID: v.ID,
		At:      time.Now(),
		View:    vi.ViewCounter,
		Comment: vi.CommentNum,
		Mylist:  vi.MylistCounter,
	}
	err = x.Insert(&l)
	if err != nil {
		return err
	}

	added, removed := diffTags(ots.StringSet(), nts.StringSet())
	return l.UpdateTags(x, added, removed)
}

func main() {
	dbFile := flag.String("db", "newnicoch.sqlite3", "database filename")
	mylistID := flag.Int64("id", 0, "nicovideo mylist id")

	flag.Parse()
	dbmap, err := db2.New("sqlite3", *dbFile, modl.SqliteDialect{})
	if err != nil {
		log.Fatal(err)
	}
	dbmap.TraceOn("", log.New(os.Stdout, "myapp:", log.Lmicroseconds))

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
