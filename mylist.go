package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type NicoVideoMylist struct {
	Title       string                `xml:"channel>title"`
	Description string                `xml:"channel>description"`
	Link        string                `xml:"channel>link"`
	Item        []NicoVideoMylistItem `xml:"channel>item"`
}

type NicoVideoMylistItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

func (item NicoVideoMylistItem) ExtractVideoID() string {
	return item.Link[strings.LastIndex(item.Link, "/")+1:]
}

func GetNicoVideoMylist(id int64) (*NicoVideoMylist, error) {
	if id == 0 {
		return nil, fmt.Errorf("id: %d is invalid", id)
	}

	url := fmt.Sprintf(`http://www.nicovideo.jp/mylist/%d?rss=2.0`, id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return getNicoVideoMylistFromReader(resp.Body)
}

func getNicoVideoMylistFromReader(r io.Reader) (*NicoVideoMylist, error) {
	var ml NicoVideoMylist
	err := xml.NewDecoder(r).Decode(&ml)
	if err != nil {
		return nil, err
	}
	return &ml, nil
}
