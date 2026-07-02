package main

import (
	"log/slog"
	"net/url"
	"os"
	"sort"
	"testing"
)

// TestParseDataSources covers the /solace `m.<Name>=vpn|item[|metricFilter]` parsing used by the SBB nginx proxy.
func TestParseDataSources(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name string
		form url.Values
		want []string // "Name|Vpn|Item|metricFilterJoinedByComma"
	}{
		{
			name: "single v1 target",
			form: url.Values{"m.Vpn": {"*|*"}},
			want: []string{"Vpn|*|*|"},
		},
		{
			name: "v2 target with metric filter",
			form: url.Values{"m.QueueStats": {"myvpn|myqueue|rx,tx"}},
			want: []string{"QueueStats|myvpn|myqueue|rx,tx"},
		},
		{
			name: "multiple targets and non-m keys ignored",
			form: url.Values{
				"m.Health":    {"*|*"},
				"m.Spool":     {"*|*"},
				"username":    {"ignored"},
				"scrapeURI":   {"http://ignored"},
				"m.QueueRate": {"vpn|item"},
			},
			want: []string{"Health|*|*|", "QueueRate|vpn|item|", "Spool|*|*|"},
		},
		{
			name: "malformed value (no pipe) is skipped",
			form: url.Values{"m.Vpn": {"noPipeHere"}},
			want: nil,
		},
		{
			name: "empty metric filter part is treated as absent",
			form: url.Values{"m.QueueStats": {"vpn|item|  "}},
			want: []string{"QueueStats|vpn|item|"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDataSources(tt.form, logger)
			var gotStr []string
			for _, ds := range got {
				mf := ""
				for i, m := range ds.MetricFilter {
					if i > 0 {
						mf += ","
					}
					mf += m
				}
				gotStr = append(gotStr, ds.Name+"|"+ds.VpnFilter+"|"+ds.ItemFilter+"|"+mf)
			}
			sort.Strings(gotStr)
			sort.Strings(tt.want)

			if len(gotStr) != len(tt.want) {
				t.Fatalf("got %v, want %v", gotStr, tt.want)
			}
			for i := range gotStr {
				if gotStr[i] != tt.want[i] {
					t.Errorf("got[%d]=%q want %q (all got=%v)", i, gotStr[i], tt.want[i], gotStr)
				}
			}
		})
	}
}
