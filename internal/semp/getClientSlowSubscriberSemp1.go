package semp

import (
	"solace_exporter/internal/semp/types"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// GetClientSlowSubscriberSemp1 Get slow subscriber client of VPNs
// This can result in heavy system load when lots of clients are connected
func (semp *Semp) GetClientSlowSubscriberSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientAddress string `xml:"client-address"`
							ClientProfile string `xml:"profile"`
							ClientName    string `xml:"name"`
							MsgVpnName    string `xml:"message-vpn"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><client><name>" + semp1Escape(itemFilter) + "</name><vpn-name>" + semp1Escape(vpnFilter) + "</vpn-name><slow-subscriber/></client></show></rpc>"
	return fetchSemp1Pages(semp, command, "ClientSlowSubscriberSemp1", func(target *Data) {
		const slowSubscriber float64 = 1.0

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIP := strings.Split(client.ClientAddress, ":")[0]
			ch <- semp.NewMetric(MetricDesc["ClientSlowSubscriber"]["client_slow_subscriber"], prometheus.GaugeValue, slowSubscriber, client.MsgVpnName, client.ClientName, clientIP, "")
		}
	})
}
