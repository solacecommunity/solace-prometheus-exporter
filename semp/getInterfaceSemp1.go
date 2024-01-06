package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get interface information
func (e *Semp) GetInterfaceSemp1(ch chan<- PrometheusMetric, interfaceFilter string) (ok float64, err error) {
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><interface><phy-interface>" + interfaceFilter + "</phy-interface></interface></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "InterfaceSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape InterfaceSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml InterfaceSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, intf := range target.RPC.Show.Interface.Interfaces.Interface {
		ch <- e.NewMetric(MetricDesc["Interface"]["network_if_rx_bytes"], prometheus.CounterValue, intf.Stats.RxBytes, intf.Name)
		ch <- e.NewMetric(MetricDesc["Interface"]["network_if_tx_bytes"], prometheus.CounterValue, intf.Stats.TxBytes, intf.Name)
		ch <- e.NewMetric(MetricDesc["Interface"]["network_if_state"], prometheus.GaugeValue, encodeMetricMulti(intf.State, []string{"Down", "Up"}), intf.Name)
	}

	return 1, nil
}
