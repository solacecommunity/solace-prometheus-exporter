package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetInterfaceSemp1 Get interface information
func (semp *Semp) GetInterfaceSemp1(ch chan<- PrometheusMetric, interfaceFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Interface struct {
					Interfaces struct {
						Interface []struct {
							Name    string `xml:"phy-interface"`
							Enabled string `xml:"enabled"`
							State   string `xml:"oper-status"`
							Stats   struct {
								RxBytes float64 `xml:"rx-bytes"`
								TxBytes float64 `xml:"tx-bytes"`
							} `xml:"stats"`
						} `xml:"interface"`
					} `xml:"interfaces"`
				} `xml:"interface"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><interface><phy-interface>" + interfaceFilter + "</phy-interface></interface></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "InterfaceSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape InterfaceSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml InterfaceSemp1", "err", err, "broker", semp.brokerURI)
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

	for _, intf := range target.RPC.Show.Interface.Interfaces.Interface {
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_rx_bytes"], prometheus.CounterValue, intf.Stats.RxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_tx_bytes"], prometheus.CounterValue, intf.Stats.TxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_state"], prometheus.GaugeValue, encodeMetricMulti(intf.State, []string{"Down", "Up"}), intf.Name)
	}

	return 1, nil
}
