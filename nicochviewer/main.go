package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/oov/nicoch/db"

	"github.com/jmoiron/modl"
	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
)

var tpl = template.Must(template.New("").ParseGlob("*.html"))
var port = flag.String("http", ":80", "http listen address")
var dbFile = flag.String("db", "nicoch.sqlite3", "database filename")

func useDB(h func(dbm *modl.DbMap, w http.ResponseWriter, r *http.Request, p httprouter.Params)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		dbm, err := db.New("sqlite3", *dbFile, modl.SqliteDialect{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h(dbm, w, r, p)
		err = dbm.Dbx.Close()
		if err != nil {
			log.Println("err:", err)
		}
	}
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

func Video(dbm *modl.DbMap, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var vars struct {
		Video db.Video
		Logs  []*db.Log
	}
	var err error
	vars.Video, err = db.GetVideoByCode(dbm, params.ByName("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars.Logs, err = LogDaily(dbm, vars.Video.ID, time.Now())
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

func Index(dbm *modl.DbMap, w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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

	router := httprouter.New()
	router.GET("/", useDB(Index))
	router.GET("/:code", useDB(Video))

	log.Fatal(http.ListenAndServe(*port, router))
}
