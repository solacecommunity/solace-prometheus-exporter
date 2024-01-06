package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Config Sync Status for Broker and Vpn
func (e *Semp) GetConfigSyncRouterSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Database struct {
						Local struct {
							Tables struct {
								Table []struct {
									Type               string  `xml:"type"`
									TimeInStateSeconds float64 `xml:"time-in-state-seconds"`
									Name               string  `xml:"name"`
									Ownership          string  `xml:"ownership"`
									SyncState          string  `xml:"sync-state"`
									TimeInState        string  `xml:"time-in-state"`
								} `xml:"table"`
							} `xml:"tables"`
						} `xml:"local"`
					} `xml:"database"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><config-sync><database/><router/></config-sync></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "ConfigSyncRouterSemp1", 1)
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

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
	}

	return 1, nil
}
