package exporter

import (
	"errors"
	"solace_exporter/semp"
	"strings"
	"sync"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// CollectPrometheusMetric fetches the stats from configured Solace location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) CollectPrometheusMetric(ch chan<- semp.PrometheusMetric) {
	var up float64 = 1
	var err error
	var vpnName string

	for _, dataSource := range *e.dataSource {
		switch dataSource.Name {
		case "Version", "VersionV1":
			up, err = e.semp.GetVersionSemp1(ch)
		case "Health", "HealthV1":
			if !e.config.IsHWBroker {
				up, err = e.semp.GetHealthSemp1(ch)
			} else {
				up = 0
				err = errors.New("Software only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Software only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "StorageElement", "StorageElementV1":
			if !e.config.IsHWBroker {
				up, err = e.semp.GetStorageElementSemp1(ch, dataSource.ItemFilter)
			} else {
				up = 0
				err = errors.New("Software only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Software only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "Disk", "DiskV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetDiskSemp1(ch)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "Raid", "RaidV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetRaidSemp1(ch)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "Memory", "MemoryV1":
			up, err = e.semp.GetMemorySemp1(ch)
		case "Interface", "InterfaceV1":
			up, err = e.semp.GetInterfaceSemp1(ch, dataSource.ItemFilter)
		case "InterfaceHW", "InterfaceHWV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetInterfaceHWSemp1(ch, dataSource.ItemFilter)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "GlobalStats", "GlobalStatsV1":
			up, err = e.semp.GetGlobalStatsSemp1(ch)
		case "GlobalSystemInfo", "GlobalSystemInfoV1":
			up, err = e.semp.GetGlobalSystemInfoSemp1(ch)
		case "Spool", "SpoolV1":
			up, err = e.semp.GetSpoolSemp1(ch)
		case "Redundancy", "RedundancyV1":
			up, err = e.semp.GetRedundancySemp1(ch)
		case "Alarm", "AlarmV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetAlarmSemp1(ch)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "Environment", "EnvironmentV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetEnvironmentSemp1(ch)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "Hardware", "HardwareV1":
			if e.config.IsHWBroker {
				up, err = e.semp.GetHardwareSemp1(ch)
			} else {
				up = 0
				err = errors.New("Hardware only scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
				_ = level.Error(e.logger).Log("Hardware only  scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			}
		case "ReplicationStats", "ReplicationStatsV1":
			up, err = e.semp.GetReplicationStatsSemp1(ch)
		case "ConfigSyncRouter", "ConfigSyncRouterV1":
			up, err = e.semp.GetConfigSyncRouterSemp1(ch)
		case "ConfigSync", "ConfigSyncV1":
			up, err = e.semp.GetConfigSyncSemp1(ch)
		case "Vpn", "VpnV1":
			up, err = e.semp.GetVpnSemp1(ch, dataSource.VpnFilter)
		case "VpnReplication", "VpnReplicationV1":
			up, err = e.semp.GetVpnReplicationSemp1(ch, dataSource.VpnFilter)
		case "ConfigSyncVpn", "ConfigSyncVpnV1":
			up, err = e.semp.GetConfigSyncVpnSemp1(ch, dataSource.VpnFilter)
		case "Bridge", "BridgeV1":
			up, err = e.semp.GetBridgeSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "BridgeRemote", "BridgeRemoteV1":
			up, err = e.semp.GetBridgeRemoteSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "VpnSpool", "VpnSpoolV1":
			up, err = e.semp.GetVpnSpoolSemp1(ch, dataSource.VpnFilter)
		case "Client", "ClientV1":
			up, err = e.semp.GetClientSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "ClientProfile", "ClientProfileV1":
			up, err = e.semp.GetClientProfileSemp1(ch, dataSource.VpnFilter)
		case "ClientSlowSubscriber", "ClientSlowSubscriberV1":
			up, err = e.semp.GetClientSlowSubscriberSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "ClientStats", "ClientStatsV1":
			up, err = e.semp.GetClientStatsSemp1(ch, dataSource.ItemFilter)
		case "ClientConnections", "ClientConnectionsV1":
			up, err = e.semp.GetClientConnectionStatsSemp1(ch, dataSource.ItemFilter)
		case "ClientMessageSpoolStats", "ClientMessageSpoolStatsV1":
			up, err = e.semp.GetClientMessageSpoolStatsSemp1(ch, dataSource.VpnFilter)
		case "ClusterLinks", "ClusterLinksV1":
			up, err = e.semp.GetClusterLinksSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "VpnStats", "VpnStatsV1":
			up, err = e.semp.GetVpnStatsSemp1(ch, dataSource.VpnFilter)
		case "BridgeStats", "BridgeStatsV1":
			up, err = e.semp.GetBridgeStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueRates", "QueueRatesV1":
			up, err = e.semp.GetQueueRatesSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueStats", "QueueStatsV1":
			up, err = e.semp.GetQueueStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "QueueStatsV2":
			vpnName, err = e.getVpnName(dataSource.VpnFilter)
			if err == nil {
				up, err = e.semp.GetQueueStatsSemp2(ch, vpnName, dataSource.ItemFilter, dataSource.MetricFilter)
			}
		case "QueueDetails", "QueueDetailsV1":
			up, err = e.semp.GetQueueDetailsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointRates", "TopicEndpointRatesV1":
			up, err = e.semp.GetTopicEndpointRatesSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointStats", "TopicEndpointStatsV1":
			up, err = e.semp.GetTopicEndpointStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "TopicEndpointDetails", "TopicEndpointDetailsV1":
			up, err = e.semp.GetTopicEndpointDetailsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "RestConsumerStats", "RestConsumerStatsV1":
			up, err = e.semp.GetRestConsumerStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "RdpStats", "RdpStatsV1":
			up, err = e.semp.GetRdpStatsSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		case "RdpInfo", "RdpInfoV1":
			up, err = e.semp.GetRdpInfoSemp1(ch, dataSource.VpnFilter, dataSource.ItemFilter)
		default:
			up = 0
			err = errors.New("Unknown scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
			_ = level.Error(e.logger).Log("Unknown scrape target: \"" + dataSource.Name + "\". Please check documentation for valid targets.")
		}

		var endpoint = dataSource.Name
		if up < 1 {
			if up < 0 {
				endpoint = "global"
			}

			if err != nil {
				ch <- e.semp.NewMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 0, err.Error(), endpoint)
			} else {
				ch <- e.semp.NewMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 0, "Unknown", endpoint)
			}

			if up < 0 {
				// Unrecoverable error that will be repeated on all dataSources
				break
			}
		} else {
			ch <- e.semp.NewMetric(semp.MetricDesc["Global"]["up"], prometheus.GaugeValue, 1, "", endpoint)
		}
	}
}

func (e *Exporter) Collect(pch chan<- prometheus.Metric) {
	var ch = make(chan semp.PrometheusMetric, capMetricChan)
	var wg sync.WaitGroup

	wg.Add(1)

	collectWorker := func() {
		e.CollectPrometheusMetric(ch)
		wg.Done()
	}
	go collectWorker()

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Ensure `checkedMetricChan` and `uncheckedMetricChan` are drained in case of an early return.
	defer func() {
		if ch != nil {
			for range ch {
			}
		}
	}()

	// read from chanel until the channel is closed
	var distinctMetrics = make(map[string]semp.PrometheusMetric)

	for {
		metric, ok := <-ch
		if !ok {
			for _, metric := range distinctMetrics {
				pch <- metric.AsPrometheusMetric()
			}

			return
		}
		// using a map to filter duplicates and use always most current received value
		distinctMetrics[metric.Name()] = metric
	}
}

func (e *Exporter) getVpnName(vpnFilter string) (string, error) {
	if vpnFilter == "*" {
		if len(strings.TrimSpace(e.config.DefaultVpn)) == 0 {
			return "", errors.New("can't scrape Semp2 As vpnFilter was an * given and the defaultVpn is not set in configuration")
		}
		return e.config.DefaultVpn, nil
	}

	return vpnFilter, nil
}
