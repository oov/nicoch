package db

import (
	"database/sql"
	"math"
	"time"

	"github.com/jmoiron/modl"
)

type Video struct {
	ID       int64
	Code     string
	Name     string
	PostedAt time.Time
	Thumb    string
}

type Log struct {
	ID      int64
	VideoID int64
	At      time.Time
	View    int64
	Comment int64
	Mylist  int64
}

type Tag struct {
	ID   int64
	Name string
}

type Tags []Tag

type VideoTag struct {
	VideoID int64
	TagID   int64
}

type LogTag struct {
	LogID int64
	TagID int64
	Score int64
}

func New(driver, dsn string, dialect modl.Dialect) (*modl.DbMap, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, err
	}

	dbmap := modl.NewDbMap(db, dialect)
	dbmap.AddTable(Video{}).SetKeys(true, "id")
	dbmap.AddTable(Log{}).SetKeys(true, "id")
	dbmap.AddTable(Tag{}).SetKeys(true, "id")
	dbmap.AddTable(VideoTag{}).SetKeys(false, "videoid", "tagid")
	dbmap.AddTable(LogTag{}).SetKeys(false, "logid", "tagid")
	return dbmap, nil
}

func GetTag(x modl.SqlExecutor, name string) (Tag, error) {
	var t Tag
	err := x.SelectOne(&t, `SELECT * FROM tag WHERE name = ?`, name)
	switch {
	case err == nil:
		return t, nil
	case err == sql.ErrNoRows:
		t.Name = name
		err = x.Insert(&t)
		return t, err
	default:
		return t, err
	}
}

func GetVideoByCode(x modl.SqlExecutor, code string) (Video, error) {
	var v Video
	err := x.SelectOne(&v, `SELECT * FROM video WHERE code = ?`, code)
	return v, err
}

func (v *Video) Tags(x modl.SqlExecutor) (Tags, error) {
	var tags Tags
	err := x.Select(&tags, `SELECT tag.* FROM tag INNER JOIN videotag ON (videotag.tagid = tag.id)AND(videoid = ?)`, v.ID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func (v *Video) RemoveAllTags(x modl.SqlExecutor) error {
	_, err := x.Exec(`DELETE FROM videotag WHERE videotag.videoid = ?`, v.ID)
	return err
}

func (v *Video) UpdateTags(x modl.SqlExecutor, tags []string) error {
	err := v.RemoveAllTags(x)
	if err != nil {
		return err
	}

	for _, tag := range tags {
		t, err := GetTag(x, tag)
		if err != nil {
			return err
		}

		err = x.Insert(&VideoTag{
			VideoID: v.ID,
			TagID:   t.ID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Point() float64 {
	if l.View == 0 {
		return 0
	}

	v, c, m := float64(l.View), float64(l.Comment), float64(l.Mylist)
	corrA := (v + m) / (v + c + m)
	corrB := math.Min((m/(v*100))*2, 40)
	return v + c*corrA + m*corrB
}

func (l *Log) RemoveAllTags(x modl.SqlExecutor) error {
	_, err := x.Exec(`DELETE FROM logtag WHERE logtag.logid = ?`, l.ID)
	return err
}

func (l *Log) UpdateTags(x modl.SqlExecutor, added, removed []string) error {
	err := l.RemoveAllTags(x)
	if err != nil {
		return err
	}

	for _, tag := range added {
		t, err := GetTag(x, tag)
		if err != nil {
			return err
		}

		err = x.Insert(&LogTag{
			LogID: l.ID,
			TagID: t.ID,
			Score: 1,
		})
		if err != nil {
			return err
		}
	}

	for _, tag := range removed {
		t, err := GetTag(x, tag)
		if err != nil {
			return err
		}

		err = x.Insert(&LogTag{
			LogID: l.ID,
			TagID: t.ID,
			Score: -1,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t Tags) StringSlice() []string {
	ss := make([]string, len(t))
	for i, v := range t {
		ss[i] = v.Name
	}
	return ss
}
