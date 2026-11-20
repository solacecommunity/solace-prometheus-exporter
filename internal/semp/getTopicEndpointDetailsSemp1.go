package semp

import (
	"fmt"
	"math"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetTopicEndpointDetailsSemp1 Get some statistics for each individual topic-endpoint of all VPNs
// This can result in heavy system load for lots of topic endpoints
func (semp *Semp) GetTopicEndpointDetailsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
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
		types.Reply
	}

	var lastTopicEndpointName = ""
	command := fmt.Sprintf("<rpc><show><topic-endpoint><name>"+itemFilter+"</name><vpn-name>"+vpnFilter+"</vpn-name><detail/><count/><num-elements>%d</num-elements></topic-endpoint></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "TopicEndpointDetailsSemp1", func(target *Data) {
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
	})
}
