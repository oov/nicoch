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

func addDateTime(t time.Time, years int, months int, days int, hours int, mins int, secs int, nsecs int) time.Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return time.Date(year+years, month+time.Month(months), day+days, hour+hours, min+mins, sec+secs, t.Nanosecond()+nsecs, t.Location())
}

func LogDaily(x modl.SqlExecutor, videoID int64, from, to time.Time) ([]*Log, error) {
	var logs []Log
	err := x.Select(&logs, `
SELECT
 *
FROM
 log
WHERE
 (videoid = ?)AND
 (strftime('%H', at) = (SELECT strftime('%H', at) FROM log WHERE (videoid = ?)AND(? <= at) ORDER BY at ASC LIMIT 1))AND
 (? <= at)AND(at < ?)
ORDER BY at ASC
`, videoID, videoID, from, from, to)
	if err != nil {
		return nil, err
	}
	var rLogs []*Log
	if len(logs) == 0 {
		return rLogs, nil
	}
	l, r := addDateTime(logs[0].At, 0, 0, 0, 0, -30, 0, 0), addDateTime(logs[0].At, 0, 0, 0, 0, 30, 0, 0)
	for i := 0; i < len(logs); i++ {
		if logs[i].At.Before(l) {
			continue
		}
		if logs[i].At.After(r) {
			rLogs = append(rLogs, &Log{At: addDateTime(l, 0, 0, 0, 0, 30, 0, 0)})
			i--
			continue
		}
		rLogs = append(rLogs, &logs[i])
		l, r = l.AddDate(0, 0, 1), r.AddDate(0, 0, 1)
	}
	return rLogs, nil
}
