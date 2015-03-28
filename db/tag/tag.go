package tag

import (
	"time"

	"github.com/jmoiron/modl"
)

type Change struct {
	At      time.Time
	Added   []string
	Removed []string
}

type rawChangeLog []struct {
	ID    int64
	At    time.Time
	Name  string
	Score int64
}

func (rcl rawChangeLog) Changes() []Change {
	var cs []Change
	id, idx := int64(-1), -1
	for _, v := range rcl {
		if id != v.ID {
			id, idx, cs = v.ID, len(cs), append(cs, Change{At: v.At})
		}
		if v.Score > 0 {
			cs[idx].Added = append(cs[idx].Added, v.Name)
		} else {
			cs[idx].Removed = append(cs[idx].Removed, v.Name)
		}
	}
	return cs
}

func ChangeLogs(x modl.SqlExecutor, videoID int64, from, to time.Time) ([]Change, error) {
	var t rawChangeLog
	err := x.Select(&t, `
SELECT
 log.id,
 log.at,
 tag.name,
 logtag.score
FROM log
INNER JOIN logtag ON logtag.logid = log.id
INNER JOIN tag ON tag.id = logtag.tagid
WHERE
 (log.videoid = ?)AND
 (? <= log.at)AND
 (log.at < ?)
ORDER BY log.at DESC
`, videoID, from, to)
	if err != nil {
		return nil, err
	}
	return t.Changes(), nil
}
