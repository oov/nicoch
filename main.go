package main

import (
	"database/sql"
	"flag"
	"log"
	"time"

	"github.com/oov/nicoch/db"

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

func getAndUpdateVideo(x modl.SqlExecutor, vi *NicoVideoInfo, tweetedAt time.Time) (db.Video, error) {
	v, err := db.GetVideoByCode(x, vi.VideoID)
	if err != nil && err != sql.ErrNoRows {
		return v, err
	}

	v.Code = vi.VideoID
	v.Name = vi.Title
	v.PostedAt = vi.FirstRetrieve
	if tweetedAt.After(v.TweetedAt) {
		v.TweetedAt = tweetedAt
	}
	v.Thumb = vi.Thumbnail
	if err == sql.ErrNoRows {
		err = x.Insert(&v)
	} else {
		_, err = x.Update(&v)
	}
	return v, err
}

func write(x modl.SqlExecutor, vi *NicoVideoInfo, tweetedAt time.Time) error {
	v, err := getAndUpdateVideo(x, vi, tweetedAt)
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

	l := db.Log{
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
	dbFile := flag.String("db", "nicoch.sqlite3", "database filename")
	mylistID := flag.Int64("id", 0, "nicovideo mylist id")
	twitterConsumerKey := flag.String("consumer-key", "", "Twiter Application Consumer Key")
	twitterConsumerSecret := flag.String("consumer-secret", "", "Twiter Application Consumer Secret")
	twitterOAuthToken := flag.String("token", "", "Twiter OAuth Token")
	twitterOAuthSecret := flag.String("secret", "", "Twiter OAuth Secret")
	flag.Parse()
	dbmap, err := db.New("sqlite3", *dbFile, modl.SqliteDialect{})
	if err != nil {
		log.Fatal(err)
	}

	defer dbmap.Dbx.Close()

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

		tweetedAt, err := SearchLatestTweet(
			vi.VideoID,
			*twitterConsumerKey, *twitterConsumerSecret,
			*twitterOAuthToken, *twitterOAuthSecret,
		)
		if err != nil {
			log.Println(err)
			continue
		}

		tx, err := dbmap.Begin()
		if err != nil {
			log.Println(err)
			continue
		}

		err = write(tx, vi, tweetedAt)
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
