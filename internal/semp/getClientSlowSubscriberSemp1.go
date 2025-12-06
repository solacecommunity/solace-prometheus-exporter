package semp

import (
	"encoding/xml"
	"errors"
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
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><slow-subscriber/></client></show></rpc>"; nextRequest != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", nextRequest, "ClientSlowSubscriberSemp1", page)
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
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
			return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
		}

		nextRequest = target.MoreCookie.RPC
		const slowSubscriber float64 = 1.0

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIP := strings.Split(client.ClientAddress, ":")[0]
			ch <- semp.NewMetric(MetricDesc["ClientSlowSubscriber"]["client_slow_subscriber"], prometheus.GaugeValue, slowSubscriber, client.MsgVpnName, client.ClientName, clientIP, "")
		}
		body.Close()
	}

	return 1, nil
}
