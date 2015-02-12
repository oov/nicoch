package main

import (
	"net/url"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

func SearchLatestTweet(q, consumerKey, consumerSecret, token, secretToken string) (time.Time, error) {
	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(token, secretToken)
	sr, err := api.GetSearch(q, url.Values{"count": []string{"1"}, "result_type": []string{"recent"}})
	if err != nil {
		return time.Time{}, err
	}
	if len(sr.Statuses) == 0 {
		return time.Time{}, nil
	}
	return sr.Statuses[0].CreatedAtTime()
}
