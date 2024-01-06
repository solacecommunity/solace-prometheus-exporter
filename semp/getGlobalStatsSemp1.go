package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get global stats information
func (e *Semp) GetGlobalStatsSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Stats struct {
					Client struct {
						Global struct {
							Stats struct {
								ClientsConnected float64 `xml:"total-clients-connected"`
								DataRxMsgCount   float64 `xml:"client-data-messages-received"`
								DataTxMsgCount   float64 `xml:"client-data-messages-sent"`
								DataRxByteCount  float64 `xml:"client-data-bytes-received"`
								DataTxByteCount  float64 `xml:"client-data-bytes-sent"`
								RxMsgsRate       float64 `xml:"current-ingress-rate-per-second"`
								TxMsgsRate       float64 `xml:"current-egress-rate-per-second"`
								RxBytesRate      float64 `xml:"current-ingress-byte-rate-per-second"`
								TxBytesRate      float64 `xml:"current-egress-byte-rate-per-second"`
								IngressDiscards  struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"global"`
					} `xml:"client"`
				} `xml:"stats"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><stats><client/></stats></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "GlobalStatsSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape GlobalStatsSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml GlobalStatsSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_total_clients_connected"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.ClientsConnected)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_rx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxMsgCount)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_tx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxMsgCount)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_rx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxByteCount)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_tx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxByteCount)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_total_rx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.IngressDiscards.DiscardedRxMsgCount)
	ch <- e.NewMetric(MetricDesc["GlobalStats"]["system_total_tx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.EgressDiscards.DiscardedTxMsgCount)

	return 1, nil
}
