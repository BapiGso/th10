package main

import (
	"reflect"
	"testing"
)

func TestParseFrameList(t *testing.T) {
	good := []struct {
		in   string
		want []int32
	}{
		{"1200", []int32{1200}},
		{"1200,1300-1302", []int32{1200, 1300, 1301, 1302}},
		{"1302, 1200 ,1300-1301", []int32{1200, 1300, 1301, 1302}},
		{"5-5,5", []int32{5}},
	}
	for _, c := range good {
		got, err := parseFrameList(c.in)
		if err != nil {
			t.Errorf("parseFrameList(%q): %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseFrameList(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	for _, bad := range []string{"", "x", "5-3", "-5", "1,,2", "1-", "99999999999"} {
		if got, err := parseFrameList(bad); err == nil {
			t.Errorf("parseFrameList(%q) = %v, want error", bad, got)
		}
	}
}
