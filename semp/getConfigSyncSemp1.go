package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Config Sync Status for Broker and Vpn
func (e *Semp) GetConfigSyncSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Status struct {
						AdminStatus string `xml:"admin-status"`
						OperStatus  string `xml:"oper-status"`
					} `xml:"status"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><config-sync></config-sync></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "ConfigSyncSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- e.NewMetric(MetricDesc["ConfigSync"]["configsync_admin_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.AdminStatus, []string{"Shutdown", "Enabled"}))
	ch <- e.NewMetric(MetricDesc["ConfigSync"]["configsync_oper_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.OperStatus, []string{"Down", "Up", "Shutting Down"}))

	return 1, nil
}
