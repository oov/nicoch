package db

import (
	"time"

	"github.com/jmoiron/modl"
)

type Video struct {
	ID        int64
	Code      string
	Name      string
	PostedAt  time.Time
	TweetedAt time.Time
	Thumb     string
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
