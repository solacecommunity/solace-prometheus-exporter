package semp

import (
	"solace_exporter/internal/semp/types"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// GetClientSemp1 Get summary for each client of VPNs
// This can result in heavy system load when lots of clients are connected
func (semp *Semp) GetClientSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientAddress    string  `xml:"client-address"`
							ClientName       string  `xml:"name"`
							MsgVpnName       string  `xml:"message-vpn"`
							NumSubscriptions float64 `xml:"num-subscriptions"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><client><name>" + semp1Escape(itemFilter) + "</name><vpn-name>" + semp1Escape(vpnFilter) + "</vpn-name><connected/></client></show></rpc>"
	return fetchSemp1Pages(semp, command, "ClientSemp1", func(target *Data) {
		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIP := strings.Split(client.ClientAddress, ":")[0]
			ch <- semp.NewMetric(MetricDesc["Client"]["client_num_subscriptions"], prometheus.GaugeValue, client.NumSubscriptions, client.MsgVpnName, client.ClientName, clientIP)
		}
	})
}
