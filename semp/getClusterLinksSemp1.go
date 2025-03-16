package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Cluster link states of broker
func (semp *Semp) GetClusterLinksSemp1(ch chan<- PrometheusMetric, clusterFilter string, linkFilter string) (ok float64, err error) {
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><cluster><cluster-name-pattern>" + clusterFilter + "</cluster-name-pattern><link-name-pattern>" + linkFilter + "</link-name-pattern></cluster></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "ClusterLinksSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape ClusterLinksSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml ClusterLinksSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
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
