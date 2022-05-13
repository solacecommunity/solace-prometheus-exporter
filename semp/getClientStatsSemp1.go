package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Semp) GetClientStatsSemp1(ch chan<- prometheus.Metric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName     string `xml:"name"`
							ClientUsername string `xml:"client-username"`
							MsgVpnName     string `xml:"message-vpn"`
							SlowSubscriber bool   `xml:"slow-subscriber"`
							Stats          struct {
								DataRxByteCount   float64 `xml:"client-data-bytes-received"`
								DataRxMsgCount    float64 `xml:"client-data-messages-received"`
								DataTxByteCount   float64 `xml:"client-data-bytes-sent"`
								DataTxMsgCount    float64 `xml:"client-data-messages-sent"`
								AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
								AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
								AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
								AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
								RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
								RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
								TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
								TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								IngressDiscards   struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><stats/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientStatSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientStatSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_rx_msgs_total"], prometheus.CounterValue, client.Stats.DataRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_tx_msgs_total"], prometheus.CounterValue, client.Stats.DataTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_rx_bytes_total"], prometheus.CounterValue, client.Stats.DataRxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_tx_bytes_total"], prometheus.CounterValue, client.Stats.DataTxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_rx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.IngressDiscards.DiscardedRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_tx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.EgressDiscards.DiscardedTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientStats"]["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername)
		}
		body.Close()
	}

	return 1, nil
}
