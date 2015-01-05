package main

import "testing"

func TestNicoVideoInfoTagSlice(t *testing.T) {
	if len(NewNicoVideoInfoTagSlice("")) != 0 {
		t.Fail()
	}
	ts := NewNicoVideoInfoTagSlice("test")
	if len(ts) != 1 {
		t.Fail()
	}
	if ts.String() != "test" {
		t.Fail()
	}
}
