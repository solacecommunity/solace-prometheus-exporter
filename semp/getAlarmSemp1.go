package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get system Alarm information
func (e *Semp) GetAlarmSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><alarm/></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "AlarmSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape AlarmSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml AlarmSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}
	var alarmsExist = false
	// Check if an alarm is present
	if len(target.RPC.Show.Alarm.Alarms.Alarm) != 0 {
		alarmsExist = true
	}
	ch <- e.NewMetric(MetricDesc["Alarm"]["system_alarm"], prometheus.GaugeValue, encodeMetricBool(alarmsExist))

	return 1, nil
}
