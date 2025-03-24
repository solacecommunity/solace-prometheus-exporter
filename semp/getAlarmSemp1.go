package semp

import (
	"encoding/xml"
	"errors"
	"io"

	"github.com/go-kit/log/level"
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><alarm/></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "AlarmSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape AlarmSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Error closing body", "err", err)
		}
	}(body)
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml AlarmSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
	}
	var alarmsExist = false
	// Check if an alarm is present
	if len(target.RPC.Show.Alarm.Alarms.Alarm) != 0 {
		alarmsExist = true
	}
	ch <- semp.NewMetric(MetricDesc["Alarm"]["system_alarm"], prometheus.GaugeValue, encodeMetricBool(alarmsExist))

	return 1, nil
}
