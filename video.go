package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type NicoVideoInfo struct {
	VideoID       string    `xml:"thumb>video_id"`
	Title         string    `xml:"thumb>title"`
	Description   string    `xml:"thumb>description"`
	FirstRetrieve time.Time `xml:"thumb>first_retrieve"`
	Length        string    `xml:"thumb>length"`
	MovieType     string    `xml:"thumb>movie_type"`
	ViewCounter   int64     `xml:"thumb>view_counter"`
	CommentNum    int64     `xml:"thumb>comment_num"`
	MylistCounter int64     `xml:"thumb>mylist_counter"`
	Tags          []struct {
		Domain string                `xml:"domain,attr"`
		Tag    NicoVideoInfoTagSlice `xml:"tag"`
	} `xml:"thumb>tags"`
}

type NicoVideoInfoTagSlice []struct {
	Category bool   `xml:"category,attr,omitempty"`
	Lock     bool   `xml:"lock,attr,omitempty"`
	Value    string `xml:",chardata"`
}

func (ts NicoVideoInfoTagSlice) Len() int           { return len(ts) }
func (ts NicoVideoInfoTagSlice) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
func (ts NicoVideoInfoTagSlice) Less(i, j int) bool { return ts[i].Value < ts[j].Value }

func (ts NicoVideoInfoTagSlice) StringSlice() []string {
	tags := make([]string, len(ts))
	for i, v := range ts {
		tags[i] = v.Value
	}
	return tags
}

func (ts NicoVideoInfoTagSlice) StringSet() map[string]struct{} {
	tags := map[string]struct{}{}
	for _, v := range ts {
		tags[v.Value] = struct{}{}
	}
	return tags
}

func (ts NicoVideoInfoTagSlice) String() string {
	return strings.Join(ts.StringSlice(), "\n")
}

func NewNicoVideoInfoTagSlice(s string) NicoVideoInfoTagSlice {
	ss := strings.Split(strings.TrimSpace(s), "\n")
	if len(ss) == 1 && ss[0] == "" {
		return make(NicoVideoInfoTagSlice, 0)
	}
	sort.Strings(ss)
	tags := make(NicoVideoInfoTagSlice, len(ss))
	for i, v := range ss {
		tags[i].Value = v
	}
	return tags
}

func (info NicoVideoInfo) TagsByDomain(domain string) NicoVideoInfoTagSlice {
	for i := 0; i < len(info.Tags); i++ {
		if info.Tags[i].Domain == domain {
			return info.Tags[i].Tag
		}
	}
	return make(NicoVideoInfoTagSlice, 0)
}

func GetNicoVideoInfo(id string) (*NicoVideoInfo, error) {
	url := fmt.Sprintf(`http://ext.nicovideo.jp/api/getthumbinfo/%s`, id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return getNicoVideoInfoFromReader(resp.Body)
}

func getNicoVideoInfoFromReader(r io.Reader) (*NicoVideoInfo, error) {
	var vi NicoVideoInfo
	err := xml.NewDecoder(r).Decode(&vi)
	if err != nil {
		return nil, err
	}

	for _, v := range vi.Tags {
		sort.Sort(v.Tag)
	}
	return &vi, nil
}
