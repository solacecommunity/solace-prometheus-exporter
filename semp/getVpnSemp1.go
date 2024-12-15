package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetVpnSemp1 Get info of all VPNs
func (semp *Semp) GetVpnSemp1(ch chan<- PrometheusMetric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					ManagementMessageVpn string `xml:"management-message-vpn"`
					Vpn                  []struct {
						Name                           string  `xml:"name"`
						IsManagementMessageVpn         bool    `xml:"is-management-message-vpn"`
						Enabled                        bool    `xml:"enabled"`
						Operational                    bool    `xml:"operational"`
						LocallyConfigured              bool    `xml:"locally-configured"`
						LocalStatus                    string  `xml:"local-status"`
						UniqueSubscriptions            float64 `xml:"unique-subscriptions"`
						TotalLocalUniqueSubscriptions  float64 `xml:"total-local-unique-subscriptions"`
						TotalRemoteUniqueSubscriptions float64 `xml:"total-remote-unique-subscriptions"`
						TotalUniqueSubscriptions       float64 `xml:"total-unique-subscriptions"`
						Connections                    float64 `xml:"connections"`
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name></message-vpn></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "VpnSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "Unexpected result for VpnSemp1", "command", command, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_is_management_vpn"], prometheus.GaugeValue, encodeMetricBool(vpn.IsManagementMessageVpn), vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_enabled"], prometheus.GaugeValue, encodeMetricBool(vpn.Enabled), vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_operational"], prometheus.GaugeValue, encodeMetricBool(vpn.Operational), vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_locally_configured"], prometheus.GaugeValue, encodeMetricBool(vpn.LocallyConfigured), vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_unique_subscriptions"], prometheus.GaugeValue, vpn.UniqueSubscriptions, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_local_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalLocalUniqueSubscriptions, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_remote_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalRemoteUniqueSubscriptions, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalUniqueSubscriptions, vpn.Name)
	}

	return 1, nil
}
