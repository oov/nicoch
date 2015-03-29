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
	"github.com/oov/nicoch/db/tag"
)

var tpl = template.Must(template.New("").ParseGlob("*.html"))
var dbFile = flag.String("db", "nicoch.sqlite3", "database filename")

type key int

const dbKey key = 0

func useDB(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		dbm, err := db.New("sqlite3", "file:"+*dbFile+"?_loc=auto", modl.SqliteDialect{})
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

func Video(c web.C, w http.ResponseWriter, r *http.Request) {
	dbm := getDB(c)
	var vars struct {
		Video      db.Video
		Logs       []*db.Log
		TagChanges []tag.Change
	}
	var err error
	vars.Video, err = db.GetVideoByCode(dbm, c.URLParams["code"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	vars.Logs, err = db.LogDaily(dbm, vars.Video.ID, now.AddDate(0, -1, 0), now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vars.TagChanges, err = tag.ChangeLogs(dbm, vars.Video.ID, now.AddDate(-1, 0, 0), now)
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
