package db

import (
	"database/sql"

	"github.com/jmoiron/modl"
)

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
