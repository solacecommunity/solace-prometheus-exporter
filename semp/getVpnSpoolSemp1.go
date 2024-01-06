package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Replication Config and status
func (e *Semp) GetVpnSpoolSemp1(ch chan<- PrometheusMetric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageSpool struct {
					MessageVpn struct {
						Vpn []struct {
							Name                string  `xml:"name"`
							SpooledMsgCount     float64 `xml:"current-messages-spooled"`
							SpoolUsageCurrentMb float64 `xml:"current-spool-usage-mb"`
							SpoolUsageMaxMb     float64 `xml:"maximum-spool-usage-mb"`
							CurrentEndpoints    float64 `xml:"current-queues-and-topic-endpoints"`
							MaximumEndpoints    float64 `xml:"maximum-queues-and-topic-endpoints"`
							CurrentEgressFlows  float64 `xml:"current-egress-flows"`
							MaximumEgressFlows  float64 `xml:"maximum-egress-flows"`
							CurrentIngressFlows float64 `xml:"current-ingress-flows"`
							MaximumIngressFlows float64 `xml:"maximum-ingress-flows"`
						} `xml:"vpn"`
					} `xml:"message-vpn"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool><vpn-name>" + vpnFilter + "</vpn-name><detail/></message-spool></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "VpnSpoolSemp1", 1)
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

	for _, vpn := range target.RPC.Show.MessageSpool.MessageVpn.Vpn {
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_quota_bytes"], prometheus.GaugeValue, vpn.SpoolUsageMaxMb*1024*1024, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_usage_bytes"], prometheus.GaugeValue, vpn.SpoolUsageCurrentMb*1024*1024, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_usage_msgs"], prometheus.GaugeValue, vpn.SpooledMsgCount, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_current_endpoints"], prometheus.GaugeValue, vpn.CurrentEndpoints, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_maximum_endpoints"], prometheus.GaugeValue, vpn.MaximumEndpoints, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_current_egress_flows"], prometheus.GaugeValue, vpn.CurrentEgressFlows, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_maximum_egress_flows"], prometheus.GaugeValue, vpn.MaximumEgressFlows, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_current_ingress_flows"], prometheus.GaugeValue, vpn.CurrentIngressFlows, vpn.Name)
		ch <- e.NewMetric(MetricDesc["VpnSpool"]["vpn_spool_maximum_ingress_flows"], prometheus.GaugeValue, vpn.MaximumIngressFlows, vpn.Name)
	}

	return 1, nil
}
