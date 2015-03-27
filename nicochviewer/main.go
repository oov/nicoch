package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"

	"github.com/oov/nicoch/db"
)

var tpl = template.Must(template.New("").ParseGlob("*.html"))
var dbFile = flag.String("db", "nicoch.sqlite3", "database filename")

type key int

const dbKey key = 0

func useDB(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		dbm, err := db.New("sqlite3", "file:"+*dbFile+"?loc=auto", modl.SqliteDialect{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c.Env[dbKey] = dbm
		h.ServeHTTP(w, r)
		if err = dbm.Dbx.Close(); err != nil {
			log.Println("err:", err)
		}
	}
	return http.HandlerFunc(fn)
}

func getDB(c web.C) *modl.DbMap {
	v, ok := c.Env[dbKey]
	if !ok {
		return nil
	}
	if dbm, ok := v.(*modl.DbMap); ok {
		return dbm
	}
	return nil
}

func addDateTime(t time.Time, years int, months int, days int, hours int, mins int, secs int, nsecs int) time.Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return time.Date(year+years, month+time.Month(months), day+days, hour+hours, min+mins, sec+secs, t.Nanosecond()+nsecs, t.Location())
}

func LogHourly(dbm *modl.DbMap, videoID string, o time.Time) ([]*db.Log, error) {
	var Logs []db.Log
	err := dbm.Select(&Logs, "SELECT * FROM log WHERE (videoid = ?)AND(datetime(?, '-24 hours') <= at)AND(at < ?) GROUP BY strftime('%Y%m%d%H', at) ORDER BY at ASC", videoID, o, o)
	if err != nil {
		return nil, err
	}
	rLogs := make([]*db.Log, 24)
	var l, r time.Time
	r = addDateTime(o, 0, 0, 0, -24, 0, 0, 0)
	for i, j := 0, 0; i < len(rLogs); i++ {
		l = r
		r = addDateTime(o, 0, 0, 0, -24+i+1, 0, 0, 0)
		if j >= len(Logs) {
			continue
		}
		if Logs[j].At.After(r) {
			continue
		}
		if Logs[j].At.Before(l) {
			j++
			continue
		}
		rLogs[i] = &Logs[j]
		j++
	}
	return rLogs, nil
}

func LogDaily(x modl.SqlExecutor, videoID int64, o time.Time) ([]*db.Log, error) {
	var Logs []db.Log
	err := x.Select(&Logs, "SELECT * FROM log WHERE (videoid = ?)AND(datetime(?, '-30 days') <= at)AND(at < ?) GROUP BY strftime('%Y%m%d', at) ORDER BY at ASC", videoID, o, o)
	if err != nil {
		return nil, err
	}
	rLogs := make([]*db.Log, 30)
	var l, r time.Time
	r = addDateTime(o, 0, 0, -30, 0, 0, 0, 0)
	for i, j := 0, 0; i < len(rLogs); i++ {
		l = r
		r = addDateTime(o, 0, 0, -30+i+1, 0, 0, 0, 0)
		if j >= len(Logs) {
			continue
		}
		if Logs[j].At.After(r) {
			continue
		}
		if Logs[j].At.Before(l) {
			j++
			continue
		}
		rLogs[i] = &Logs[j]
		j++
	}
	return rLogs, nil
}

type tag struct {
	Name  string
	Score int64
}

type tagM struct {
	At   time.Time
	Tags []tag
}

func tagMovement(x modl.SqlExecutor, videoID int64, o time.Time) ([]tagM, error) {
	var tmp []struct {
		ID    int64
		At    time.Time
		Name  string
		Score int64
	}
	err := x.Select(&tmp, "SELECT log.id, log.at, tag.name, logtag.score FROM log INNER JOIN logtag ON logtag.logid = log.id INNER JOIN tag ON tag.id = logtag.tagid WHERE (log.videoid = ?)AND(datetime(?, '-1 years') <= log.at)AND(log.at < ?) ORDER BY log.at DESC", videoID, o, o)
	if err != nil {
		return nil, err
	}

	var tagMs []tagM
	mp := map[int64]int{}
	for _, v := range tmp {
		idx, ok := mp[v.ID]
		if !ok {
			idx = len(tagMs)
			mp[v.ID] = idx
			tagMs = append(tagMs, tagM{At: v.At})
		}
		tagMs[idx].Tags = append(tagMs[idx].Tags, tag{Name: v.Name, Score: v.Score})
	}
	return tagMs, nil
}

func Video(c web.C, w http.ResponseWriter, r *http.Request) {
	dbm := getDB(c)
	var vars struct {
		Video db.Video
		Logs  []*db.Log
		TagMs []tagM
	}
	var err error
	vars.Video, err = db.GetVideoByCode(dbm, c.URLParams["code"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	vars.Logs, err = LogDaily(dbm, vars.Video.ID, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars.TagMs, err = tagMovement(dbm, vars.Video.ID, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tpl.ExecuteTemplate(w, "video.html", vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Index(c web.C, w http.ResponseWriter, r *http.Request) {
	dbm := getDB(c)
	type Stat struct {
		LatestLog   *db.Log
		AWeekAgo    *db.Log
		ViewDiff    int64
		CommentDiff int64
		MylistDiff  int64
		Growth      float64
	}
	var vars struct {
		Videos []db.Video
		Stats  map[int64]*Stat
	}
	err := dbm.Select(&vars.Videos, `SELECT * FROM video`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars.Stats = map[int64]*Stat{}
	for _, v := range vars.Videos {
		vars.Stats[v.ID] = &Stat{}
	}
	var logs []db.Log
	err = dbm.Select(&logs, `SELECT log.* FROM log, (SELECT videoid, MAX(at) AS at FROM log GROUP BY videoid) AS latest WHERE (log.videoid = latest.videoid)AND(log.at = latest.at)`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for k, l := range logs {
		s := vars.Stats[l.VideoID]
		s.LatestLog = &logs[k]
	}
	err = dbm.Select(&logs, `SELECT log.* FROM log, (SELECT videoid, MAX(at) AS at FROM log WHERE at < datetime('now', '-7 days') GROUP BY videoid) AS latest WHERE (log.videoid = latest.videoid)AND(log.at = latest.at)`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for k, l := range logs {
		s := vars.Stats[l.VideoID]
		s.AWeekAgo = &logs[k]
		if s.LatestLog != nil && s.AWeekAgo != nil {
			s.ViewDiff = s.LatestLog.View - s.AWeekAgo.View
			s.CommentDiff = s.LatestLog.Comment - s.AWeekAgo.Comment
			s.MylistDiff = s.LatestLog.Mylist - s.AWeekAgo.Mylist
			if s.AWeekAgo.View != 0 {
				s.Growth = 100.0 * (s.LatestLog.Point()/s.AWeekAgo.Point() - 1)
			}
		}
	}

	err = tpl.ExecuteTemplate(w, "index.html", vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()
	goji.Use(middleware.EnvInit)
	goji.Use(useDB)
	goji.Get("/:code/", Video)
	goji.Get("/", Index)
	goji.Serve()
}
