package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"
	"strings"

	"github.com/go-kit/log/level"
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var page = 1
	for command := "<rpc><show><client><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><slow-subscriber/></client></show></rpc>"; command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "ClientSlowSubscriberSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape ClientSlowSubscriberSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}

		defer body.Close()

		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode ClientSlowSubscriberSemp1", "err", err, "broker", semp.brokerURI)
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

		command = target.MoreCookie.RPC
		const slowSubscriber float64 = 1.0

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIP := strings.Split(client.ClientAddress, ":")[0]
			ch <- semp.NewMetric(MetricDesc["ClientSlowSubscriber"]["client_slow_subscriber"], prometheus.GaugeValue, slowSubscriber, client.MsgVpnName, client.ClientName, clientIP, "")
		}
		body.Close()
	}

	return 1, nil
}
