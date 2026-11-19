package semp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---- Disk ----

const diskReply = `<rpc-reply><rpc><show><disk><disk-infos>` +
	`<disk-info><mounted-on>/usr/sw/jail</mounted-on><file-system>/dev/sda1</file-system>` +
	`<blocks>1000</blocks><used>250</used><available>750</available><use>25%</use></disk-info>` +
	`<disk-info><mounted-on>/var</mounted-on><file-system>/dev/sda2</file-system>` +
	`<blocks>2000</blocks><used>1800</used><available>200</available><use>90%</use></disk-info>` +
	`</disk-infos></disk></show></rpc><execute-result code="ok"/></rpc-reply>`

// TestGetDiskSemp1EmitsPerMountMetrics pins the 1 KiB block-to-byte conversion (metric names end in _bytes), the
// stripping of the "%" suffix and the path/device_name label order, which dashboards depend on.
func TestGetDiskSemp1EmitsPerMountMetrics(t *testing.T) {
	t.Parallel()
	s := newSempReplyingWith(t, diskReply, true)

	ch := make(chan PrometheusMetric, 50)
	up, err := s.GetDiskSemp1(ch)
	metrics := drain(ch)

	if err != nil || up != 1 {
		t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
	}
	want := map[string]float64{
		`solace_system_disk_used_percent{path="/usr/sw/jail",device_name="/dev/sda1"}`: 25,
		`solace_system_disk_used_bytes{path="/usr/sw/jail",device_name="/dev/sda1"}`:   250 * 1024,
		`solace_system_disk_avail_bytes{path="/usr/sw/jail",device_name="/dev/sda1"}`:  750 * 1024,
		`solace_system_disk_used_percent{path="/var",device_name="/dev/sda2"}`:         90,
		`solace_system_disk_used_bytes{path="/var",device_name="/dev/sda2"}`:           1800 * 1024,
		`solace_system_disk_avail_bytes{path="/var",device_name="/dev/sda2"}`:          200 * 1024,
	}
	got := make(map[string]float64, len(metrics))
	for _, m := range metrics {
		got[m.Name()] = m.value
	}
	if len(got) != len(want) {
		t.Errorf("emitted %d metrics, want %d: %v", len(got), len(want), got)
	}
	for name, w := range want {
		if g, ok := got[name]; !ok || g != w {
			t.Errorf("%s = %v (present=%v), want %v", name, g, ok, w)
		}
	}
}

// TestGetDiskSemp1NoDisksEmitsNothing: an empty disk list is a successful scrape without series.
func TestGetDiskSemp1NoDisksEmitsNothing(t *testing.T) {
	t.Parallel()
	s := newSempReplyingWith(t, `<rpc-reply><rpc><show><disk><disk-infos/></disk></show></rpc><execute-result code="ok"/></rpc-reply>`, true)

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetDiskSemp1(ch)
	metrics := drain(ch)

	if err != nil || up != 1 {
		t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
	}
	if len(metrics) != 0 {
		t.Errorf("emitted %d metrics, want 0", len(metrics))
	}
}

// TestGetDiskSemp1UnparsablePercentFallsBackToZero documents the lenient parse: a malformed "use" value must not
// fail the scrape; the percent gauge reads 0 while the byte gauges stay correct.
func TestGetDiskSemp1UnparsablePercentFallsBackToZero(t *testing.T) {
	t.Parallel()
	reply := `<rpc-reply><rpc><show><disk><disk-infos><disk-info><mounted-on>/</mounted-on>` +
		`<file-system>dev</file-system><blocks>10</blocks><used>4</used><available>6</available>` +
		`<use>n/a</use></disk-info></disk-infos></disk></show></rpc><execute-result code="ok"/></rpc-reply>`
	s := newSempReplyingWith(t, reply, true)

	ch := make(chan PrometheusMetric, 10)
	up, err := s.GetDiskSemp1(ch)
	values := metricValuesByFQName(drain(ch))

	if err != nil || up != 1 {
		t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
	}
	if v := values["solace_system_disk_used_percent"]; v != 0 {
		t.Errorf("used_percent = %v, want 0", v)
	}
	if v := values["solace_system_disk_used_bytes"]; v != 4*1024 {
		t.Errorf("used_bytes = %v, want %v", v, 4*1024)
	}
}

// ---- Alarm ----

func alarmReply(alarms string) string {
	return `<rpc-reply><rpc><show><alarm><alarms>` + alarms +
		`</alarms></alarm></show></rpc><execute-result code="ok"/></rpc-reply>`
}

// TestGetAlarmSemp1ReportsPresence pins the 0/1 encoding of "any alarm present", which alert rules rely on.
func TestGetAlarmSemp1ReportsPresence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		alarms string
		want   float64
	}{
		{"no alarms", ``, 0},
		{"one alarm", `<alarm><name>a</name></alarm>`, 1},
		{"several alarms", `<alarm/><alarm/><alarm/>`, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newSempReplyingWith(t, alarmReply(tc.alarms), false)

			ch := make(chan PrometheusMetric, 5)
			up, err := s.GetAlarmSemp1(ch)
			values := metricValuesByFQName(drain(ch))

			if err != nil || up != 1 {
				t.Fatalf("up=%v err=%v, want up=1 and no error", up, err)
			}
			if len(values) != 1 {
				t.Fatalf("emitted %d metrics, want 1: %v", len(values), values)
			}
			if got := values["solace_system_alarm"]; got != tc.want {
				t.Errorf("solace_system_alarm = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDiskAndAlarmFailurePaths covers the three error contracts shared by both getters: a not-ok result and a
// malformed body are recoverable (up=0), an unreachable broker is unrecoverable (up=-1); none may emit metrics.
func TestDiskAndAlarmFailurePaths(t *testing.T) {
	t.Parallel()

	getters := map[string]func(*Semp, chan<- PrometheusMetric) (float64, error){
		"disk":  (*Semp).GetDiskSemp1,
		"alarm": (*Semp).GetAlarmSemp1,
	}
	for name, get := range getters {
		t.Run(name+"/not ok result", func(t *testing.T) {
			t.Parallel()
			s := newSempReplyingWith(t, `<rpc-reply><rpc/><execute-result code="fail" reason="nope"/></rpc-reply>`, true)
			ch := make(chan PrometheusMetric, 5)
			up, err := get(s, ch)
			if err == nil || up != 0 || len(drain(ch)) != 0 {
				t.Errorf("up=%v err=%v, want up=0, an error and no metrics", up, err)
			}
		})
		t.Run(name+"/malformed xml", func(t *testing.T) {
			t.Parallel()
			s := newSempReplyingWith(t, `<rpc-reply><rpc>`, true)
			ch := make(chan PrometheusMetric, 5)
			up, err := get(s, ch)
			if err == nil || up != 0 || len(drain(ch)) != 0 {
				t.Errorf("up=%v err=%v, want up=0, an error and no metrics", up, err)
			}
		})
		t.Run(name+"/broker unreachable", func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.NotFoundHandler())
			s := newSempReplyingWith(t, "", true)
			s.brokerURI = server.URL
			server.Close() // nothing listens on the address any more
			ch := make(chan PrometheusMetric, 5)
			up, err := get(s, ch)
			if err == nil || up != -1 || len(drain(ch)) != 0 {
				t.Errorf("up=%v err=%v, want up=-1, an error and no metrics", up, err)
			}
		})
	}
}
