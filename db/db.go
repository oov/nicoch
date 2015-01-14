package db

import (
	"database/sql"
	"time"

	"github.com/jmoiron/modl"
)

type Video struct {
	ID   string
	Tags string
}

type Log struct {
	VideoID  string
	At       time.Time
	View     int32
	Comment  int32
	Mylist   int32
	TagsDiff string
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

	dbmap := modl.NewDbMap(db, dialect)
	dbmap.AddTable(Video{}).SetKeys(false, "id")
	dbmap.AddTable(Log{}).SetKeys(false, "videoid", "at")

	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		return nil, err
	}
	return dbmap, nil
}
