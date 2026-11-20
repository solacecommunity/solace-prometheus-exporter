package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetVpnSemp1 Get info of all VPNs
func (semp *Semp) GetVpnSemp1(ch chan<- PrometheusMetric, vpnFilter string, sempPageSize int64) (float64, error) {
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
						MaxSubscriptionsEffective      float64 `xml:"max-subscriptions-effective"`
						Connections                    float64 `xml:"connections"`
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastVpnName = ""
	command := fmt.Sprintf("<rpc><show><message-vpn><vpn-name>"+vpnFilter+"</vpn-name><detail/><count/><num-elements>%d</num-elements></message-vpn></show></rpc>", sempPageSize)
	up, err := fetchSemp1Pages(semp, command, "VpnSemp1", func(target *Data) {
		for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
			vpnKey := vpn.Name
			if vpnKey == lastVpnName {
				continue
			}
			lastVpnName = vpnKey
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_is_management_vpn"], prometheus.GaugeValue, encodeMetricBool(vpn.IsManagementMessageVpn), vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_enabled"], prometheus.GaugeValue, encodeMetricBool(vpn.Enabled), vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_operational"], prometheus.GaugeValue, encodeMetricBool(vpn.Operational), vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_locally_configured"], prometheus.GaugeValue, encodeMetricBool(vpn.LocallyConfigured), vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_unique_subscriptions"], prometheus.GaugeValue, vpn.UniqueSubscriptions, vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_local_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalLocalUniqueSubscriptions, vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_remote_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalRemoteUniqueSubscriptions, vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_total_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalUniqueSubscriptions, vpn.Name)
			ch <- semp.NewMetric(MetricDesc["Vpn"]["vpn_max_subscriptions_effective"], prometheus.GaugeValue, vpn.MaxSubscriptionsEffective, vpn.Name)
		}
	})
	return up, withLegacyResultErrorText(err)
}
