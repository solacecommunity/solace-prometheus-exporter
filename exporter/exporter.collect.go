package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"solace_exporter/semp"
)

// Collect fetches the stats from configured Solace location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	var up float64 = 1
	var err error = nil

	for _, dataSource := range e.config.DataSource {
		if up < 1 {
			if err != nil {
				ch <- prometheus.MustNewConstMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 0, err.Error())
			} else {
				ch <- prometheus.MustNewConstMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 0, "Unknown")
			}
			return
		}

		switch dataSource.Name {
		case "Version":
			up, err = e.semp.GetVersionSemp1(ch)
		case "Health":
			up, err = e.semp.GetHealthSemp1(ch)
		case "StorageElement":
			up, err = e.semp.GetStorageElementSemp1(ch, dataSource.ItemFilter)
		case "Disk":
			up, err = e.semp.GetDiskSemp1(ch)
		case "Memory":
			up, err = e.semp.GetMemorySemp1(ch)
		case "Interface":
			up, err = e.semp.GetInterfaceSemp1(ch, dataSource.ItemFilter)
		case "GlobalStats":
			up, err = e.semp.GetGlobalStatsSemp1(ch)
		case "Spool":
			up, err = e.semp.GetSpoolSemp1(ch)
		case "Redundancy":
			up, err = e.semp.GetRedundancySemp1(ch)
		case "ReplicationStats":
			up, err = e.semp.GetReplicationStatsSemp1(ch)
		case "ConfigSyncRouter":
			up, err = e.semp.GetConfigSyncRouterSemp1(ch)
		case "Vpn":
			up, err = e.semp.GetVpnSemp1(ch, dataSource.VpnFilter)
		case "VpnReplication":
			up, err = e.semp.GetVpnReplicationSemp1(ch, dataSource.VpnFilter)
		case "ConfigSyncVpn":
			up, err = e.semp.GetConfigSyncVpnSemp1(ch, dataSource.VpnFilter)
		case "Bridge":
			up, err = e.semp.GetBridgeSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "VpnSpool":
			up, err = e.semp.GetVpnSpoolSemp1(ch, dataSource.VpnFilter)
		case "Client":
			up, err = e.semp.GetClientSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "ClientSlowSubscriber":
			up, err = e.semp.GetClientSlowSubscriberSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "ClientStats":
			up, err = e.semp.GetClientStatsSemp1(ch, dataSource.VpnFilter)
		case "ClientMessageSpoolStats":
			up, err = e.semp.GetClientMessageSpoolStatsSemp1(ch, dataSource.VpnFilter)
		case "ClusterLinks":
			up, err = e.semp.GetClusterLinksSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)

		case "VpnStats":
			up, err = e.semp.GetVpnStatsSemp1(ch, dataSource.VpnFilter)
		case "BridgeStats":
			up, err = e.semp.GetBridgeStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueRates":
			up, err = e.semp.GetQueueRatesSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueStats":
			up, err = e.semp.GetQueueStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueDetails":
			up, err = e.semp.GetQueueDetailsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointRates":
			up, err = e.semp.GetTopicEndpointRatesSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointStats":
			up, err = e.semp.GetTopicEndpointStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointDetails":
			up, err = e.semp.GetTopicEndpointDetailsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		}
	}
	ch <- prometheus.MustNewConstMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 1, "")
}
