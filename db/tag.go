package db

import (
	"database/sql"

	"github.com/jmoiron/modl"
)

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

func (t Tags) StringSlice() []string {
	ss := make([]string, len(t))
	for i, v := range t {
		ss[i] = v.Name
	}
	return ss
}
