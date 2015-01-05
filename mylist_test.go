package main

import "testing"

func TestExtractVideoID(t *testing.T) {
	var item NicoVideoMylistItem
	if item.ExtractVideoID() != "" {
		t.Fail()
	}
	item.Link = "invalid"
	if item.ExtractVideoID() != "invalid" {
		t.Fail()
	}
	item.Link = "http://www.nicovideo.jp/watch/sm9"
	if item.ExtractVideoID() != "sm9" {
		t.Fail()
	}
}
