package semp

import (
	"encoding/xml"
    "fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetConfigSyncVpnSemp1 Sync Status for Broker and Vpn
func (semp *Semp) GetConfigSyncVpnSemp1(ch chan<- PrometheusMetric, vpnFilter string, sempPageSize int64) (float64, error) {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

    var page = 1
	var lastTableName = ""
	for command := fmt.Sprintf("<rpc><show><config-sync><database/><message-vpn/><vpn-name>" + vpnFilter + "</vpn-name><count/><num-elements>%d</num-elements></config-sync></show></rpc>", sempPageSize); command != ""; {
        body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "ConfigSyncVpnSemp1", page)
        page++

        if err != nil {
            semp.logger.Error("Can't scrape ConfigSyncVpnSemp1", "err", err, "broker", semp.brokerURI)
            return -1, err
        }
        defer func() { _ = body.Close() }()
        decoder := xml.NewDecoder(body)
        var target Data
        err = decoder.Decode(&target)
        if err != nil {
            semp.logger.Error("Can't decode Xml ConfigSyncSemp1", "err", err, "broker", semp.brokerURI)
            _ = body.Close()
            return 0, err
        }
        if err := target.ExecuteResult.OK(); err != nil {
            semp.logger.Error(
                "unexpected result",
                "command", command,
                "result", target.ExecuteResult.Result,
                "reason", target.ExecuteResult.Reason,
                "broker", semp.brokerURI,
            )
            return 0, err
        }

        semp.logger.Debug("Result of ConfigSyncSemp1", "results", len(target.RPC.Show.ConfigSync.Database.Local.Tables.Table), "page", page-1)
        command = target.MoreCookie.RPC

        for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
			tableKey := table.Name
			if tableKey == lastTableName {
				continue
			}
			lastTableName = tableKey
            ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
            ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
            ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
            ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
        }
		_ = body.Close()
    }

	return 1, nil
}
