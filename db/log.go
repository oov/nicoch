package db

import (
	"math"
	"time"

	"github.com/jmoiron/modl"
)

type Log struct {
	ID      int64
	VideoID int64
	At      time.Time
	View    int64
	Comment int64
	Mylist  int64
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
