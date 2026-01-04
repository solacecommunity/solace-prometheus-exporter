package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetConfigSyncSemp1 Sync Status for Broker and Vpn
func (semp *Semp) GetConfigSyncSemp1(ch chan<- PrometheusMetric) (float64, error) {
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><config-sync></config-sync></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "ConfigSyncSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		_ = level.Error(semp.logger).Log(
			"msg", "unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

	ch <- semp.NewMetric(MetricDesc["ConfigSync"]["configsync_admin_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.AdminStatus, []string{"Shutdown", "Enabled"}))
	ch <- semp.NewMetric(MetricDesc["ConfigSync"]["configsync_oper_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.OperStatus, []string{"Down", "Up", "Shutting Down"}))

	return 1, nil
}
