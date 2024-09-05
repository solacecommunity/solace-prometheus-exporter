package semp

import (
	"encoding/xml"
	"errors"
	"strings"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get slow subscriber client of vpns
// This can result in heavy system load when lots of clients are connected
func (e *Semp) GetClientSlowSubscriberSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (ok float64, err error) {
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
		} `xml:"execute-result"`
	}

	var page = 1
	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><slow-subscriber/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest, "ClientSlowSubscriberSemp1", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientSlowSubscriberSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientSlowSubscriberSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC
		const slowSubscriber float64 = 1.0

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIp := strings.Split(client.ClientAddress, ":")[0]
			ch <- e.NewMetric(MetricDesc["ClientSlowSubscriber"]["client_slow_subscriber"], prometheus.GaugeValue, slowSubscriber, client.MsgVpnName, client.ClientName, clientIp, "")
		}
		body.Close()
	}

	return 1, nil
}
