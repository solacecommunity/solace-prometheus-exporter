package semp

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "solace" // For Prometheus metrics.
)

var (
	variableLabelsUp               = []string{"error"}
	variableLabelsRedundancy       = []string{"mate_name"}
	variableLabelsReplication      = []string{"mate_name"}
	variableLabelsVpn              = []string{"vpn_name"}
	variableLabelsClientInfo       = []string{"vpn_name", "client_name", "client_address"}
	variableLabelsVpnClient        = []string{"vpn_name", "client_name"}
	variableLabelsVpnClientUser    = []string{"vpn_name", "client_name", "client_username"}
	variableLabelsVpnClientDetail  = []string{"vpn_name", "client_name", "client_username", "client_profile", "acl_profile"}
	variableLabelsVpnClientFlow    = []string{"vpn_name", "client_name", "client_username", "client_profile", "acl_profile", "flow_id"}
	variableLabelsVpnQueue         = []string{"vpn_name", "queue_name"}
	variableLabelsVpnTopicEndpoint = []string{"vpn_name", "topic_endpoint_name"}
	variableLabelsCluserLink       = []string{"cluster", "node_name", "remote_cluster", "remote_node_name"}
	variableLabelsBridge           = []string{"vpn_name", "bridge_name"}
	variableLabelsBridgeStats      = []string{"vpn_name", "bridge_name", "remote_router_name", "remote_vpn_name"}
	variableLabelsConfigSyncTable  = []string{"table_name"}
	variableLabelsStorageElement   = []string{"path", "device_name", "element_name"}
	variableLabelsDisk             = []string{"path", "device_name"}
	variableLabelsInterface        = []string{"interface_name"}
)

type Metrics map[string]*prometheus.Desc

var MetricDesc = map[string]Metrics{
	"Global": {
		"up": prometheus.NewDesc(namespace+"_up", "Was the last scrape of Solace broker successful.", variableLabelsUp, nil),
	},
	"Version": {
		"system_version_currentload":      prometheus.NewDesc(namespace+"_"+"system_version_currentload", "Solace Version as WWWXXXYYYZZZ ", nil, nil),
		"system_version_uptime_totalsecs": prometheus.NewDesc(namespace+"_"+"system_version_uptime_totalsecs", "Broker uptime in seconds ", nil, nil),
		"exporter_version_current":        prometheus.NewDesc(namespace+"_"+"exporter_version_current", "Exporter Version as XXXYYYZZZ", nil, nil),
	},
	"Health": {
		"system_disk_latency_min_seconds":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_min_seconds", "Minimum disk latency.", nil, nil),
		"system_disk_latency_max_seconds":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_max_seconds", "Maximum disk latency.", nil, nil),
		"system_disk_latency_avg_seconds":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_avg_seconds", "Average disk latency.", nil, nil),
		"system_disk_latency_cur_seconds":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_cur_seconds", "Current disk latency.", nil, nil),
		"system_compute_latency_min_seconds":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_min_seconds", "Minimum compute latency.", nil, nil),
		"system_compute_latency_max_seconds":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_max_seconds", "Maximum compute latency.", nil, nil),
		"system_compute_latency_avg_seconds":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_avg_seconds", "Average compute latency.", nil, nil),
		"system_compute_latency_cur_seconds":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_cur_seconds", "Current compute latency.", nil, nil),
		"system_mate_link_latency_min_seconds": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_min_seconds", "Minimum mate link latency.", nil, nil),
		"system_mate_link_latency_max_seconds": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_max_seconds", "Maximum mate link latency.", nil, nil),
		"system_mate_link_latency_avg_seconds": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_avg_seconds", "Average mate link latency.", nil, nil),
		"system_mate_link_latency_cur_seconds": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_cur_seconds", "Current mate link latency.", nil, nil),
	},
	//SEMPv1 (Software): show storage element <element-name>
	"StorageElement": {
		"system_storage_used_percent": prometheus.NewDesc(namespace+"_"+"system_storage_used_percent", "Storage Element used percent.", variableLabelsStorageElement, nil),
		"system_storage_used_bytes":   prometheus.NewDesc(namespace+"_"+"system_storage_used_bytes", "Storage Element used bytes.", variableLabelsStorageElement, nil),
		"system_storage_avail_bytes":  prometheus.NewDesc(namespace+"_"+"system_storage_avail_bytes", "Storage Element available bytes.", variableLabelsStorageElement, nil),
	},
	//SEMPv1 (Appliance): show disk detail
	"Disk": {
		"system_disk_used_percent": prometheus.NewDesc(namespace+"_"+"system_disk_used_percent", "Disk used percent.", variableLabelsDisk, nil),
		"system_disk_used_bytes":   prometheus.NewDesc(namespace+"_"+"system_disk_used_bytes", "Disk used bytes.", variableLabelsDisk, nil),
		"system_disk_avail_bytes":  prometheus.NewDesc(namespace+"_"+"system_disk_avail_bytes", "Disk available bytes.", variableLabelsDisk, nil),
	},
	//SEMPv1: show memory
	"Memory": {
		"system_memory_physical_usage_percent":     prometheus.NewDesc(namespace+"_"+"system_memory_physical_usage_percent", "Physical memory usage percent.", nil, nil),
		"system_memory_subscription_usage_percent": prometheus.NewDesc(namespace+"_"+"system_memory_subscription_usage_percent", "Subscription memory usage percent.", nil, nil),
		"system_nab_buffer_load_factor":            prometheus.NewDesc(namespace+"_"+"system_nab_buffer_load_factor", "NAB buffer load factor.", nil, nil),
	},
	//SEMPv1: show interface <interface-name>
	"Interface": {
		"network_if_rx_bytes": prometheus.NewDesc(namespace+"_"+"network_if_rx_bytes", "Network Interface Received Bytes.", variableLabelsInterface, nil),
		"network_if_tx_bytes": prometheus.NewDesc(namespace+"_"+"network_if_tx_bytes", "Network Interface Transmitted Bytes.", variableLabelsInterface, nil),
		"network_if_state":    prometheus.NewDesc(namespace+"_"+"network_if_state", "Network Interface State.", variableLabelsInterface, nil),
	},
	//SEMPv1: show stats client
	"GlobalStats": {
		"system_total_clients_connected": prometheus.NewDesc(namespace+"_"+"system_total_clients_connected", "Total clients connected.", nil, nil),
		"system_rx_msgs_total":           prometheus.NewDesc(namespace+"_"+"system_rx_msgs_total", "Total client messages received.", nil, nil),
		"system_tx_msgs_total":           prometheus.NewDesc(namespace+"_"+"system_tx_msgs_total", "Total client messages sent.", nil, nil),
		"system_rx_bytes_total":          prometheus.NewDesc(namespace+"_"+"system_rx_bytes_total", "Total client bytes received.", nil, nil),
		"system_tx_bytes_total":          prometheus.NewDesc(namespace+"_"+"system_tx_bytes_total", "Total client bytes sent.", nil, nil),
		"system_total_rx_discards":       prometheus.NewDesc(namespace+"_"+"system_total_rx_discards", "Total ingress discards.", nil, nil),
		"system_total_tx_discards":       prometheus.NewDesc(namespace+"_"+"system_total_tx_discards", "Total egress discards.", nil, nil),
	},
	"Spool": {
		"system_spool_quota_bytes":                         prometheus.NewDesc(namespace+"_"+"system_spool_quota_bytes", "Spool configured max disk usage.", nil, nil),
		"system_spool_quota_msgs":                          prometheus.NewDesc(namespace+"_"+"system_spool_quota_msgs", "Spool configured max number of messages.", nil, nil),
		"system_spool_disk_partition_usage_active_percent": prometheus.NewDesc(namespace+"_"+"system_spool_disk_partition_usage_active_percent", "Total disk usage in percent.", nil, nil),
		"system_spool_disk_partition_usage_mate_percent":   prometheus.NewDesc(namespace+"_"+"system_spool_disk_partition_usage_mate_percent", "Total disk usage of mate instance in percent.", nil, nil),
		"system_spool_usage_bytes":                         prometheus.NewDesc(namespace+"_"+"system_spool_usage_bytes", "Spool total persisted usage.", nil, nil),
		"system_spool_usage_msgs":                          prometheus.NewDesc(namespace+"_"+"system_spool_usage_msgs", "Spool total number of persisted messages.", nil, nil),
		"system_spool_files_utilization_percent":           prometheus.NewDesc(namespace+"_"+"system_spool_files_utilization_percent", "Utilization of spool files in percent.", nil, nil),
	},
	"Redundancy": {
		"system_redundancy_up":           prometheus.NewDesc(namespace+"_"+"system_redundancy_up", "Is redundancy up? (0=Down, 1=Up).", variableLabelsRedundancy, nil),
		"system_redundancy_config":       prometheus.NewDesc(namespace+"_"+"system_redundancy_config", "Redundancy configuration (0-Disabled, 1-Enabled, 2-Shutdown)", variableLabelsRedundancy, nil),
		"system_redundancy_role":         prometheus.NewDesc(namespace+"_"+"system_redundancy_role", "Redundancy role (0=Backup, 1=Primary, 2=Monitor, 3-Undefined).", variableLabelsRedundancy, nil),
		"system_redundancy_local_active": prometheus.NewDesc(namespace+"_"+"system_redundancy_local_active", "Is local node the active messaging node? (0-not active, 1-active).", variableLabelsRedundancy, nil),
	},
	//SEMPv1: show replication stats
	"ReplicationStats": {
		//Active stats
		//Message processing
		"system_replication_bridge_admin_state":                   prometheus.NewDesc(namespace+"_"+"system_replication_bridge_admin_state", "Replication Config Sync Bridge Admin State", variableLabelsReplication, nil),
		"system_replication_bridge_state":                         prometheus.NewDesc(namespace+"_"+"system_replication_bridge_state", "Replication Config Sync Bridge State", variableLabelsReplication, nil),
		"system_replication_sync_msgs_queued_to_standby":          prometheus.NewDesc(namespace+"_"+"system_replication_sync_msgs_queued_to_standby", "Replication sync messages queued to standby", variableLabelsReplication, nil),
		"system_replication_sync_msgs_queued_to_standby_as_async": prometheus.NewDesc(namespace+"_"+"system_replication_sync_msgs_queued_to_standby_as_async", "Replication sync messages queued to standby as Async", variableLabelsReplication, nil),
		"system_replication_async_msgs_queued_to_standby":         prometheus.NewDesc(namespace+"_"+"system_replication_async_msgs_queued_to_standby", "Replication async messages queued to standby", variableLabelsReplication, nil),
		"system_replication_promoted_msgs_queued_to_standby":      prometheus.NewDesc(namespace+"_"+"system_replication_promoted_msgs_queued_to_standby", "Replication promoted messages queued to standby", variableLabelsReplication, nil),
		"system_replication_pruned_locally_consumed_msgs":         prometheus.NewDesc(namespace+"_"+"system_replication_pruned_locally_consumed_msgs", "Replication Pruned locally consumed messages", variableLabelsReplication, nil),
		//Sync replication
		"system_replication_transitions_to_ineligible": prometheus.NewDesc(namespace+"_"+"system_replication_transitions_to_ineligible", "Replication transitions to ineligible", variableLabelsReplication, nil),
		//Ack propagation
		"system_replication_msgs_tx_to_standby":   prometheus.NewDesc(namespace+"_"+"system_replication_msgs_tx_to_standby", "system_replication_msgs_tx_to_standby", variableLabelsReplication, nil),
		"system_replication_rec_req_from_standby": prometheus.NewDesc(namespace+"_"+"system_replication_rec_req_from_standby", "system_replication_rec_req_from_standby", variableLabelsReplication, nil),
		//Standby stats
		//Message processing
		"system_replication_msgs_rx_from_active": prometheus.NewDesc(namespace+"_"+"system_replication_msgs_rx_from_active", "Replication msgs rx from active", variableLabelsReplication, nil),
		//Ack propagation
		"system_replication_ack_prop_msgs_rx": prometheus.NewDesc(namespace+"_"+"system_replication_ack_prop_msgs_rx", "Replication ack prop msgs rx", variableLabelsReplication, nil),
		"system_replication_recon_req_tx":     prometheus.NewDesc(namespace+"_"+"system_replication_recon_req_tx", "Replication recon req tx", variableLabelsReplication, nil),
		"system_replication_out_of_seq_rx":    prometheus.NewDesc(namespace+"_"+"system_replication_out_of_seq_rx", "Replication out of seq rx", variableLabelsReplication, nil),
		//Transaction replication
		"system_replication_xa_req":                  prometheus.NewDesc(namespace+"_"+"system_replication_xa_req", "Replication transanction requests", variableLabelsReplication, nil),
		"system_replication_xa_req_success":          prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_success", "Replication transanction requests success", variableLabelsReplication, nil),
		"system_replication_xa_req_success_prepare":  prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_success_prepare", "Replication transanction requests success prepare", variableLabelsReplication, nil),
		"system_replication_xa_req_success_commit":   prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_success_commit", "Replication transanction requests success commit", variableLabelsReplication, nil),
		"system_replication_xa_req_success_rollback": prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_success_rollback", "Replication transanction requests success rollback", variableLabelsReplication, nil),
		"system_replication_xa_req_fail":             prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_fail", "Replication transanction requests fail", variableLabelsReplication, nil),
		"system_replication_xa_req_fail_prepare":     prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_fail_prepare", "Replication transanction requests fail prepare", variableLabelsReplication, nil),
		"system_replication_xa_req_fail_commit":      prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_fail_commit", "Replication transanction requests fail commit", variableLabelsReplication, nil),
		"system_replication_xa_req_fail_rollback":    prometheus.NewDesc(namespace+"_"+"system_replication_xa_req_fail_rollback", "Replication transanction requests fail rollback", variableLabelsReplication, nil),
	},
	"Vpn": {
		"vpn_is_management_vpn":                 prometheus.NewDesc(namespace+"_"+"vpn_is_management_vpn", "VPN is a management VPN", variableLabelsVpn, nil),
		"vpn_enabled":                           prometheus.NewDesc(namespace+"_"+"vpn_enabled", "VPN is enabled", variableLabelsVpn, nil),
		"vpn_operational":                       prometheus.NewDesc(namespace+"_"+"vpn_operational", "VPN is operational", variableLabelsVpn, nil),
		"vpn_locally_configured":                prometheus.NewDesc(namespace+"_"+"vpn_locally_configured", "VPN is locally configured", variableLabelsVpn, nil),
		"vpn_local_status":                      prometheus.NewDesc(namespace+"_"+"vpn_local_status", "Local status (0=Down, 1=Up)", variableLabelsVpn, nil),
		"vpn_unique_subscriptions":              prometheus.NewDesc(namespace+"_"+"vpn_unique_subscriptions", "total subscriptions count", variableLabelsVpn, nil),
		"vpn_total_local_unique_subscriptions":  prometheus.NewDesc(namespace+"_"+"vpn_total_local_unique_subscriptions", "total unique local subscriptions count", variableLabelsVpn, nil),
		"vpn_total_remote_unique_subscriptions": prometheus.NewDesc(namespace+"_"+"vpn_total_remote_unique_subscriptions", "total unique remote subscriptions count", variableLabelsVpn, nil),
		"vpn_total_unique_subscriptions":        prometheus.NewDesc(namespace+"_"+"vpn_total_unique_subscriptions", "total unique subscriptions count", variableLabelsVpn, nil),
		"vpn_connections_service_amqp":          prometheus.NewDesc(namespace+"_"+"vpn_connections_service_amqp", "total number of amq connections", variableLabelsVpn, nil),
		"vpn_connections_service_smf":           prometheus.NewDesc(namespace+"_"+"vpn_connections_service_smf", "total number of smf connections", variableLabelsVpn, nil),
		"vpn_connections":                       prometheus.NewDesc(namespace+"_"+"vpn_connections", "Number of connections.", variableLabelsVpn, nil),
		"vpn_quota_connections":                 prometheus.NewDesc(namespace+"_"+"vpn_quota_connections", "Maximum number of connections.", variableLabelsVpn, nil),
	},
	"VpnReplication": {
		"vpn_replication_admin_state":                  prometheus.NewDesc(namespace+"_"+"vpn_replication_admin_state", "Replication Admin Status (0-shutdown, 1-enabled, 2-n/a)", variableLabelsVpn, nil),
		"vpn_replication_config_state":                 prometheus.NewDesc(namespace+"_"+"vpn_replication_config_state", "Replication Config Status (0-standby, 1-active, 2-n/a)", variableLabelsVpn, nil),
		"vpn_replication_transaction_replication_mode": prometheus.NewDesc(namespace+"_"+"vpn_replication_transaction_replication_mode", "Replication Tx Replication Mode (0-async, 1-sync)", variableLabelsVpn, nil),
	},
	"ConfigSyncVpn": {
		"configsync_table_type":               prometheus.NewDesc(namespace+"_"+"configsync_table_type", "Config Sync Resource Type (0-Router, 1-Vpn, 2-Unknown, 3-None, 4-All)", variableLabelsConfigSyncTable, nil),
		"configsync_table_timeinstateseconds": prometheus.NewDesc(namespace+"_"+"configsync_table_timeinstateseconds", "Config Sync Time in State", variableLabelsConfigSyncTable, nil),
		"configsync_table_ownership":          prometheus.NewDesc(namespace+"_"+"configsync_table_ownership", "Config Sync Ownership (0-Master, 1-Slave, 2-Unknown)", variableLabelsConfigSyncTable, nil),
		"configsync_table_syncstate":          prometheus.NewDesc(namespace+"_"+"configsync_table_syncstate", "Config Sync State (0-Down, 1-Up, 2-Unknown, 3-In-Sync, 4-Reconciling, 5-Blocked, 6-Out-Of-Sync)", variableLabelsConfigSyncTable, nil),
	},
	"ConfigSyncRouter": {
		"configsync_table_type":               prometheus.NewDesc(namespace+"_"+"configsync_table_type", "Config Sync Resource Type (0-Router, 1-Vpn, 2-Unknown, 3-None, 4-All)", variableLabelsConfigSyncTable, nil),
		"configsync_table_timeinstateseconds": prometheus.NewDesc(namespace+"_"+"configsync_table_timeinstateseconds", "Config Sync Time in State", variableLabelsConfigSyncTable, nil),
		"configsync_table_ownership":          prometheus.NewDesc(namespace+"_"+"configsync_table_ownership", "Config Sync Ownership (0-Master, 1-Slave, 2-Unknown)", variableLabelsConfigSyncTable, nil),
		"configsync_table_syncstate":          prometheus.NewDesc(namespace+"_"+"configsync_table_syncstate", "Config Sync State (0-Down, 1-Up, 2-Unknown, 3-In-Sync, 4-Reconciling, 5-Blocked, 6-Out-Of-Sync)", variableLabelsConfigSyncTable, nil),
	},
	"Bridge": {
		"bridges_num_total_bridges":                         prometheus.NewDesc(namespace+"_"+"bridges_num_total_bridges", "Number of Bridges", nil, nil),
		"bridges_max_num_total_bridges":                     prometheus.NewDesc(namespace+"_"+"bridges_max_num_total_bridges", "Max number of Bridges", nil, nil),
		"bridges_num_local_bridges":                         prometheus.NewDesc(namespace+"_"+"bridges_num_local_bridges", "Number of Local Bridges", nil, nil),
		"bridges_max_num_local_bridges":                     prometheus.NewDesc(namespace+"_"+"bridges_max_num_local_bridges", "Max number of Local Bridges", nil, nil),
		"bridges_num_remote_bridges":                        prometheus.NewDesc(namespace+"_"+"bridges_num_remote_bridges", "Number of Remote Bridges", nil, nil),
		"bridges_max_num_remote_bridges":                    prometheus.NewDesc(namespace+"_"+"bridges_max_num_remote_bridges", "Max number of Remote Bridges", nil, nil),
		"bridges_num_total_remote_bridge_subscriptions":     prometheus.NewDesc(namespace+"_"+"bridges_num_total_remote_bridge_subscriptions", "Total number of Remote Bridge Subscription", nil, nil),
		"bridges_max_num_total_remote_bridge_subscriptions": prometheus.NewDesc(namespace+"_"+"bridges_max_num_total_remote_bridge_subscriptions", "Max total number of Remote Bridge Subscription", nil, nil),
		"bridge_admin_state":                                prometheus.NewDesc(namespace+"_"+"bridge_admin_state", "Bridge Administrative State (0-Enabled 1-Disabled, 2--)", variableLabelsBridge, nil),
		"bridge_connection_establisher":                     prometheus.NewDesc(namespace+"_"+"bridge_connection_establisher", "Connection Establisher (0-NotApplicable, 1-Local, 2-Remote, 3-Invalid)", variableLabelsBridge, nil),
		"bridge_inbound_operational_state":                  prometheus.NewDesc(namespace+"_"+"bridge_inbound_operational_state", "Inbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge, nil),
		"bridge_inbound_operational_failure_reason":         prometheus.NewDesc(namespace+"_"+"bridge_inbound_operational_failure_reason", "Inbound Ops Failure Reason (various very long codes)", variableLabelsBridge, nil),
		"bridge_outbound_operational_state":                 prometheus.NewDesc(namespace+"_"+"bridge_outbound_operational_state", "Outbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge, nil),
		"bridge_queue_operational_state":                    prometheus.NewDesc(namespace+"_"+"bridge_queue_operational_state", "Queue Ops State (0-NotApplicable, 1-Bound, 2-Unbound)", variableLabelsBridge, nil),
		"bridge_redundancy":                                 prometheus.NewDesc(namespace+"_"+"bridge_redundancy", "Bridge Redundancy (0-NotApplicable, 1-auto, 2-primary, 3-backup, 4-static, 5-none)", variableLabelsBridge, nil),
		"bridge_connection_uptime_in_seconds":               prometheus.NewDesc(namespace+"_"+"bridge_connection_uptime_in_seconds", "Connection Uptime (s)", variableLabelsBridge, nil),
	},
	"VpnSpool": {
		"vpn_spool_quota_bytes": prometheus.NewDesc(namespace+"_"+"vpn_spool_quota_bytes", "Spool configured max disk usage.", variableLabelsVpn, nil),
		"vpn_spool_usage_bytes": prometheus.NewDesc(namespace+"_"+"vpn_spool_usage_bytes", "Spool total persisted usage.", variableLabelsVpn, nil),
		"vpn_spool_usage_msgs":  prometheus.NewDesc(namespace+"_"+"vpn_spool_usage_msgs", "Spool total number of persisted messages.", variableLabelsVpn, nil),
	},
	//SEMPv1: show client <client-name> message-vpn <vpn-name> connected
	"Client": {
		"client_num_subscriptions": prometheus.NewDesc(namespace+"_"+"client_num_subscriptions", "Number of client subscriptions.", variableLabelsClientInfo, nil),
	},
	//SEMPv1: show client <client-name> message-vpn <vpn-name> connected
	"ClientSlowSubscriber": {
		"client_slow_subscriber": prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsClientInfo, nil),
	},
	"ClientStats": {
		"client_rx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_rx_msgs_total", "Number of received messages.", variableLabelsVpnClientUser, nil),
		"client_tx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_tx_msgs_total", "Number of transmitted messages.", variableLabelsVpnClientUser, nil),
		"client_rx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_rx_bytes_total", "Number of received bytes.", variableLabelsVpnClientUser, nil),
		"client_tx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_tx_bytes_total", "Number of transmitted bytes.", variableLabelsVpnClientUser, nil),
		"client_rx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_rx_discarded_msgs_total", "Number of discarded received messages.", variableLabelsVpnClientUser, nil),
		"client_tx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_tx_discarded_msgs_total", "Number of discarded transmitted messages.", variableLabelsVpnClientUser, nil),
		"client_slow_subscriber":         prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClientUser, nil),
	},
	"ClientMessageSpoolStats": {
		"client_flows_ingress":   prometheus.NewDesc(namespace+"_"+"client_flows_ingress", "Number of ingress flows, created/openend by this client.", variableLabelsVpnClientDetail, nil),
		"client_flows_egress":    prometheus.NewDesc(namespace+"_"+"client_flows_egress", "Number of egress flows, created/openend by this client.", variableLabelsVpnClientDetail, nil),
		"client_slow_subscriber": prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClientDetail, nil),

		"spooling_not_ready":                prometheus.NewDesc(namespace+"_"+"client_ingress_spooling_not_ready", "Number of connections closed caused by spoolingNotReady", variableLabelsVpnClientFlow, nil),
		"out_of_order_messages_received":    prometheus.NewDesc(namespace+"_"+"client_ingress_out_of_order_messages_received", "Number of messages, received in wrong order.", variableLabelsVpnClientFlow, nil),
		"duplicate_messages_received":       prometheus.NewDesc(namespace+"_"+"client_ingress_duplicate_messages_received", "Number of messages, received more than once", variableLabelsVpnClientFlow, nil),
		"no_eligible_destinations":          prometheus.NewDesc(namespace+"_"+"client_ingress_no_eligible_destinations", "???", variableLabelsVpnClientFlow, nil),
		"guaranteed_messages":               prometheus.NewDesc(namespace+"_"+"client_ingress_guaranteed_messages", "Number of gurantied messages, received.", variableLabelsVpnClientFlow, nil),
		"no_local_delivery":                 prometheus.NewDesc(namespace+"_"+"client_ingress__no_local_delivery", "Number of messages, no localy delivered.", variableLabelsVpnClientFlow, nil),
		"seq_num_rollover":                  prometheus.NewDesc(namespace+"_"+"client_ingress_seq_num_rollover", "???", variableLabelsVpnClientFlow, nil),
		"seq_num_messages_discarded":        prometheus.NewDesc(namespace+"_"+"client_ingress_seq_num_messages_discarded", "???", variableLabelsVpnClientFlow, nil),
		"transacted_messages_not_sequenced": prometheus.NewDesc(namespace+"_"+"client_ingress_transacted_messages_not_sequenced", "???", variableLabelsVpnClientFlow, nil),
		"destination_group_error":           prometheus.NewDesc(namespace+"_"+"client_ingress_destination_group_error", "???", variableLabelsVpnClientFlow, nil),
		"smf_ttl_exceeded":                  prometheus.NewDesc(namespace+"_"+"client_ingress_smf_ttl_exceeded", "???", variableLabelsVpnClientFlow, nil),
		"publish_acl_denied":                prometheus.NewDesc(namespace+"_"+"client_ingress_publish_acl_denied", "???", variableLabelsVpnClientFlow, nil),

		"window_size":                           prometheus.NewDesc(namespace+"_"+"client_egress_window_size", "Configured window size", variableLabelsVpnClientFlow, nil),
		"used_window":                           prometheus.NewDesc(namespace+"_"+"client_egress_used_window", "Used windows size.", variableLabelsVpnClientFlow, nil),
		"window_closed":                         prometheus.NewDesc(namespace+"_"+"client_egress_window_closed", "Number windows closed.", variableLabelsVpnClientFlow, nil),
		"message_redelivered":                   prometheus.NewDesc(namespace+"_"+"client_egress_message_redelivered", "Number of messages, was been redelivered.", variableLabelsVpnClientFlow, nil),
		"message_transport_retransmit":          prometheus.NewDesc(namespace+"_"+"client_egress_message_transport_retransmit", "Number of messages, was been retransmitted.", variableLabelsVpnClientFlow, nil),
		"message_confirmed_delivered":           prometheus.NewDesc(namespace+"_"+"client_egress_message_confirmed_delivered", "Number of messages succesfully delivered.", variableLabelsVpnClientFlow, nil),
		"confirmed_delivered_store_and_forward": prometheus.NewDesc(namespace+"_"+"client_egress_confirmed_delivered_store_and_forward", "???", variableLabelsVpnClientFlow, nil),
		"confirmed_delivered_cut_through":       prometheus.NewDesc(namespace+"_"+"client_egress_confirmed_delivered_cut_through", "???", variableLabelsVpnClientFlow, nil),
		"unacked_messages":                      prometheus.NewDesc(namespace+"_"+"client_egress_unacked_messages", "Number of unacknowledged messages.", variableLabelsVpnClientFlow, nil),
	},
	"VpnStats": {
		"vpn_rx_msgs_total":           prometheus.NewDesc(namespace+"_"+"vpn_rx_msgs_total", "Number of received messages.", variableLabelsVpn, nil),
		"vpn_tx_msgs_total":           prometheus.NewDesc(namespace+"_"+"vpn_tx_msgs_total", "Number of transmitted messages.", variableLabelsVpn, nil),
		"vpn_rx_bytes_total":          prometheus.NewDesc(namespace+"_"+"vpn_rx_bytes_total", "Number of received bytes.", variableLabelsVpn, nil),
		"vpn_tx_bytes_total":          prometheus.NewDesc(namespace+"_"+"vpn_tx_bytes_total", "Number of transmitted bytes.", variableLabelsVpn, nil),
		"vpn_rx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"vpn_rx_discarded_msgs_total", "Number of discarded received messages.", variableLabelsVpn, nil),
		"vpn_tx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"vpn_tx_discarded_msgs_total", "Number of discarded transmitted messages.", variableLabelsVpn, nil),
	},
	"BridgeStats": {
		"bridge_client_num_subscriptions":               prometheus.NewDesc(namespace+"_"+"bridge_client_num_subscriptions", "Bridge Client Subscription", variableLabelsBridgeStats, nil),
		"bridge_client_slow_subscriber":                 prometheus.NewDesc(namespace+"_"+"bridge_client_slow_subscriber", "Bridge Slow Subscriber", variableLabelsBridgeStats, nil),
		"bridge_total_client_messages_received":         prometheus.NewDesc(namespace+"_"+"bridge_total_client_messages_received", "Bridge Total Client Messages Received", variableLabelsBridgeStats, nil),
		"bridge_total_client_messages_sent":             prometheus.NewDesc(namespace+"_"+"bridge_total_client_messages_sent", "Bridge Total Client Messages sent", variableLabelsBridgeStats, nil),
		"bridge_client_data_messages_received":          prometheus.NewDesc(namespace+"_"+"bridge_client_data_messages_received", "Bridge Client Data Msgs Received", variableLabelsBridgeStats, nil),
		"bridge_client_data_messages_sent":              prometheus.NewDesc(namespace+"_"+"bridge_client_data_messages_sent", "Bridge Client Data Msgs Sent", variableLabelsBridgeStats, nil),
		"bridge_client_persistent_messages_received":    prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_messages_received", "Bridge Client Persistent Msgs Received", variableLabelsBridgeStats, nil),
		"bridge_client_persistent_messages_sent":        prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_messages_sent", "Bridge Client Persistent Msgs Sent", variableLabelsBridgeStats, nil),
		"bridge_client_nonpersistent_messages_received": prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_messages_received", "Bridge Client Non-Persistent Msgs Received", variableLabelsBridgeStats, nil),
		"bridge_client_nonpersistent_messages_sent":     prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_messages_sent", "Bridge Client Non-Persistent Msgs Sent", variableLabelsBridgeStats, nil),
		"bridge_client_direct_messages_received":        prometheus.NewDesc(namespace+"_"+"bridge_client_direct_messages_received", "Bridge Client Direct Msgs Received", variableLabelsBridgeStats, nil),
		"bridge_client_direct_messages_sent":            prometheus.NewDesc(namespace+"_"+"bridge_client_direct_messages_sent", "Bridge Client Direct Msgs Sent", variableLabelsBridgeStats, nil),
		"bridge_total_client_bytes_received":            prometheus.NewDesc(namespace+"_"+"bridge_total_client_bytes_received", "Bridge Total Client Bytes Received", variableLabelsBridgeStats, nil),
		"bridge_total_client_bytes_sent":                prometheus.NewDesc(namespace+"_"+"bridge_total_client_bytes_sent", "Bridge Total Client Bytes sent", variableLabelsBridgeStats, nil),
		"bridge_client_data_bytes_received":             prometheus.NewDesc(namespace+"_"+"bridge_client_data_bytes_received", "Bridge Client Data Bytes Received", variableLabelsBridgeStats, nil),
		"bridge_client_data_bytes_sent":                 prometheus.NewDesc(namespace+"_"+"bridge_client_data_bytes_sent", "Bridge Client Data Bytes Sent", variableLabelsBridgeStats, nil),
		"bridge_client_persistent_bytes_received":       prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_bytes_received", "Bridge Client Persistent Bytes Received", variableLabelsBridgeStats, nil),
		"bridge_client_persistent_bytes_sent":           prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_bytes_sent", "Bridge Client Persistent Bytes Sent", variableLabelsBridgeStats, nil),
		"bridge_client_nonpersistent_bytes_received":    prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_bytes_received", "Bridge Client Non-Persistent Bytes Received", variableLabelsBridgeStats, nil),
		"bridge_client_nonpersistent_bytes_sent":        prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_bytes_sent", "Bridge Client Non-Persistent Bytes Sent", variableLabelsBridgeStats, nil),
		"bridge_client_direct_bytes_received":           prometheus.NewDesc(namespace+"_"+"bridge_client_direct_bytes_received", "Bridge Client Direct Bytes Received", variableLabelsBridgeStats, nil),
		"bridge_client_direct_bytes_sent":               prometheus.NewDesc(namespace+"_"+"bridge_client_direct_bytes_sent", "Bridge Client Direct Bytes Sent", variableLabelsBridgeStats, nil),
		"bridge_client_large_messages_received":         prometheus.NewDesc(namespace+"_"+"bridge_client_large_messages_received", "Bridge Client Large Messages received", variableLabelsBridgeStats, nil),
		"bridge_denied_duplicate_clients":               prometheus.NewDesc(namespace+"_"+"bridge_denied_duplicate_clients", "Bridge Deneid Duplicate Clients", variableLabelsBridgeStats, nil),
		"bridge_not_enough_space_msgs_sent":             prometheus.NewDesc(namespace+"_"+"bridge_not_enough_space_msgs_sent", "Bridge Not Enough Space Messages Sent", variableLabelsBridgeStats, nil),
		"bridge_max_exceeded_msgs_sent":                 prometheus.NewDesc(namespace+"_"+"bridge_max_exceeded_msgs_sent", "Bridge Max Exceeded Messages Sent", variableLabelsBridgeStats, nil),
		"bridge_subscribe_client_not_found":             prometheus.NewDesc(namespace+"_"+"bridge_subscribe_client_not_found", "Bridge Subscriber Client Not Found", variableLabelsBridgeStats, nil),
		"bridge_not_found_msgs_sent":                    prometheus.NewDesc(namespace+"_"+"bridge_not_found_msgs_sent", "Bridge Not Found Messages Sent", variableLabelsBridgeStats, nil),
		"bridge_current_ingress_rate_per_second":        prometheus.NewDesc(namespace+"_"+"bridge_current_ingress_rate_per_second", "Current Ingress Rate / s", variableLabelsBridgeStats, nil),
		"bridge_current_egress_rate_per_second":         prometheus.NewDesc(namespace+"_"+"bridge_current_egress_rate_per_second", "Current Egress Rate / s", variableLabelsBridgeStats, nil),
		"bridge_total_ingress_discards":                 prometheus.NewDesc(namespace+"_"+"bridge_total_ingress_discards", "Total Ingress Discards", variableLabelsBridgeStats, nil),
		"bridge_total_egress_discards":                  prometheus.NewDesc(namespace+"_"+"bridge_total_egress_discards", "Total Egress Discards", variableLabelsBridgeStats, nil),
	},
	"QueueRates": {
		"queue_rx_msg_rate":      prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate", "Rate of received messages.", variableLabelsVpnQueue, nil),
		"queue_tx_msg_rate":      prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpnQueue, nil),
		"queue_rx_byte_rate":     prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate", "Rate of received bytes.", variableLabelsVpnQueue, nil),
		"queue_tx_byte_rate":     prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpnQueue, nil),
		"queue_rx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpnQueue, nil),
		"queue_tx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpnQueue, nil),
		"queue_rx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpnQueue, nil),
		"queue_tx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpnQueue, nil),
	},
	"QueueDetails": {
		"queue_spool_quota_bytes": prometheus.NewDesc(namespace+"_"+"queue_spool_quota_bytes", "Queue spool configured max disk usage in bytes.", variableLabelsVpnQueue, nil),
		"queue_spool_usage_bytes": prometheus.NewDesc(namespace+"_"+"queue_spool_usage_bytes", "Queue spool usage in bytes.", variableLabelsVpnQueue, nil),
		"queue_spool_usage_msgs":  prometheus.NewDesc(namespace+"_"+"queue_spool_usage_msgs", "Queue spooled number of messages.", variableLabelsVpnQueue, nil),
		"queue_binds":             prometheus.NewDesc(namespace+"_"+"queue_binds", "Number of clients bound to queue.", variableLabelsVpnQueue, nil),
	},
	"QueueStats": {
		"total_bytes_spooled":                 prometheus.NewDesc(namespace+"_"+"queue_byte_spooled", "Queue spool total of all spooled messages in bytes.", variableLabelsVpnQueue, nil),
		"total_messages_spooled":              prometheus.NewDesc(namespace+"_"+"queue_msg_spooled", "Queue spool total of all spooled messages.", variableLabelsVpnQueue, nil),
		"messages_redelivered":                prometheus.NewDesc(namespace+"_"+"queue_msg_redelivered", "Queue total msg redeliveries.", variableLabelsVpnQueue, nil),
		"messages_transport_retransmited":     prometheus.NewDesc(namespace+"_"+"queue_msg_retransmited", "Queue total msg retransmitted on transport.", variableLabelsVpnQueue, nil),
		"spool_usage_exceeded":                prometheus.NewDesc(namespace+"_"+"queue_msg_spool_usage_exceeded", "Queue total number of messages exceeded the spool usage.", variableLabelsVpnQueue, nil),
		"max_message_size_exceeded":           prometheus.NewDesc(namespace+"_"+"queue_msg_max_msg_size_exceeded", "Queue total number of messages exceeded the max message size.", variableLabelsVpnQueue, nil),
		"total_deleted_messages":              prometheus.NewDesc(namespace+"_"+"queue_msg_total_deleted", "Queue total number that was deleted.", variableLabelsVpnQueue, nil),
		"messages_shutdown_discarded":         prometheus.NewDesc(namespace+"_"+"queue_msg_shutdown_discarded", "Queue total number of messages discarded due to spool shutdown.", variableLabelsVpnQueue, nil),
		"messages_ttl_discarded":              prometheus.NewDesc(namespace+"_"+"queue_msg_ttl_discarded", "Queue total number of messages discarded due to ttl expiry.", variableLabelsVpnQueue, nil),
		"messages_ttl_dmq":                    prometheus.NewDesc(namespace+"_"+"queue_msg_ttl_dmq", "Queue total number of messages delivered to dmq due to ttl expiry.", variableLabelsVpnQueue, nil),
		"messages_ttl_dmq_failed":             prometheus.NewDesc(namespace+"_"+"queue_msg_ttl_dmq_failed", "Queue total number of messages that failed delivery to dmq due to ttl expiry.", variableLabelsVpnQueue, nil),
		"messages_max_redelivered_discarded":  prometheus.NewDesc(namespace+"_"+"queue_msg_max_redelivered_discarded", "Queue total number of messages discarded due to exceeded max redelivery.", variableLabelsVpnQueue, nil),
		"messages_max_redelivered_dmq":        prometheus.NewDesc(namespace+"_"+"queue_msg_max_redelivered_dmq", "Queue total number of messages delivered to dmq due to exceeded max redelivery.", variableLabelsVpnQueue, nil),
		"messages_max_redelivered_dmq_failed": prometheus.NewDesc(namespace+"_"+"queue_msg_max_redelivered_dmq_failed", "Queue total number of messages failed delivery to dmq due to exceeded max redelivery.", variableLabelsVpnQueue, nil),
	},
	"TopicEndpointRates": {
		"rx_msg_rate":      prometheus.NewDesc(namespace+"_"+"topic_endpoint_rx_msg_rate", "Rate of received messages.", variableLabelsVpnTopicEndpoint, nil),
		"tx_msg_rate":      prometheus.NewDesc(namespace+"_"+"topic_endpoint_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpnTopicEndpoint, nil),
		"rx_byte_rate":     prometheus.NewDesc(namespace+"_"+"topic_endpoint_rx_byte_rate", "Rate of received bytes.", variableLabelsVpnTopicEndpoint, nil),
		"tx_byte_rate":     prometheus.NewDesc(namespace+"_"+"topic_endpoint_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpnTopicEndpoint, nil),
		"rx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"topic_endpoint_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpnTopicEndpoint, nil),
		"tx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"topic_endpoint_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpnTopicEndpoint, nil),
		"rx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"topic_endpoint_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpnTopicEndpoint, nil),
		"tx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"topic_endpoint_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpnTopicEndpoint, nil),
	},
	"TopicEndpointDetails": {
		"spool_quota_bytes": prometheus.NewDesc(namespace+"_"+"topic_endpoint_spool_quota_bytes", "Topic Endpoint spool configured max disk usage in bytes.", variableLabelsVpnTopicEndpoint, nil),
		"spool_usage_bytes": prometheus.NewDesc(namespace+"_"+"topic_endpoint_spool_usage_bytes", "Topic Endpoint spool usage in bytes.", variableLabelsVpnTopicEndpoint, nil),
		"spool_usage_msgs":  prometheus.NewDesc(namespace+"_"+"topic_endpoint_spool_usage_msgs", "Topic Endpoint spooled number of messages.", variableLabelsVpnTopicEndpoint, nil),
		"binds":             prometheus.NewDesc(namespace+"_"+"topic_endpoint_binds", "Number of clients bound to topic-endpoin.", variableLabelsVpnTopicEndpoint, nil),
	},
	"TopicEndpointStats": {
		"total_bytes_spooled":             prometheus.NewDesc(namespace+"_"+"topic_endpoint_byte_spooled", "Topic Endpoint spool total of all spooled messages in bytes.", variableLabelsVpnTopicEndpoint, nil),
		"total_messages_spooled":          prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_spooled", "Topic Endpoint spool total of all spooled messages.", variableLabelsVpnTopicEndpoint, nil),
		"messages_redelivered":            prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_redelivered", "Topic Endpoint total msg redeliveries.", variableLabelsVpnTopicEndpoint, nil),
		"messages_transport_retransmited": prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_retransmited", "Topic Endpoint total msg retransmitted on transport.", variableLabelsVpnTopicEndpoint, nil),
		"spool_usage_exceeded":            prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_spool_usage_exceeded", "Topic Endpoint total number of messages exceeded the spool usage.", variableLabelsVpnTopicEndpoint, nil),
		"max_message_size_exceeded":       prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_max_msg_size_exceeded", "Topic Endpoint total number of messages exceeded the max message size.", variableLabelsVpnTopicEndpoint, nil),
		"total_deleted_messages":          prometheus.NewDesc(namespace+"_"+"topic_endpoint_msg_total_deleted", "Topic Endpoint total number that was deleted.", variableLabelsVpnTopicEndpoint, nil),
	},
	"ClusterLinks": {
		"enabled":     prometheus.NewDesc(namespace+"_"+"cluster_link_enabled", "Clustter link is enabled.", variableLabelsCluserLink, nil),
		"oper_up":     prometheus.NewDesc(namespace+"_"+"cluster_link_operational", "Clustter link is operational.", variableLabelsCluserLink, nil),
		"oper_uptime": prometheus.NewDesc(namespace+"_"+"cluster_link_uptime", "Clustter link utime in seconds.", variableLabelsCluserLink, nil),
	},
	"ClientConnections": {
		"connection_is_zip":           				   prometheus.NewDesc(namespace+"_"+"connection_is_zip", "Connection is zip compressed.", variableLabelsVpnClient, nil),
		"connection_is_ssl":           				   prometheus.NewDesc(namespace+"_"+"connection_is_ssl", "Connection is ssl encrypted.", variableLabelsVpnClient, nil),
		"connection_receive_queue_bytes":              prometheus.NewDesc(namespace+"_"+"connection_receive_queue_bytes", "The number of bytes currently in the event broker receive queue for the TCP connection.", variableLabelsVpnClient, nil),
		"connection_send_queue_bytes":                 prometheus.NewDesc(namespace+"_"+"connection_send_queue_bytes", "The number of bytes currently in the event broker receive queue for the TCP connection.", variableLabelsVpnClient, nil),
		"connection_receive_queue_segments":           prometheus.NewDesc(namespace+"_"+"connection_receive_queue_segments", "The number of bytes currently queued for the client in both the client’s egress queues and the TCP send queue.", variableLabelsVpnClient, nil),
		"connection_send_queue_segments":              prometheus.NewDesc(namespace+"_"+"connection_send_queue_segments", "The number of messages currently queued for the client in its egress queues.", variableLabelsVpnClient, nil),
		"connection_maximum_segment_size":             prometheus.NewDesc(namespace+"_"+"connection_maximum_segment_size", "The maximum segment size (MSS) configured for the client connection. The MSS is configured in the client profile. See RFC 879 for further details.", variableLabelsVpnClient, nil),
		"connection_sent_bytes":                       prometheus.NewDesc(namespace+"_"+"connection_sent_bytes", "The number of bytes sent by the event broker on the TCP connection", variableLabelsVpnClient, nil),
		"connection_received_bytes":                   prometheus.NewDesc(namespace+"_"+"connection_received_bytes", "The number of bytes received by the event broker on the TCP connection", variableLabelsVpnClient, nil),

		"connection_retransmit_milliseconds":          prometheus.NewDesc(namespace+"_"+"connection_retransmit_milliseconds", "The retransmission timeout (RTO) in milliseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient, nil),
		"connection_roundtrip_smth_microseconds":      prometheus.NewDesc(namespace+"_"+"connection_roundtrip_smth_microseconds", "The smoothed round-trip time (SRTT) in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient, nil),
		"connection_roundtrip_min_microseconds":       prometheus.NewDesc(namespace+"_"+"connection_roundtrip_min_microseconds", "The minimum round-trip time in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient, nil),
		"connection_roundtrip_var_microseconds":       prometheus.NewDesc(namespace+"_"+"connection_roundtrip_var_microseconds", "The round-trip time variation (RTTVAR) in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient, nil),
		
		"connection_advertised_window":       		   prometheus.NewDesc(namespace+"_"+"connection_advertised_window", "The receive window size in bytes advertised to the client on the remote end of the TCP connection. See RFC 793 for further details.", variableLabelsVpnClient, nil),
		"connection_transmit_window":       		   prometheus.NewDesc(namespace+"_"+"connection_transmit_window", "The send window size in bytes. See RFC 793 for further details.", variableLabelsVpnClient, nil),
		"connection_congestion_window":       		   prometheus.NewDesc(namespace+"_"+"connection_congestion_window", "The congestion window size in bytes (cwnd). See RFC 5681 for further details.", variableLabelsVpnClient, nil),
		
		"connection_slow_start_threshold":       	   prometheus.NewDesc(namespace+"_"+"connection_slow_start_threshold", "The slow start threshold in bytes (ssthresh). See RFC 5681 for further details.", variableLabelsVpnClient, nil),
		"connection_received_outoforder":       	   prometheus.NewDesc(namespace+"_"+"connection_received_outoforder", "The number of TCP segments received out of order.", variableLabelsVpnClient, nil),
		"connection_fast_retransmit":       	       prometheus.NewDesc(namespace+"_"+"connection_fast_retransmit", "The number of TCP segments retransmitted due to the receipt of duplicate acknowledgments (‘ACKs’). See RFC 5681 for further details.", variableLabelsVpnClient, nil),
		"connection_timed_retransmit":       	       prometheus.NewDesc(namespace+"_"+"connection_timed_retransmit", "The number of TCP segments re-transmitted due to timeout awaiting an ACK. See RFC 793 for further details.", variableLabelsVpnClient, nil),
	},
}
