package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// Cluster link states of broker
func (semp *Semp) GetClusterLinksSemp1(ch chan<- PrometheusMetric, clusterFilter string, linkFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Cluster struct {
					Clusters struct {
						Cluster []struct {
							ClusterName string `xml:"cluster-name"`
							NodeName    string `xml:"node-name"`
							Links       struct {
								Link []struct {
									Enabled           string  `xml:"enabled"`
									Operational       string  `xml:"oper-up"`
									UptimeInSeconds   float64 `xml:"oper-uptime-seconds"`
									RemoteClusterName string  `xml:"remote-cluster-name"`
									RemoteNodeName    string  `xml:"remote-node-name"`
								} `xml:"link"`
							} `xml:"links"`
						} `xml:"cluster"`
					} `xml:"clusters"`
				} `xml:"cluster"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><cluster><cluster-name-pattern>" + clusterFilter + "</cluster-name-pattern><link-name-pattern>" + linkFilter + "</link-name-pattern></cluster></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "ClusterLinksSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, cluster := range target.RPC.Show.Cluster.Clusters.Cluster {
		for _, link := range cluster.Links.Link {
			ch <- semp.NewMetric(MetricDesc["ClusterLinks"]["enabled"], prometheus.GaugeValue, encodeMetricMulti(link.Enabled, []string{"false", "true", "n/a"}), cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
			ch <- semp.NewMetric(MetricDesc["ClusterLinks"]["oper_up"], prometheus.GaugeValue, encodeMetricMulti(link.Operational, []string{"false", "true", "n/a"}), cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
			ch <- semp.NewMetric(MetricDesc["ClusterLinks"]["oper_uptime"], prometheus.GaugeValue, link.UptimeInSeconds, cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
		}
	}

	return 1, nil
}
