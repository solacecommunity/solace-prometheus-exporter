package semp

import (
	"encoding/xml"
	"errors"
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
					Vpn []struct {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

    var page = 1
    var lastVpnName = ""
	for command := fmt.Sprintf("<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><count/><num-elements>%d</num-elements></message-vpn></show></rpc>", sempPageSize); command != ""; {
        body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "VpnSemp1", page)
        page++

        if err != nil {
            semp.logger.Error("Can't scrape VpnSemp1", "err", err, "broker", semp.brokerURI)
            return -1, err
        }
        defer func() { _ = body.Close() }()
        decoder := xml.NewDecoder(body)
        var target Data
        err = decoder.Decode(&target)
        if err != nil {
            semp.logger.Error("Can't decode Xml VpnSemp1", "err", err, "broker", semp.brokerURI)
            _ = body.Close()
            return 0, err
        }

        semp.logger.Debug("Result of VpnSemp1", "results", len(target.RPC.Show.MessageVpn.Vpn), "page", page-1)
        command = target.MoreCookie.RPC

        if target.ExecuteResult.Result != "ok" {
            semp.logger.Error("Unexpected result for VpnSemp1", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
            return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
        }

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
        }
		_ = body.Close()
    }

	return 1, nil
}
