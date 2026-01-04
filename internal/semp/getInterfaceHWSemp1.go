package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetInterfaceHWSemp1 Get interface information
func (semp *Semp) GetInterfaceHWSemp1(ch chan<- PrometheusMetric, interfaceFilter string) (float64, error) {
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
								RxPackets float64 `xml:"rx-pkts"`
								RxBytes   float64 `xml:"rx-bytes"`
								TxPackets float64 `xml:"tx-pkts"`
								TxBytes   float64 `xml:"tx-bytes"`
							} `xml:"stats"`
							LAG struct {
								ConfiguredMembers struct {
									Member []struct {
									} `xml:"member"`
								} `xml:"configured-members"`
								AvailableMembers struct {
									Member []struct {
									} `xml:"member"`
								} `xml:"available-members"`
								OperationalMembers struct {
									Member []struct {
									} `xml:"member"`
								} `xml:"operational-members"`
							} `xml:"lag" `
							ETH struct {
								LinkDetected string `xml:"link-detected" optional:"true" omitifempty:"true"`
							} `xml:"eth"`
						} `xml:"interface"`
					} `xml:"interfaces"`
				} `xml:"interface"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><interface>"
	// * is an invalid HW interface filter, instead no filter shows all interfaces
	if interfaceFilter != "*" {
		command += "<phy-interface>" + interfaceFilter + "</phy-interface>"
	}
	command += "</interface></show></rpc>"

	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "InterfaceHWSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape InterfaceHWSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml InterfaceHWSemp1", "err", err, "broker", semp.brokerURI)
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
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_rx_bytes"], prometheus.CounterValue, intf.Stats.RxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_tx_bytes"], prometheus.CounterValue, intf.Stats.TxBytes, intf.Name)
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_rx_packets"], prometheus.CounterValue, intf.Stats.RxPackets, intf.Name)
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_tx_packets"], prometheus.CounterValue, intf.Stats.TxPackets, intf.Name)
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_state"], prometheus.GaugeValue, encodeMetricMulti(intf.State, []string{"Down", "Up"}), intf.Name)
		ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_enabled"], prometheus.GaugeValue, encodeMetricMulti(intf.Enabled, []string{"No", "Yes"}), intf.Name)
		if intf.LAG.ConfiguredMembers.Member != nil {
			ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_lag_configured_members"], prometheus.GaugeValue, float64(len(intf.LAG.ConfiguredMembers.Member)), intf.Name)
			ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_lag_available_members"], prometheus.GaugeValue, float64(len(intf.LAG.AvailableMembers.Member)), intf.Name)
			ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_lag_operational_members"], prometheus.GaugeValue, float64(len(intf.LAG.OperationalMembers.Member)), intf.Name)
		} else if len(intf.ETH.LinkDetected) > 0 {
			ch <- semp.NewMetric(MetricDesc["InterfaceHW"]["network_ifhw_link_detected"], prometheus.GaugeValue, encodeMetricMulti(intf.ETH.LinkDetected, []string{"No", "Yes"}), intf.Name)
		}
	}

	return 1, nil
}
