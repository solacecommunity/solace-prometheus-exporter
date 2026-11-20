package semp

import (
	"solace_exporter/internal/semp/types"

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
		types.Reply
	}

	command := "<rpc><show><interface><phy-interface>" + semp1Escape(interfaceFilter) + "</phy-interface></interface></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "InterfaceSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, intf := range target.RPC.Show.Interface.Interfaces.Interface {
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_rx_bytes"], prometheus.CounterValue, intf.Stats.RxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_tx_bytes"], prometheus.CounterValue, intf.Stats.TxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["Interface"]["network_if_state"], prometheus.GaugeValue, encodeMetricMulti(intf.State, []string{"Down", "Up"}), intf.Name)
	}

	return 1, nil
}
