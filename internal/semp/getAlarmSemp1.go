package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetAlarmSemp1 Get system Alarm information.
func (semp *Semp) GetAlarmSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Alarm struct {
					Alarms struct {
						Alarm []struct { // we don't need to parse out the values as we are just testing if an alarm is present
						} `xml:"alarm"`
					} `xml:"alarms"`
				} `xml:"alarm"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><alarm/></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "AlarmSemp1", 1, &target); err != nil {
		return up, err
	}
	var alarmsExist = false
	// Check if an alarm is present
	if len(target.RPC.Show.Alarm.Alarms.Alarm) != 0 {
		alarmsExist = true
	}
	ch <- semp.NewMetric(MetricDesc["Alarm"]["system_alarm"], prometheus.GaugeValue, encodeMetricBool(alarmsExist))

	return 1, nil
}
