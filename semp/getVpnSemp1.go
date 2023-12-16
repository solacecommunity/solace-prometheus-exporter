package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get info of all vpn's
func (e *Semp) GetVpnSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
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
						ConnectionsAmqService          float64 `xml:"connections-service-amqp"`
						ConnectionsSmfService          float64 `xml:"connections-service-smf"`
						Connections                    float64 `xml:"connections"`
						QuotaConnections               float64 `xml:"max-connections"`
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "VpnSemp1", 1)
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
		_ = level.Error(e.logger).Log("msg", "Unexpected result for VpnSemp1", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_is_management_vpn"], prometheus.GaugeValue, encodeMetricBool(vpn.IsManagementMessageVpn), vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_enabled"], prometheus.GaugeValue, encodeMetricBool(vpn.Enabled), vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_operational"], prometheus.GaugeValue, encodeMetricBool(vpn.Operational), vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_locally_configured"], prometheus.GaugeValue, encodeMetricBool(vpn.LocallyConfigured), vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_unique_subscriptions"], prometheus.GaugeValue, vpn.UniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_total_local_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalLocalUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_total_remote_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalRemoteUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_total_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_connections_service_amqp"], prometheus.GaugeValue, vpn.ConnectionsAmqService, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_connections_service_smf"], prometheus.GaugeValue, vpn.ConnectionsSmfService, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_connections"], prometheus.GaugeValue, vpn.Connections, vpn.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Vpn"]["vpn_quota_connections"], prometheus.GaugeValue, vpn.QuotaConnections, vpn.Name)
	}

	return 1, nil
}
