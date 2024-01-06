package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get statistics of all vpn's
func (e *Semp) GetVpnStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Vpn []struct {
						Name        string  `xml:"name"`
						LocalStatus string  `xml:"local-status"`
						Connections float64 `xml:"connections"`
						Stats       struct {
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
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><stats/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "VpnStatsSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_rx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataRxMsgCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_tx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataTxMsgCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_rx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataRxByteCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_tx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataTxByteCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_rx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.IngressDiscards.DiscardedRxMsgCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnStats"]["vpn_tx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.EgressDiscards.DiscardedTxMsgCount, vpn.Name)
	}

	return 1, nil
}
