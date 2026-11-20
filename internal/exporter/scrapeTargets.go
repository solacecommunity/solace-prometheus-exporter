package exporter

import (
	"solace_exporter/internal/semp"
)

// brokerKind restricts a scrape target to the broker type that supports it.
type brokerKind int

const (
	anyBroker brokerKind = iota
	softwareBrokerOnly
	hardwareBrokerOnly
)

// scrapeFunc collects the metrics of one scrape target. It follows the getter return contract: up >= 1 success,
// up == 0 recoverable failure of this target, up < 0 unrecoverable failure that aborts the remaining targets.
type scrapeFunc func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error)

type scrapeTarget struct {
	brokerKind brokerKind
	scrape     scrapeFunc
}

// targetRegistration declares a scrape target under one or more names (the bare name and its legacy "V1" alias).
type targetRegistration struct {
	names  []string
	target scrapeTarget
}

// scrapeTargets maps every supported data source name to its scrape target.
var scrapeTargets = buildScrapeTargets([]targetRegistration{
	{[]string{"Version", "VersionV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetVersionSemp1(ch)
	}}},
	{[]string{"Health", "HealthV1"}, scrapeTarget{softwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetHealthSemp1(ch)
	}}},
	{[]string{"StorageElement", "StorageElementV1"}, scrapeTarget{softwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetStorageElementSemp1(ch, ds.ItemFilter)
	}}},
	{[]string{"Disk", "DiskV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetDiskSemp1(ch)
	}}},
	{[]string{"Raid", "RaidV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetRaidSemp1(ch)
	}}},
	{[]string{"Memory", "MemoryV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetMemorySemp1(ch)
	}}},
	{[]string{"Interface", "InterfaceV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetInterfaceSemp1(ch, ds.ItemFilter)
	}}},
	{[]string{"InterfaceHW", "InterfaceHWV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetInterfaceHWSemp1(ch, ds.ItemFilter)
	}}},
	{[]string{"GlobalStats", "GlobalStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetGlobalStatsSemp1(ch)
	}}},
	{[]string{"GlobalSystemInfo", "GlobalSystemInfoV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetGlobalSystemInfoSemp1(ch)
	}}},
	{[]string{"Spool", "SpoolV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetSpoolSemp1(ch)
	}}},
	{[]string{"SpoolStats", "SpoolStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetSpoolStatsSemp1(ch)
	}}},
	{[]string{"Redundancy", "RedundancyV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetRedundancySemp1(ch)
	}}},
	{[]string{"Alarm", "AlarmV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetAlarmSemp1(ch)
	}}},
	{[]string{"Environment", "EnvironmentV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetEnvironmentSemp1(ch)
	}}},
	{[]string{"Hardware", "HardwareV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetHardwareSemp1(ch)
	}}},
	{[]string{"ClockDetail", "ClockDetailV1"}, scrapeTarget{hardwareBrokerOnly, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetClockDetailSemp1(ch)
	}}},
	{[]string{"ReplicationStats", "ReplicationStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetReplicationStatsSemp1(ch)
	}}},
	{[]string{"ConfigSyncRouter", "ConfigSyncRouterV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetConfigSyncRouterSemp1(ch)
	}}},
	{[]string{"ConfigSync", "ConfigSyncV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetConfigSyncSemp1(ch)
	}}},
	{[]string{"Vpn", "VpnV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetVpnSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
	}}},
	{[]string{"VpnReplication", "VpnReplicationV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetVpnReplicationSemp1(ch, ds.VpnFilter)
	}}},
	{[]string{"ConfigSyncVpn", "ConfigSyncVpnV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetConfigSyncVpnSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
	}}},
	{[]string{"Bridge", "BridgeV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetBridgeSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"BridgeRemote", "BridgeRemoteV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetBridgeRemoteSemp1(ch, ds.VpnFilter, ds.ItemFilter)
	}}},
	{[]string{"BridgeDetail", "BridgeDetailV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetBridgeDetailSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"BridgeClientCert", "BridgeClientCertV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetBridgeClientCertSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"VpnSpool", "VpnSpoolV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetVpnSpoolSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
	}}},
	{[]string{"Client", "ClientV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientSemp1(ch, ds.VpnFilter, ds.ItemFilter)
	}}},
	{[]string{"ClientProfile", "ClientProfileV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientProfileSemp1(ch, ds.VpnFilter)
	}}},
	{[]string{"ClientSlowSubscriber", "ClientSlowSubscriberV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientSlowSubscriberSemp1(ch, ds.VpnFilter, ds.ItemFilter)
	}}},
	{[]string{"ClientStats", "ClientStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientStatsSemp1(ch, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"ClientConnections", "ClientConnectionsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientConnectionStatsSemp1(ch, ds.ItemFilter)
	}}},
	// NOTE: passes the VPN filter as client name filter, although docs/CONFIG.md documents only the item filter as
	// supported. Kept unchanged so existing scrape configurations keep working.
	{[]string{"ClientMessageSpoolStats", "ClientMessageSpoolStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientMessageSpoolStatsSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
	}}},
	{[]string{"ClientMessageSpoolEgress", "ClientMessageSpoolEgressV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClientMessageSpoolEgressSemp1(ch, ds.ItemFilter)
	}}},
	{[]string{"ClusterLinks", "ClusterLinksV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetClusterLinksSemp1(ch, ds.VpnFilter, ds.ItemFilter)
	}}},
	{[]string{"VpnStats", "VpnStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetVpnStatsSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
	}}},
	{[]string{"BridgeStats", "BridgeStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetBridgeStatsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"QueueRates", "QueueRatesV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetQueueRatesSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"QueueStats", "QueueStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetQueueStatsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"QueueStatsV2"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		vpnName, err := e.getVpnName(ds.VpnFilter)
		if err != nil {
			return 0, err
		}
		return e.semp.GetQueueStatsSemp2(ch, vpnName, ds.ItemFilter, ds.MetricFilter)
	}}},
	{[]string{"QueueDetails", "QueueDetailsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetQueueDetailsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"TopicEndpointRates", "TopicEndpointRatesV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetTopicEndpointRatesSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"TopicEndpointStats", "TopicEndpointStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetTopicEndpointStatsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"TopicEndpointDetails", "TopicEndpointDetailsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetTopicEndpointDetailsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"RestConsumerStats", "RestConsumerStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetRestConsumerStatsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"RdpStats", "RdpStatsV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetRdpStatsSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"RdpInfo", "RdpInfoV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetRdpInfoSemp1(ch, ds.VpnFilter, ds.ItemFilter)
	}}},
	{[]string{"MqttSession"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, ds DataSource) (float64, error) {
		return e.semp.GetMqttSessionSemp1(ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}}},
	{[]string{"StatsClientDetail", "StatsClientDetailV1"}, scrapeTarget{anyBroker, func(e *Exporter, ch chan<- semp.PrometheusMetric, _ DataSource) (float64, error) {
		return e.semp.GetStatsClientDetailSemp1(ch)
	}}},
})

// buildScrapeTargets indexes the registrations by name and panics on a duplicate name, which would otherwise
// silently shadow a target.
func buildScrapeTargets(registrations []targetRegistration) map[string]scrapeTarget {
	targets := make(map[string]scrapeTarget)
	for _, r := range registrations {
		for _, name := range r.names {
			if _, exists := targets[name]; exists {
				panic("duplicate scrape target name: " + name)
			}
			targets[name] = r.target
		}
	}
	return targets
}
