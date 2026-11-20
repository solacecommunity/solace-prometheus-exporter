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

// targetRegistration declares a scrape target under one or more data source names.
type targetRegistration struct {
	names  []string
	target scrapeTarget
}

// scrapeTargets maps every supported data source name to its scrape target.
var scrapeTargets = buildScrapeTargets([]targetRegistration{
	target("Version", anyBroker, noFilter((*semp.Semp).GetVersionSemp1)),
	target("Health", softwareBrokerOnly, noFilter((*semp.Semp).GetHealthSemp1)),
	target("StorageElement", softwareBrokerOnly, byItem((*semp.Semp).GetStorageElementSemp1)),
	target("Disk", hardwareBrokerOnly, noFilter((*semp.Semp).GetDiskSemp1)),
	target("Raid", hardwareBrokerOnly, noFilter((*semp.Semp).GetRaidSemp1)),
	target("Memory", anyBroker, noFilter((*semp.Semp).GetMemorySemp1)),
	target("Interface", anyBroker, byItem((*semp.Semp).GetInterfaceSemp1)),
	target("InterfaceHW", hardwareBrokerOnly, byItem((*semp.Semp).GetInterfaceHWSemp1)),
	target("GlobalStats", anyBroker, noFilter((*semp.Semp).GetGlobalStatsSemp1)),
	target("GlobalSystemInfo", anyBroker, noFilter((*semp.Semp).GetGlobalSystemInfoSemp1)),
	target("Spool", anyBroker, noFilter((*semp.Semp).GetSpoolSemp1)),
	target("SpoolStats", anyBroker, noFilter((*semp.Semp).GetSpoolStatsSemp1)),
	target("Redundancy", anyBroker, noFilter((*semp.Semp).GetRedundancySemp1)),
	target("Alarm", hardwareBrokerOnly, noFilter((*semp.Semp).GetAlarmSemp1)),
	target("Environment", hardwareBrokerOnly, noFilter((*semp.Semp).GetEnvironmentSemp1)),
	target("Hardware", hardwareBrokerOnly, noFilter((*semp.Semp).GetHardwareSemp1)),
	target("ClockDetail", hardwareBrokerOnly, noFilter((*semp.Semp).GetClockDetailSemp1)),
	target("ReplicationStats", anyBroker, noFilter((*semp.Semp).GetReplicationStatsSemp1)),
	target("ConfigSyncRouter", anyBroker, noFilter((*semp.Semp).GetConfigSyncRouterSemp1)),
	target("ConfigSync", anyBroker, noFilter((*semp.Semp).GetConfigSyncSemp1)),
	target("Vpn", anyBroker, byVpnPaged((*semp.Semp).GetVpnSemp1)),
	target("VpnReplication", anyBroker, byVpn((*semp.Semp).GetVpnReplicationSemp1)),
	target("ConfigSyncVpn", anyBroker, byVpnPaged((*semp.Semp).GetConfigSyncVpnSemp1)),
	target("Bridge", anyBroker, byVpnAndItemPaged((*semp.Semp).GetBridgeSemp1)),
	target("BridgeRemote", anyBroker, byVpnAndItem((*semp.Semp).GetBridgeRemoteSemp1)),
	target("BridgeDetail", anyBroker, byVpnAndItemPaged((*semp.Semp).GetBridgeDetailSemp1)),
	target("BridgeClientCert", anyBroker, byVpnAndItemPaged((*semp.Semp).GetBridgeClientCertSemp1)),
	target("VpnSpool", anyBroker, byVpnPaged((*semp.Semp).GetVpnSpoolSemp1)),
	target("Client", anyBroker, byVpnAndItem((*semp.Semp).GetClientSemp1)),
	target("ClientProfile", anyBroker, byVpn((*semp.Semp).GetClientProfileSemp1)),
	target("ClientSlowSubscriber", anyBroker, byVpnAndItem((*semp.Semp).GetClientSlowSubscriberSemp1)),
	target("ClientStats", anyBroker, byItemPaged((*semp.Semp).GetClientStatsSemp1)),
	target("ClientConnections", anyBroker, byItem((*semp.Semp).GetClientConnectionStatsSemp1)),
	target("ClientMessageSpoolStats", anyBroker, scrapeClientMessageSpoolStats),
	target("ClientMessageSpoolEgress", anyBroker, byItem((*semp.Semp).GetClientMessageSpoolEgressSemp1)),
	target("ClusterLinks", anyBroker, byVpnAndItem((*semp.Semp).GetClusterLinksSemp1)),
	target("VpnStats", anyBroker, byVpnPaged((*semp.Semp).GetVpnStatsSemp1)),
	target("BridgeStats", anyBroker, byVpnAndItemPaged((*semp.Semp).GetBridgeStatsSemp1)),
	target("QueueRates", anyBroker, byVpnAndItemPaged((*semp.Semp).GetQueueRatesSemp1)),
	target("QueueStats", anyBroker, byVpnAndItemPaged((*semp.Semp).GetQueueStatsSemp1)),
	targetWithoutV1Alias("QueueStatsV2", anyBroker, scrapeQueueStatsV2),
	target("QueueDetails", anyBroker, byVpnAndItemPaged((*semp.Semp).GetQueueDetailsSemp1)),
	target("TopicEndpointRates", anyBroker, byVpnAndItemPaged((*semp.Semp).GetTopicEndpointRatesSemp1)),
	target("TopicEndpointStats", anyBroker, byVpnAndItemPaged((*semp.Semp).GetTopicEndpointStatsSemp1)),
	target("TopicEndpointDetails", anyBroker, byVpnAndItemPaged((*semp.Semp).GetTopicEndpointDetailsSemp1)),
	target("RestConsumerStats", anyBroker, byVpnAndItemPaged((*semp.Semp).GetRestConsumerStatsSemp1)),
	target("RdpStats", anyBroker, byVpnAndItemPaged((*semp.Semp).GetRdpStatsSemp1)),
	target("RdpInfo", anyBroker, byVpnAndItem((*semp.Semp).GetRdpInfoSemp1)),
	targetWithoutV1Alias("MqttSession", anyBroker, byVpnAndItemPaged((*semp.Semp).GetMqttSessionSemp1)),
	target("StatsClientDetail", anyBroker, noFilter((*semp.Semp).GetStatsClientDetailSemp1)),
})

// target registers a scrape target under its name and the legacy "<name>V1" alias.
func target(name string, kind brokerKind, scrape scrapeFunc) targetRegistration {
	return targetRegistration{names: []string{name, name + "V1"}, target: scrapeTarget{kind, scrape}}
}

// targetWithoutV1Alias registers a scrape target under its name only.
func targetWithoutV1Alias(name string, kind brokerKind, scrape scrapeFunc) targetRegistration {
	return targetRegistration{names: []string{name}, target: scrapeTarget{kind, scrape}}
}

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

// Adapters from the SEMP getter signatures to scrapeFunc. They are named after the data source filters they pass;
// "Paged" getters additionally get the configured SEMP page size.

type metricChan = chan<- semp.PrometheusMetric

func noFilter(get func(*semp.Semp, metricChan) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, _ DataSource) (float64, error) {
		return get(e.semp, ch)
	}
}

func byItem(get func(*semp.Semp, metricChan, string) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.ItemFilter)
	}
}

func byVpn(get func(*semp.Semp, metricChan, string) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.VpnFilter)
	}
}

func byVpnAndItem(get func(*semp.Semp, metricChan, string, string) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.VpnFilter, ds.ItemFilter)
	}
}

func byItemPaged(get func(*semp.Semp, metricChan, string, int64) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.ItemFilter, e.config.SempPageSize)
	}
}

func byVpnPaged(get func(*semp.Semp, metricChan, string, int64) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.VpnFilter, e.config.SempPageSize)
	}
}

func byVpnAndItemPaged(get func(*semp.Semp, metricChan, string, string, int64) (float64, error)) scrapeFunc {
	return func(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
		return get(e.semp, ch, ds.VpnFilter, ds.ItemFilter, e.config.SempPageSize)
	}
}

// scrapeClientMessageSpoolStats passes the VPN filter as client name filter, although docs/CONFIG.md documents only
// the item filter as supported. Kept unchanged so existing scrape configurations keep working.
func scrapeClientMessageSpoolStats(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
	return e.semp.GetClientMessageSpoolStatsSemp1(ch, ds.VpnFilter, e.config.SempPageSize)
}

// scrapeQueueStatsV2 resolves the concrete VPN SEMP v2 needs (a "*" filter falls back to defaultVpn).
func scrapeQueueStatsV2(e *Exporter, ch metricChan, ds DataSource) (float64, error) {
	vpnName, err := e.getVpnName(ds.VpnFilter)
	if err != nil {
		return 0, err
	}
	return e.semp.GetQueueStatsSemp2(ch, vpnName, ds.ItemFilter, ds.MetricFilter)
}
