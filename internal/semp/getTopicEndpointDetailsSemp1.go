package semp

import (
	"encoding/xml"
	"math"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetTopicEndpointDetailsSemp1 Get some statistics for each individual topic-endpoint of all VPNs
// This can result in heavy system load for lots of topic endpoints
func (semp *Semp) GetTopicEndpointDetailsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndpointName string `xml:"name"`
							Info              struct {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var page = 1
	var lastTopicEndpointName = ""
	for command := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "TopicEndpointDetailsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode TopicEndpointDetailsSemp1", "err", err, "broker", semp.brokerURI)
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

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndpointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- semp.NewMetric(MetricDesc["TopicEndpointDetails"]["spool_quota_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Quota*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointDetails"]["spool_usage_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Usage*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointDetails"]["spool_usage_msgs"], prometheus.GaugeValue, topicEndpoint.Info.SpooledMsgCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointDetails"]["binds"], prometheus.GaugeValue, topicEndpoint.Info.BindCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
		}
		body.Close()
	}

	return 1, nil
}
