package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"math"
)

// Get some statistics for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topic endpoints
func (e *Semp) GetTopicEndpointDetailsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
								MsgVpnName      string  `xml:"message-vpn"`
								Quota           float64 `xml:"quota"`
								Usage           float64 `xml:"current-spool-usage-in-mb"`
								SpooledMsgCount float64 `xml:"num-messages-spooled"`
								BindCount       float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointDetailsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointDetails"]["spool_quota_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Quota*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointDetails"]["spool_usage_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Usage*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointDetails"]["spool_usage_msgs"], prometheus.GaugeValue, topicEndpoint.Info.SpooledMsgCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointDetails"]["binds"], prometheus.GaugeValue, topicEndpoint.Info.BindCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}
