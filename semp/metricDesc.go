package semp

const (
	namespace = "solace" // For Prometheus metrics.
)

var (
	variableLabelsUp               = []string{"error"}
	variableLabelsEnvironment      = []string{"sensor_name"}
	variableLabelsHardwareFC       = []string{"channel_number"}
	variableLabelsHardwareLUN      = []string{"lun_number"}
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
	variableLabelsBridgeRemote     = []string{"vpn_name", "bridge_name", "remote_vpn_name", "remote_router"}
	variableLabelsBridgeStats      = []string{"vpn_name", "bridge_name", "remote_router_name", "remote_vpn_name"}
	variableLabelsConfigSyncTable  = []string{"table_name"}
	variableLabelsStorageElement   = []string{"path", "device_name", "element_name"}
	variableLabelsDisk             = []string{"path", "device_name"}
	variableLabelsInterface        = []string{"interface_name"}
	variableLabelsRaid             = []string{"disk_number", "device_model"}
)

var QueueStats = Descriptions{
	"total_bytes_spooled":                 NewSemDesc("queue_byte_spooled", "spooledByteCount", "The total amount of all messages ever spooled in the queue, in bytes.", variableLabelsVpnQueue),
	"total_messages_spooled":              NewSemDesc("queue_msg_spooled", "spooledMsgCount", "Queue spool total of all spooled messages.", variableLabelsVpnQueue),
	"messages_redelivered":                NewSemDesc("queue_msg_redelivered", "redeliveredMsgCount", "Queue total msg redeliveries.", variableLabelsVpnQueue),
	"messages_transport_retransmited":     NewSemDesc("queue_msg_retransmited", "transportRetransmitMsgCount", "Queue total msg retransmitted on transport.", variableLabelsVpnQueue),
	"spool_usage_exceeded":                NewSemDesc("queue_msg_spool_usage_exceeded", "maxMsgSpoolUsageExceededDiscardedMsgCount", "Queue total number of messages exceeded the spool usage.", variableLabelsVpnQueue),
	"max_message_size_exceeded":           NewSemDesc("queue_msg_max_msg_size_exceeded", "maxMsgSizeExceededDiscardedMsgCount", "Queue total number of messages exceeded the max message size.", variableLabelsVpnQueue),
	"total_deleted_messages":              NewSemDesc("queue_msg_total_deleted", "deletedMsgCount", "Queue total number that was deleted.", variableLabelsVpnQueue),
	"messages_shutdown_discarded":         NewSemDesc("queue_msg_shutdown_discarded", "disabledDiscardedMsgCount", "Queue total number of messages discarded due to spool shutdown.", variableLabelsVpnQueue),
	"messages_ttl_discarded":              NewSemDesc("queue_msg_ttl_discarded", "maxTtlExpiredDiscardedMsgCount", "Queue total number of messages discarded due to ttl expiry.", variableLabelsVpnQueue),
	"messages_ttl_dmq":                    NewSemDesc("queue_msg_ttl_dmq", "maxTtlExpiredToDmqMsgCount", "Queue total number of messages delivered to dmq due to ttl expiry.", variableLabelsVpnQueue),
	"messages_ttl_dmq_failed":             NewSemDesc("queue_msg_ttl_dmq_failed", "maxTtlExpiredToDmqFailedMsgCount", "Queue total number of messages that failed delivery to dmq due to ttl expiry.", variableLabelsVpnQueue),
	"messages_max_redelivered_discarded":  NewSemDesc("queue_msg_max_redelivered_discarded", "maxRedeliveryExceededDiscardedMsgCount", "Queue total number of messages discarded due to exceeded max redelivery.", variableLabelsVpnQueue),
	"messages_max_redelivered_dmq":        NewSemDesc("queue_msg_max_redelivered_dmq", "maxRedeliveryExceededToDmqMsgCount", "Queue total number of messages delivered to dmq due to exceeded max redelivery.", variableLabelsVpnQueue),
	"messages_max_redelivered_dmq_failed": NewSemDesc("queue_msg_max_redelivered_dmq_failed", "maxRedeliveryExceededToDmqFailedMsgCount", "Queue total number of messages failed delivery to dmq due to exceeded max redelivery.", variableLabelsVpnQueue),
}

var MetricDesc = map[string]Descriptions{
	"Global": {
		"up": NewSemDesc("up", NoSempV2Ready, "Was the last scrape of Solace broker successful.", variableLabelsUp),
	},
	"Alarm": {
		"system_alarm": NewSemDesc("system_alarm", NoSempV2Ready, "A system alarm has been triggered 0 = false, 1 = true", nil),
	},
	"Version": {
		"system_version_currentload":      NewSemDesc("system_version_currentload", NoSempV2Ready, "Solace Version as WWWXXXYYYZZZ", nil),
		"system_version_uptime_totalsecs": NewSemDesc("system_version_uptime_totalsecs", NoSempV2Ready, "Broker uptime in seconds", nil),
		"exporter_version_current":        NewSemDesc("exporter_version_current", NoSempV2Ready, "Exporter Version as XXXYYYZZZ", nil),
	},
	"Health": {
		"system_disk_latency_min_seconds":      NewSemDesc("system_disk_latency_min_seconds", NoSempV2Ready, "Minimum disk latency.", nil),
		"system_disk_latency_max_seconds":      NewSemDesc("system_disk_latency_max_seconds", NoSempV2Ready, "Maximum disk latency.", nil),
		"system_disk_latency_avg_seconds":      NewSemDesc("system_disk_latency_avg_seconds", NoSempV2Ready, "Average disk latency.", nil),
		"system_disk_latency_cur_seconds":      NewSemDesc("system_disk_latency_cur_seconds", NoSempV2Ready, "Current disk latency.", nil),
		"system_compute_latency_min_seconds":   NewSemDesc("system_compute_latency_min_seconds", NoSempV2Ready, "Minimum compute latency.", nil),
		"system_compute_latency_max_seconds":   NewSemDesc("system_compute_latency_max_seconds", NoSempV2Ready, "Maximum compute latency.", nil),
		"system_compute_latency_avg_seconds":   NewSemDesc("system_compute_latency_avg_seconds", NoSempV2Ready, "Average compute latency.", nil),
		"system_compute_latency_cur_seconds":   NewSemDesc("system_compute_latency_cur_seconds", NoSempV2Ready, "Current compute latency.", nil),
		"system_mate_link_latency_min_seconds": NewSemDesc("system_mate_link_latency_min_seconds", NoSempV2Ready, "Minimum mate link latency.", nil),
		"system_mate_link_latency_max_seconds": NewSemDesc("system_mate_link_latency_max_seconds", NoSempV2Ready, "Maximum mate link latency.", nil),
		"system_mate_link_latency_avg_seconds": NewSemDesc("system_mate_link_latency_avg_seconds", NoSempV2Ready, "Average mate link latency.", nil),
		"system_mate_link_latency_cur_seconds": NewSemDesc("system_mate_link_latency_cur_seconds", NoSempV2Ready, "Current mate link latency.", nil),
	},
	//SEMPv1 (Software): show storage element <element-name>
	"StorageElement": {
		"system_storage_used_percent": NewSemDesc("system_storage_used_percent", NoSempV2Ready, "Storage Element used percent.", variableLabelsStorageElement),
		"system_storage_used_bytes":   NewSemDesc("system_storage_used_bytes", NoSempV2Ready, "Storage Element used bytes.", variableLabelsStorageElement),
		"system_storage_avail_bytes":  NewSemDesc("system_storage_avail_bytes", NoSempV2Ready, "Storage Element available bytes.", variableLabelsStorageElement),
	},
	//SEMPv1 (Appliance): show disk detail
	"Disk": {
		"system_disk_used_percent": NewSemDesc("system_disk_used_percent", NoSempV2Ready, "Disk used percent.", variableLabelsDisk),
		"system_disk_used_bytes":   NewSemDesc("system_disk_used_bytes", NoSempV2Ready, "Disk used bytes.", variableLabelsDisk),
		"system_disk_avail_bytes":  NewSemDesc("system_disk_avail_bytes", NoSempV2Ready, "Disk available bytes.", variableLabelsDisk),
	},
	//SEMPv1: show memory
	"Memory": {
		"system_memory_physical_usage_percent":     NewSemDesc("system_memory_physical_usage_percent", NoSempV2Ready, "Physical memory usage percent.", nil),
		"system_memory_subscription_usage_percent": NewSemDesc("system_memory_subscription_usage_percent", NoSempV2Ready, "Subscription memory usage percent.", nil),
		"system_nab_buffer_load_factor":            NewSemDesc("system_nab_buffer_load_factor", NoSempV2Ready, "NAB buffer load factor.", nil),
	},
	//SEMPv1: show interface <interface-name>
	"Interface": {
		"network_if_rx_bytes": NewSemDesc("network_if_rx_bytes", NoSempV2Ready, "Network Interface Received Bytes.", variableLabelsInterface),
		"network_if_tx_bytes": NewSemDesc("network_if_tx_bytes", NoSempV2Ready, "Network Interface Transmitted Bytes.", variableLabelsInterface),
		"network_if_state":    NewSemDesc("network_if_state", NoSempV2Ready, "Network Interface State.", variableLabelsInterface),
	},
	"InterfaceHW": {
		"network_if_rx_packets":           NewSemDesc("network_if_rx_packets", NoSempV2Ready, "Network Interface Received Packets.", variableLabelsInterface),
		"network_if_tx_packets":           NewSemDesc("network_if_tx_packets", NoSempV2Ready, "Network Interface Transmitted Packets.", variableLabelsInterface),
		"network_lag_configured_members":  NewSemDesc("network_lag_configured_members", NoSempV2Ready, "Network LAG Configured Members.", variableLabelsInterface),
		"network_lag_available_members":   NewSemDesc("network_lag_available_members", NoSempV2Ready, "Network LAG Available Members.", variableLabelsInterface),
		"network_lag_operational_members": NewSemDesc("network_lag_operational_members", NoSempV2Ready, "Network LAG Operational Members.", variableLabelsInterface),
		"network_if_link_detected":        NewSemDesc("network_if_link_detected", NoSempV2Ready, "Network Interface Link Detected. 0-No, 1-Yes", variableLabelsInterface),
		"network_if_enabled":              NewSemDesc("network_if_enabled", NoSempV2Ready, "Network Interface Enabled. 0-No, 1-Yes", variableLabelsInterface),
	},
	//SEMPv1: show stats client
	"GlobalStats": {
		"system_total_clients_connected": NewSemDesc("system_total_clients_connected", NoSempV2Ready, "Total clients connected.", nil),
		"system_total_clients_quota":     NewSemDesc("system_total_clients_quota", NoSempV2Ready, "Number of maximal possible clients to be connected.", nil),
		"system_message_spool_quota":     NewSemDesc("system_message_spool_quota", NoSempV2Ready, "Number of maximal possible messages to be queue.", nil),
		"system_uptime_seconds":          NewSemDesc("system_uptime_seconds", NoSempV2Ready, "Uptime in seconds.", nil),
		"system_cpu_cores":               NewSemDesc("system_cpu_cores", NoSempV2Ready, "Available cpu cores.", nil),
		"system_memory_bytes":            NewSemDesc("system_memory_bytes", NoSempV2Ready, "Available ram in bytes.", nil),
		"system_rx_msgs_total":           NewSemDesc("system_rx_msgs_total", NoSempV2Ready, "Total client messages received.", nil),
		"system_tx_msgs_total":           NewSemDesc("system_tx_msgs_total", NoSempV2Ready, "Total client messages sent.", nil),
		"system_rx_bytes_total":          NewSemDesc("system_rx_bytes_total", NoSempV2Ready, "Total client bytes received.", nil),
		"system_tx_bytes_total":          NewSemDesc("system_tx_bytes_total", NoSempV2Ready, "Total client bytes sent.", nil),
		"system_total_rx_discards":       NewSemDesc("system_total_rx_discards", NoSempV2Ready, "Total ingress discards.", nil),
		"system_total_tx_discards":       NewSemDesc("system_total_tx_discards", NoSempV2Ready, "Total egress discards.", nil),
	},
	"Raid": {
		"system_disk_state":                      NewSemDesc("system_disk_state", NoSempV2Ready, "Disk state. 0 = down, 1 = up.", variableLabelsRaid),
		"system_disk_AdministrativeStateEnabled": NewSemDesc("system_disk_AdministrativeStateEnabled", NoSempV2Ready, "Disk enablement 0 = disabled, 1 = enabled.", variableLabelsRaid),
		"system_raid_state":                      NewSemDesc("system_raid_state", NoSempV2Ready, "Current RAID state of the internal disks, 1 if fully redundant.", nil),
		"system_reload_required":                 NewSemDesc("system_reload_required", NoSempV2Ready, "1 if a system reload is required.", nil),
	},
	"Spool": {
		"system_spool_quota_bytes":                         NewSemDesc("system_spool_quota_bytes", NoSempV2Ready, "Spool configured max disk usage.", nil),
		"system_spool_quota_msgs":                          NewSemDesc("system_spool_quota_msgs", NoSempV2Ready, "Spool configured max number of messages.", nil),
		"system_spool_disk_partition_usage_active_percent": NewSemDesc("system_spool_disk_partition_usage_active_percent", NoSempV2Ready, "Total disk usage in percent.", nil),
		"system_spool_disk_partition_usage_mate_percent":   NewSemDesc("system_spool_disk_partition_usage_mate_percent", NoSempV2Ready, "Total disk usage of mate instance in percent.", nil),
		"system_spool_usage_bytes":                         NewSemDesc("system_spool_usage_bytes", NoSempV2Ready, "Spool total persisted usage.", nil),
		"system_spool_usage_msgs":                          NewSemDesc("system_spool_usage_msgs", NoSempV2Ready, "Spool total number of persisted messages.", nil),
		"system_spool_files_utilization_percent":           NewSemDesc("system_spool_files_utilization_percent", NoSempV2Ready, "Utilization of spool files in percent.", nil),
		"system_spool_message_count_utilization_percent":   NewSemDesc("system_spool_message_count_utilization_percent", NoSempV2Ready, "Utilization of queue message resource in percent.", nil),

		"system_spool_ingress_flows_quota":             NewSemDesc("system_spool_ingress_flows_quota", NoSempV2Ready, "Number of maximal possible ingress flows.", nil),
		"system_spool_ingress_flows_count":             NewSemDesc("system_spool_ingress_flows_count", NoSempV2Ready, "Number of used ingress flows.", nil),
		"system_spool_egress_flows_quota":              NewSemDesc("system_spool_egress_flows_quota", NoSempV2Ready, "Number of maximal possible egress flows.", nil),
		"system_spool_egress_flows_count":              NewSemDesc("system_spool_egress_flows_count", NoSempV2Ready, "Number of used egress flows.", nil),
		"system_spool_egress_flows_active":             NewSemDesc("system_spool_egress_flows_active", NoSempV2Ready, "Number of used egress flows in state active.", nil),
		"system_spool_egress_flows_inactive":           NewSemDesc("system_spool_egress_flows_inactive", NoSempV2Ready, "Number of used egress flows in state inactive.", nil),
		"system_spool_egress_flows_browser":            NewSemDesc("system_spool_egress_flows_browser", NoSempV2Ready, "Number of used egress flows in queue browser mode.", nil),
		"system_spool_endpoints_quota":                 NewSemDesc("system_spool_endpoints_quota", NoSempV2Ready, "Number of maximal possible queue or topic endpoints.", nil),
		"system_spool_endpoints_queue":                 NewSemDesc("system_spool_endpoints_queue", NoSempV2Ready, "Number of existing queue endpoints.", nil),
		"system_spool_endpoints_dte":                   NewSemDesc("system_spool_endpoints_dte", NoSempV2Ready, "Number of existing topic endpoints.", nil),
		"system_spool_transacted_sessions_quota":       NewSemDesc("system_spool_transacted_sessions_quota", NoSempV2Ready, "Number of maximal possible transacted sessions.", nil),
		"system_spool_transacted_sessions_used":        NewSemDesc("system_spool_transacted_sessions_used", NoSempV2Ready, "Number of used transacted sessions.", nil),
		"system_spool_queue_topic_subscriptions_quota": NewSemDesc("system_spool_queue_topic_subscriptions_quota", NoSempV2Ready, "Number of maximal possible topic subscriptions of all queues.", nil),
		"system_spool_queue_topic_subscriptions_used":  NewSemDesc("system_spool_queue_topic_subscriptions_used", NoSempV2Ready, "Number of used topic subscriptions of all queues.", nil),
		"system_spool_transactions_quota":              NewSemDesc("system_spool_transactions_quota", NoSempV2Ready, "Number of maximal possible transactions.", nil),
		"system_spool_transactions_used":               NewSemDesc("system_spool_transactions_used", NoSempV2Ready, "Number of used transactions.", nil),

		"system_spool_usage_adb_bytes":                    NewSemDesc("system_spool_usage_adb_bytes", NoSempV2Ready, "Spool total persisted usage in adb.", nil),
		"system_spool_messages_currently_spooled_adb":     NewSemDesc("system_spool_messages_currently_spooled_adb", NoSempV2Ready, "Messages stored in adb.", nil),
		"system_spool_messages_currently_spooled_disk":    NewSemDesc("system_spool_messages_currently_spooled_disk", NoSempV2Ready, "Messages stored on disk.", nil),
		"system_spool_transacted_session_utilisation_pct": NewSemDesc("system_spool_transacted_session_utilisation_pct", NoSempV2Ready, "Percentage of transacted sessions used.", nil),
		"system_spool_messages_total_disk_usage_bytes":    NewSemDesc("system_spool_messages_total_disk_usage_bytes", NoSempV2Ready, "Total disk usage.", nil),
		"system_spool_sync_status":                        NewSemDesc("system_spool_sync_status", NoSempV2Ready, "Spool sync status: 0-Synced.", nil),
	},
	"Redundancy": {
		"system_redundancy_up":           NewSemDesc("system_redundancy_up", NoSempV2Ready, "Is redundancy up? (0=Down, 1=Up).", variableLabelsRedundancy),
		"system_redundancy_config":       NewSemDesc("system_redundancy_config", NoSempV2Ready, "Redundancy configuration (0-Disabled, 1-Enabled, 2-Shutdown)", variableLabelsRedundancy),
		"system_redundancy_role":         NewSemDesc("system_redundancy_role", NoSempV2Ready, "Redundancy role (0=Backup, 1=Primary, 2=Monitor, 3-Undefined).", variableLabelsRedundancy),
		"system_redundancy_local_active": NewSemDesc("system_redundancy_local_active", NoSempV2Ready, "Is local node the active messaging node? (0-not active, 1-active).", variableLabelsRedundancy),
	},
	"RedundancyHW": {
		"system_redundancy_role":      NewSemDesc("system_redundancy_role", NoSempV2Ready, "Redundancy role (0=Backup, 1=Primary, 2-Undefined).", variableLabelsRedundancy),
		"system_redundancy_mode":      NewSemDesc("system_redundancy_mode", NoSempV2Ready, "Redundancy mode (0=Active/Active, 1=Active/Standby).", variableLabelsRedundancy),
		"system_redundancy_adb_link":  NewSemDesc("system_redundancy_adb_link", NoSempV2Ready, "Is adb link up? (0-no, 1-yes).", variableLabelsRedundancy),
		"system_redundancy_adb_hello": NewSemDesc("system_redundancy_adb_hello", NoSempV2Ready, "Is adb link connected? (0-no, 1-yes).", variableLabelsRedundancy),
	},
	"Environment": {
		"system_chassis_fan_speed_rpm": NewSemDesc("system_chassis_fan_speed_rpm", NoSempV2Ready, "Chassis Fan Speed (RPM)", variableLabelsEnvironment),
		"system_cpu_thermal_margin":    NewSemDesc("system_cpu_thermal_margin", NoSempV2Ready, "CPU thermal headroom (Degrees C, larger negative values are better.)", variableLabelsEnvironment),
		"system_nab_core_temperature":  NewSemDesc("system_nab_core_temperature", NoSempV2Ready, "NAB core temperature (Degrees C).", variableLabelsEnvironment),
	},
	"Hardware": {
		"operational_power_supplies":      NewSemDesc("operational_power_supplies", NoSempV2Ready, "Number of operational power supplies", nil),
		"fibre_channel_operational_state": NewSemDesc("fibre_channel_operational_state", NoSempV2Ready, "Fibre channel operational state 0-Link Down 1-Online", variableLabelsHardwareFC),
		"fibre_channel_state":             NewSemDesc("fibre_channel_state", NoSempV2Ready, "Fibre channel state 0-Link Down 1-Link Up, 2-Link Up Loop", variableLabelsHardwareFC),
		"external_disk_lun_state":         NewSemDesc("external_disk_lun_state", NoSempV2Ready, "External Disk LUN state 0-Offline 1-Ready", variableLabelsHardwareLUN),
		"adb_operational_state":           NewSemDesc("adb_operational_state", NoSempV2Ready, "ADB Operational State, -1,0-Not OK 1-OK", nil),
		"adb_flash_card_state":            NewSemDesc("adb_flash_card_state", NoSempV2Ready, "ADB Flash Card State, -1,0-Not OK 1-OK", nil),
		"adb_power_module_state":          NewSemDesc("adb_power_module_state", NoSempV2Ready, "ADB Power Module State, -1,0-Not OK 1-OK", nil),
		"adb_mate_link_port1_state":       NewSemDesc("adb_mate_link_port1_state", NoSempV2Ready, "ADB Matelink Port 1 State, 0-Loss of Sync 1-OK, 2-No SFP Module, 3-No Data", nil),
		"adb_mate_link_port2_state":       NewSemDesc("adb_mate_link_port2_state", NoSempV2Ready, "ADB Matelink Port 2 State, 0-Loss of Sync 1-OK, 2-No SFP Module, 3-No Data", nil),
	},
	//SEMPv1: show replication stats
	"ReplicationStats": {
		//Active stats
		//Message processing
		"system_replication_bridge_admin_state":                   NewSemDesc("system_replication_bridge_admin_state", NoSempV2Ready, "Replication Config Sync Bridge Admin State", variableLabelsReplication),
		"system_replication_bridge_state":                         NewSemDesc("system_replication_bridge_state", NoSempV2Ready, "Replication Config Sync Bridge State", variableLabelsReplication),
		"system_replication_sync_msgs_queued_to_standby":          NewSemDesc("system_replication_sync_msgs_queued_to_standby", NoSempV2Ready, "Replication sync messages queued to standby", variableLabelsReplication),
		"system_replication_sync_msgs_queued_to_standby_as_async": NewSemDesc("system_replication_sync_msgs_queued_to_standby_as_async", NoSempV2Ready, "Replication sync messages queued to standby as Async", variableLabelsReplication),
		"system_replication_async_msgs_queued_to_standby":         NewSemDesc("system_replication_async_msgs_queued_to_standby", NoSempV2Ready, "Replication async messages queued to standby", variableLabelsReplication),
		"system_replication_promoted_msgs_queued_to_standby":      NewSemDesc("system_replication_promoted_msgs_queued_to_standby", NoSempV2Ready, "Replication promoted messages queued to standby", variableLabelsReplication),
		"system_replication_pruned_locally_consumed_msgs":         NewSemDesc("system_replication_pruned_locally_consumed_msgs", NoSempV2Ready, "Replication Pruned locally consumed messages", variableLabelsReplication),
		//Sync replication
		"system_replication_transitions_to_ineligible": NewSemDesc("system_replication_transitions_to_ineligible", NoSempV2Ready, "Replication transitions to ineligible", variableLabelsReplication),
		//Ack propagation
		"system_replication_msgs_tx_to_standby":   NewSemDesc("system_replication_msgs_tx_to_standby", NoSempV2Ready, "system_replication_msgs_tx_to_standby", variableLabelsReplication),
		"system_replication_rec_req_from_standby": NewSemDesc("system_replication_rec_req_from_standby", NoSempV2Ready, "system_replication_rec_req_from_standby", variableLabelsReplication),
		//Standby stats
		//Message processing
		"system_replication_msgs_rx_from_active": NewSemDesc("system_replication_msgs_rx_from_active", NoSempV2Ready, "Replication msgs rx from active", variableLabelsReplication),
		//Ack propagation
		"system_replication_ack_prop_msgs_rx": NewSemDesc("system_replication_ack_prop_msgs_rx", NoSempV2Ready, "Replication ack prop msgs rx", variableLabelsReplication),
		"system_replication_recon_req_tx":     NewSemDesc("system_replication_recon_req_tx", NoSempV2Ready, "Replication recon req tx", variableLabelsReplication),
		"system_replication_out_of_seq_rx":    NewSemDesc("system_replication_out_of_seq_rx", NoSempV2Ready, "Replication out of seq rx", variableLabelsReplication),
		//Transaction replication
		"system_replication_xa_req":                  NewSemDesc("system_replication_xa_req", NoSempV2Ready, "Replication transaction requests", variableLabelsReplication),
		"system_replication_xa_req_success":          NewSemDesc("system_replication_xa_req_success", NoSempV2Ready, "Replication transaction requests success", variableLabelsReplication),
		"system_replication_xa_req_success_prepare":  NewSemDesc("system_replication_xa_req_success_prepare", NoSempV2Ready, "Replication transaction requests success prepare", variableLabelsReplication),
		"system_replication_xa_req_success_commit":   NewSemDesc("system_replication_xa_req_success_commit", NoSempV2Ready, "Replication transaction requests success commit", variableLabelsReplication),
		"system_replication_xa_req_success_rollback": NewSemDesc("system_replication_xa_req_success_rollback", NoSempV2Ready, "Replication transaction requests success rollback", variableLabelsReplication),
		"system_replication_xa_req_fail":             NewSemDesc("system_replication_xa_req_fail", NoSempV2Ready, "Replication transaction requests fail", variableLabelsReplication),
		"system_replication_xa_req_fail_prepare":     NewSemDesc("system_replication_xa_req_fail_prepare", NoSempV2Ready, "Replication transaction requests fail prepare", variableLabelsReplication),
		"system_replication_xa_req_fail_commit":      NewSemDesc("system_replication_xa_req_fail_commit", NoSempV2Ready, "Replication transaction requests fail commit", variableLabelsReplication),
		"system_replication_xa_req_fail_rollback":    NewSemDesc("system_replication_xa_req_fail_rollback", NoSempV2Ready, "Replication transaction requests fail rollback", variableLabelsReplication),
	},
	"Vpn": {
		"vpn_is_management_vpn":                 NewSemDesc("vpn_is_management_vpn", NoSempV2Ready, "VPN is a management VPN", variableLabelsVpn),
		"vpn_enabled":                           NewSemDesc("vpn_enabled", NoSempV2Ready, "VPN is enabled", variableLabelsVpn),
		"vpn_operational":                       NewSemDesc("vpn_operational", NoSempV2Ready, "VPN is operational", variableLabelsVpn),
		"vpn_locally_configured":                NewSemDesc("vpn_locally_configured", NoSempV2Ready, "VPN is locally configured", variableLabelsVpn),
		"vpn_local_status":                      NewSemDesc("vpn_local_status", NoSempV2Ready, "Local status (0=Down, 1=Up)", variableLabelsVpn),
		"vpn_unique_subscriptions":              NewSemDesc("vpn_unique_subscriptions", NoSempV2Ready, "Total subscriptions count", variableLabelsVpn),
		"vpn_total_local_unique_subscriptions":  NewSemDesc("vpn_total_local_unique_subscriptions", NoSempV2Ready, "Total unique local subscriptions count", variableLabelsVpn),
		"vpn_total_remote_unique_subscriptions": NewSemDesc("vpn_total_remote_unique_subscriptions", NoSempV2Ready, "Total unique remote subscriptions count", variableLabelsVpn),
		"vpn_total_unique_subscriptions":        NewSemDesc("vpn_total_unique_subscriptions", NoSempV2Ready, "Total unique subscriptions count", variableLabelsVpn),
	},
	"VpnReplication": {
		"vpn_replication_admin_state":                  NewSemDesc("vpn_replication_admin_state", NoSempV2Ready, "Replication Admin Status (0-shutdown, 1-enabled, 2-n/a)", variableLabelsVpn),
		"vpn_replication_config_state":                 NewSemDesc("vpn_replication_config_state", NoSempV2Ready, "Replication Config Status (0-standby, 1-active, 2-n/a)", variableLabelsVpn),
		"vpn_replication_transaction_replication_mode": NewSemDesc("vpn_replication_transaction_replication_mode", NoSempV2Ready, "Replication Transaction Replication Mode (0-async, 1-sync)", variableLabelsVpn),
	},
	"ConfigSync": {
		"configsync_admin_state": NewSemDesc("configsync_admin_state", NoSempV2Ready, "Config Sync Admin Status (0-Shutdown, 1-Enabled)", nil),
		"configsync_oper_state":  NewSemDesc("configsync_operational_state", NoSempV2Ready, "Config Sync Current Status (0-Down, 1-Up, 2-Shutting Down)", nil),
	},
	"ConfigSyncVpn": {
		"configsync_table_type":               NewSemDesc("configsync_table_type", NoSempV2Ready, "Config Sync Resource Type (0-Router, 1-Vpn, 2-Unknown, 3-None, 4-All)", variableLabelsConfigSyncTable),
		"configsync_table_timeinstateseconds": NewSemDesc("configsync_table_timeinstateseconds", NoSempV2Ready, "Config Sync Time in State", variableLabelsConfigSyncTable),
		"configsync_table_ownership":          NewSemDesc("configsync_table_ownership", NoSempV2Ready, "Config Sync Ownership (0-Master, 1-Slave, 2-Unknown)", variableLabelsConfigSyncTable),
		"configsync_table_syncstate":          NewSemDesc("configsync_table_syncstate", NoSempV2Ready, "Config Sync State (0-Down, 1-Up, 2-Unknown, 3-In-Sync, 4-Reconciling, 5-Blocked, 6-Out-Of-Sync)", variableLabelsConfigSyncTable),
	},
	"ConfigSyncRouter": {
		"configsync_table_type":               NewSemDesc("configsync_table_type", NoSempV2Ready, "Config Sync Resource Type (0-Router, 1-Vpn, 2-Unknown, 3-None, 4-All)", variableLabelsConfigSyncTable),
		"configsync_table_timeinstateseconds": NewSemDesc("configsync_table_timeinstateseconds", NoSempV2Ready, "Config Sync Time in State", variableLabelsConfigSyncTable),
		"configsync_table_ownership":          NewSemDesc("configsync_table_ownership", NoSempV2Ready, "Config Sync Ownership (0-Master, 1-Slave, 2-Unknown)", variableLabelsConfigSyncTable),
		"configsync_table_syncstate":          NewSemDesc("configsync_table_syncstate", NoSempV2Ready, "Config Sync State (0-Down, 1-Up, 2-Unknown, 3-In-Sync, 4-Reconciling, 5-Blocked, 6-Out-Of-Sync)", variableLabelsConfigSyncTable),
	},
	"Bridge": {
		"bridges_num_total_bridges":                         NewSemDesc("bridges_num_total_bridges", NoSempV2Ready, "Number of Bridges", nil),
		"bridges_max_num_total_bridges":                     NewSemDesc("bridges_max_num_total_bridges", NoSempV2Ready, "Max number of Bridges", nil),
		"bridges_num_local_bridges":                         NewSemDesc("bridges_num_local_bridges", NoSempV2Ready, "Number of Local Bridges", nil),
		"bridges_max_num_local_bridges":                     NewSemDesc("bridges_max_num_local_bridges", NoSempV2Ready, "Max number of Local Bridges", nil),
		"bridges_num_remote_bridges":                        NewSemDesc("bridges_num_remote_bridges", NoSempV2Ready, "Number of Remote Bridges", nil),
		"bridges_max_num_remote_bridges":                    NewSemDesc("bridges_max_num_remote_bridges", NoSempV2Ready, "Max number of Remote Bridges", nil),
		"bridges_num_total_remote_bridge_subscriptions":     NewSemDesc("bridges_num_total_remote_bridge_subscriptions", NoSempV2Ready, "Total number of Remote Bridge Subscription", nil),
		"bridges_max_num_total_remote_bridge_subscriptions": NewSemDesc("bridges_max_num_total_remote_bridge_subscriptions", NoSempV2Ready, "Max total number of Remote Bridge Subscription", nil),
		"bridge_admin_state":                                NewSemDesc("bridge_admin_state", NoSempV2Ready, "Bridge Administrative State (0-Enabled 1-Disabled, 2--)", variableLabelsBridge),
		"bridge_connection_establisher":                     NewSemDesc("bridge_connection_establisher", NoSempV2Ready, "Connection Establisher (0-NotApplicable, 1-Local, 2-Remote, 3-Invalid)", variableLabelsBridge),
		"bridge_inbound_operational_state":                  NewSemDesc("bridge_inbound_operational_state", NoSempV2Ready, "Inbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge),
		"bridge_inbound_operational_failure_reason":         NewSemDesc("bridge_inbound_operational_failure_reason", NoSempV2Ready, "Inbound Ops Failure Reason (various very long codes)", variableLabelsBridge),
		"bridge_outbound_operational_state":                 NewSemDesc("bridge_outbound_operational_state", NoSempV2Ready, "Outbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge),
		"bridge_queue_operational_state":                    NewSemDesc("bridge_queue_operational_state", NoSempV2Ready, "Queue Ops State (0-NotApplicable, 1-Bound, 2-Unbound)", variableLabelsBridge),
		"bridge_redundancy":                                 NewSemDesc("bridge_redundancy", NoSempV2Ready, "Bridge Redundancy (0-NotApplicable, 1-auto, 2-primary, 3-backup, 4-static, 5-none)", variableLabelsBridge),
		"bridge_connection_uptime_in_seconds":               NewSemDesc("bridge_connection_uptime_in_seconds", NoSempV2Ready, "Connection Uptime (s)", variableLabelsBridge),
	},
	"BridgeRemote": {
		"bridge_admin_state":                        NewSemDesc("bridge_admin_state", NoSempV2Ready, "Bridge Administrative State (0-Enabled 1-Disabled, 2--, 3-N/A)", variableLabelsBridgeRemote),
		"bridge_connection_establisher":             NewSemDesc("bridge_connection_establisher", NoSempV2Ready, "Connection Establisher (0-NotApplicable, 1-Local, 2-Remote, 3-Invalid)", variableLabelsBridgeRemote),
		"bridge_inbound_operational_state":          NewSemDesc("bridge_inbound_operational_state", NoSempV2Ready, "Inbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridgeRemote),
		"bridge_inbound_operational_failure_reason": NewSemDesc("bridge_inbound_operational_failure_reason", NoSempV2Ready, "Inbound Ops Failure Reason (0-Bridge disabled ,1-No remote message-vpns configured, 2-SMF service is disabled, 3-Msg Backbone is disabled, 4-Local message-vpn is disabled, 5-Active-Standby Role Mismatch, 6-Invalid Active-Standby Role, 7-Redundancy Disabled, 8-Not active, 9-Replication standby, 10-Remote message-vpns disabled, 11-Enforce-trusted-common-name but empty trust-common-name list, 12-SSL transport used but cipher-suite list is empty, 13-Authentication Scheme is Client-Certificate but no certificate is configured, 14-Client-Certificate Authentication Scheme used but not all Remote Message VPNs use SSL, 15-Basic Authentication Scheme used but Basic Client Username not configured, 16-Cluster Down, 17-Cluster Link Down, 18-N/A)", variableLabelsBridgeRemote),
		"bridge_outbound_operational_state":         NewSemDesc("bridge_outbound_operational_state", NoSempV2Ready, "Outbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridgeRemote),
		"bridge_queue_operational_state":            NewSemDesc("bridge_queue_operational_state", NoSempV2Ready, "Queue Ops State (0-NotApplicable, 1-Bound, 2-Unbound)", variableLabelsBridgeRemote),
		"bridge_redundancy":                         NewSemDesc("bridge_redundancy", NoSempV2Ready, "Bridge Redundancy (0-NotApplicable, 1-auto, 2-primary, 3-backup, 4-static, 5-none)", variableLabelsBridgeRemote),
		"bridge_connection_uptime_in_seconds":       NewSemDesc("bridge_connection_uptime_in_seconds", NoSempV2Ready, "Connection Uptime (s)", variableLabelsBridgeRemote),
	},
	"VpnSpool": {
		"vpn_spool_quota_bytes":                 NewSemDesc("vpn_spool_quota_bytes", NoSempV2Ready, "Spool configured max disk usage.", variableLabelsVpn),
		"vpn_spool_usage_bytes":                 NewSemDesc("vpn_spool_usage_bytes", NoSempV2Ready, "Spool total persisted usage.", variableLabelsVpn),
		"vpn_spool_usage_pct":                   NewSemDesc("vpn_spool_usage_pct", NoSempV2Ready, "Spool percentage persisted usage. (-1 means no spool has been allocated.)", variableLabelsVpn),
		"vpn_spool_usage_msgs":                  NewSemDesc("vpn_spool_usage_msgs", NoSempV2Ready, "Spool total number of persisted messages.", variableLabelsVpn),
		"vpn_spool_current_endpoints":           NewSemDesc("vpn_spool_current_endpoints", NoSempV2Ready, "Spool current number of endpoints.", variableLabelsVpn),
		"vpn_spool_maximum_endpoints":           NewSemDesc("vpn_spool_maximum_endpoints", NoSempV2Ready, "Spool maximum number of endpoints.", variableLabelsVpn),
		"vpn_spool_current_egress_flows":        NewSemDesc("vpn_spool_current_egress_flows", NoSempV2Ready, "Spool current number of egress flows.", variableLabelsVpn),
		"vpn_spool_maximum_egress_flows":        NewSemDesc("vpn_spool_maximum_egress_flows", NoSempV2Ready, "Spool maximum number of egress flows.", variableLabelsVpn),
		"vpn_spool_current_ingress_flows":       NewSemDesc("vpn_spool_current_ingress_flows", NoSempV2Ready, "Spool current number of ingress flows.", variableLabelsVpn),
		"vpn_spool_maximum_ingress_flows":       NewSemDesc("vpn_spool_maximum_ingress_flows", NoSempV2Ready, "Spool maximum number of ingress flows.", variableLabelsVpn),
		"vpn_spool_current_transacted_sessions": NewSemDesc("vpn_spool_current_transacted_sessions", NoSempV2Ready, "Spool current number of transacted sessions.", variableLabelsVpn),
		"vpn_spool_current_transacted_msgs":     NewSemDesc("vpn_spool_current_transacted_msgs", NoSempV2Ready, "Spool current number of transacted messages.", variableLabelsVpn),
	},
	//SEMPv1: show client <client-name> message-vpn <vpn-name> connected
	"Client": {
		"client_num_subscriptions": NewSemDesc("client_num_subscriptions", NoSempV2Ready, "Number of client subscriptions.", variableLabelsClientInfo),
	},
	//SEMPv1: show client <client-name> message-vpn <vpn-name> connected
	"ClientSlowSubscriber": {
		"client_slow_subscriber": NewSemDesc("client_slow_subscriber", NoSempV2Ready, "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsClientInfo),
	},
	"ClientStats": {
		"client_rx_msgs_total":           NewSemDesc("client_rx_msgs_total", NoSempV2Ready, "Number of received messages.", variableLabelsVpnClientUser),
		"client_tx_msgs_total":           NewSemDesc("client_tx_msgs_total", NoSempV2Ready, "Number of transmitted messages.", variableLabelsVpnClientUser),
		"client_rx_bytes_total":          NewSemDesc("client_rx_bytes_total", NoSempV2Ready, "Number of received bytes.", variableLabelsVpnClientUser),
		"client_tx_bytes_total":          NewSemDesc("client_tx_bytes_total", NoSempV2Ready, "Number of transmitted bytes.", variableLabelsVpnClientUser),
		"client_rx_discarded_msgs_total": NewSemDesc("client_rx_discarded_msgs_total", NoSempV2Ready, "Number of discarded received messages.", variableLabelsVpnClientUser),
		"client_tx_discarded_msgs_total": NewSemDesc("client_tx_discarded_msgs_total", NoSempV2Ready, "Number of discarded transmitted messages.", variableLabelsVpnClientUser),
		"client_slow_subscriber":         NewSemDesc("client_slow_subscriber", NoSempV2Ready, "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClientUser),
	},
	"ClientMessageSpoolStats": {
		"client_flows_ingress": NewSemDesc("client_flows_ingress", NoSempV2Ready, "Number of ingress flows, created/openend by this client.", variableLabelsVpnClientDetail),
		"client_flows_egress":  NewSemDesc("client_flows_egress", NoSempV2Ready, "Number of egress flows, created/openend by this client.", variableLabelsVpnClientDetail),

		"spooling_not_ready":                NewSemDesc("client_ingress_spooling_not_ready", NoSempV2Ready, "Number of connections closed caused by spoolingNotReady", variableLabelsVpnClientFlow),
		"out_of_order_messages_received":    NewSemDesc("client_ingress_out_of_order_messages_received", NoSempV2Ready, "Number of messages, received in wrong order.", variableLabelsVpnClientFlow),
		"duplicate_messages_received":       NewSemDesc("client_ingress_duplicate_messages_received", NoSempV2Ready, "Number of messages, received more than once", variableLabelsVpnClientFlow),
		"no_eligible_destinations":          NewSemDesc("client_ingress_no_eligible_destinations", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"guaranteed_messages":               NewSemDesc("client_ingress_guaranteed_messages", NoSempV2Ready, "Number of guarantied messages, received.", variableLabelsVpnClientFlow),
		"no_local_delivery":                 NewSemDesc("client_ingress__no_local_delivery", NoSempV2Ready, "Number of messages, no localy delivered.", variableLabelsVpnClientFlow),
		"seq_num_rollover":                  NewSemDesc("client_ingress_seq_num_rollover", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"seq_num_messages_discarded":        NewSemDesc("client_ingress_seq_num_messages_discarded", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"transacted_messages_not_sequenced": NewSemDesc("client_ingress_transacted_messages_not_sequenced", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"destination_group_error":           NewSemDesc("client_ingress_destination_group_error", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"smf_ttl_exceeded":                  NewSemDesc("client_ingress_smf_ttl_exceeded", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"publish_acl_denied":                NewSemDesc("client_ingress_publish_acl_denied", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"ingress_window_size":               NewSemDesc("client_ingress_window_size", NoSempV2Ready, "Configured window size", variableLabelsVpnClientFlow),

		"egress_window_size":                    NewSemDesc("client_egress_window_size", NoSempV2Ready, "Configured window size", variableLabelsVpnClientFlow),
		"used_window":                           NewSemDesc("client_egress_used_window", NoSempV2Ready, "Used windows size.", variableLabelsVpnClientFlow),
		"window_closed":                         NewSemDesc("client_egress_window_closed", NoSempV2Ready, "Number windows closed.", variableLabelsVpnClientFlow),
		"message_redelivered":                   NewSemDesc("client_egress_message_redelivered", NoSempV2Ready, "Number of messages, was been redelivered.", variableLabelsVpnClientFlow),
		"message_transport_retransmit":          NewSemDesc("client_egress_message_transport_retransmit", NoSempV2Ready, "Number of messages, was been retransmitted.", variableLabelsVpnClientFlow),
		"message_confirmed_delivered":           NewSemDesc("client_egress_message_confirmed_delivered", NoSempV2Ready, "Number of messages succesfully delivered.", variableLabelsVpnClientFlow),
		"confirmed_delivered_store_and_forward": NewSemDesc("client_egress_confirmed_delivered_store_and_forward", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"confirmed_delivered_cut_through":       NewSemDesc("client_egress_confirmed_delivered_cut_through", NoSempV2Ready, "???", variableLabelsVpnClientFlow),
		"unacked_messages":                      NewSemDesc("client_egress_unacked_messages", NoSempV2Ready, "Number of unacknowledged messages.", variableLabelsVpnClientFlow),
	},
	"VpnStats": {
		"vpn_rx_msgs_total":                NewSemDesc("vpn_rx_msgs_total", NoSempV2Ready, "Number of received messages.", variableLabelsVpn),
		"vpn_tx_msgs_total":                NewSemDesc("vpn_tx_msgs_total", NoSempV2Ready, "Number of transmitted messages.", variableLabelsVpn),
		"vpn_rx_bytes_total":               NewSemDesc("vpn_rx_bytes_total", NoSempV2Ready, "Number of received bytes.", variableLabelsVpn),
		"vpn_tx_bytes_total":               NewSemDesc("vpn_tx_bytes_total", NoSempV2Ready, "Number of transmitted bytes.", variableLabelsVpn),
		"vpn_rx_discarded_msgs_total":      NewSemDesc("vpn_rx_discarded_msgs_total", NoSempV2Ready, "Number of discarded received messages.", variableLabelsVpn),
		"vpn_tx_discarded_msgs_total":      NewSemDesc("vpn_tx_discarded_msgs_total", NoSempV2Ready, "Number of discarded transmitted messages.", variableLabelsVpn),
		"vpn_connections_service_amqp":     NewSemDesc("vpn_connections_service_amqp", NoSempV2Ready, "Total number of amq connections", variableLabelsVpn),
		"vpn_connections_service_mqtt":     NewSemDesc("vpn_connections_service_mqtt", NoSempV2Ready, "Total number of mqtt connections", variableLabelsVpn),
		"vpn_connections_service_smf":      NewSemDesc("vpn_connections_service_smf", NoSempV2Ready, "Total number of smf connections", variableLabelsVpn),
		"vpn_connections_service_web":      NewSemDesc("vpn_connections_service_web", NoSempV2Ready, "Total number of smf-web connections", variableLabelsVpn),
		"vpn_connections_service_rest_in":  NewSemDesc("vpn_connections_service_rest_in", NoSempV2Ready, "Total number of inbound rest connections", variableLabelsVpn),
		"vpn_connections_service_rest_out": NewSemDesc("vpn_connections_service_rest_out", NoSempV2Ready, "Total number of outbound rest connections", variableLabelsVpn),
		"vpn_connections":                  NewSemDesc("vpn_connections", NoSempV2Ready, "Number of connections.", variableLabelsVpn),
		"vpn_quota_connections":            NewSemDesc("vpn_quota_connections", NoSempV2Ready, "Maximum number of connections.", variableLabelsVpn),
		"vpn_quota_connections_amqp":       NewSemDesc("vpn_quota_connections_amqp", NoSempV2Ready, "Maximum number of amqp connections.", variableLabelsVpn),
		"vpn_quota_connections_smf":        NewSemDesc("vpn_quota_connections_smf", NoSempV2Ready, "Maximum number of smf connections.", variableLabelsVpn),
		"vpn_quota_connections_web":        NewSemDesc("vpn_quota_connections_web", NoSempV2Ready, "Maximum number of smf-web connections.", variableLabelsVpn),
		"vpn_quota_connections_mqtt":       NewSemDesc("vpn_quota_connections_mqtt", NoSempV2Ready, "Maximum number of mqtt connections.", variableLabelsVpn),
		"vpn_quota_connections_rest_in":    NewSemDesc("vpn_quota_connections_rest_in", NoSempV2Ready, "Maximum number of inbound rest connections.", variableLabelsVpn),
		"vpn_quota_connections_rest_out":   NewSemDesc("vpn_quota_connections_rest_out", NoSempV2Ready, "Maximum number of outbound rest connections.", variableLabelsVpn),
	},
	"BridgeStats": {
		"bridge_client_num_subscriptions":               NewSemDesc("bridge_client_num_subscriptions", NoSempV2Ready, "Bridge Client Subscription", variableLabelsBridgeStats),
		"bridge_client_slow_subscriber":                 NewSemDesc("bridge_client_slow_subscriber", NoSempV2Ready, "Bridge Slow Subscriber", variableLabelsBridgeStats),
		"bridge_total_client_messages_received":         NewSemDesc("bridge_total_client_messages_received", NoSempV2Ready, "Bridge Total Client Messages Received", variableLabelsBridgeStats),
		"bridge_total_client_messages_sent":             NewSemDesc("bridge_total_client_messages_sent", NoSempV2Ready, "Bridge Total Client Messages sent", variableLabelsBridgeStats),
		"bridge_client_data_messages_received":          NewSemDesc("bridge_client_data_messages_received", NoSempV2Ready, "Bridge Client Data Msgs Received", variableLabelsBridgeStats),
		"bridge_client_data_messages_sent":              NewSemDesc("bridge_client_data_messages_sent", NoSempV2Ready, "Bridge Client Data Msgs Sent", variableLabelsBridgeStats),
		"bridge_client_persistent_messages_received":    NewSemDesc("bridge_client_persistent_messages_received", NoSempV2Ready, "Bridge Client Persistent Msgs Received", variableLabelsBridgeStats),
		"bridge_client_persistent_messages_sent":        NewSemDesc("bridge_client_persistent_messages_sent", NoSempV2Ready, "Bridge Client Persistent Msgs Sent", variableLabelsBridgeStats),
		"bridge_client_nonpersistent_messages_received": NewSemDesc("bridge_client_nonpersistent_messages_received", NoSempV2Ready, "Bridge Client Non-Persistent Msgs Received", variableLabelsBridgeStats),
		"bridge_client_nonpersistent_messages_sent":     NewSemDesc("bridge_client_nonpersistent_messages_sent", NoSempV2Ready, "Bridge Client Non-Persistent Msgs Sent", variableLabelsBridgeStats),
		"bridge_client_direct_messages_received":        NewSemDesc("bridge_client_direct_messages_received", NoSempV2Ready, "Bridge Client Direct Msgs Received", variableLabelsBridgeStats),
		"bridge_client_direct_messages_sent":            NewSemDesc("bridge_client_direct_messages_sent", NoSempV2Ready, "Bridge Client Direct Msgs Sent", variableLabelsBridgeStats),
		"bridge_total_client_bytes_received":            NewSemDesc("bridge_total_client_bytes_received", NoSempV2Ready, "Bridge Total Client Bytes Received", variableLabelsBridgeStats),
		"bridge_total_client_bytes_sent":                NewSemDesc("bridge_total_client_bytes_sent", NoSempV2Ready, "Bridge Total Client Bytes sent", variableLabelsBridgeStats),
		"bridge_client_data_bytes_received":             NewSemDesc("bridge_client_data_bytes_received", NoSempV2Ready, "Bridge Client Data Bytes Received", variableLabelsBridgeStats),
		"bridge_client_data_bytes_sent":                 NewSemDesc("bridge_client_data_bytes_sent", NoSempV2Ready, "Bridge Client Data Bytes Sent", variableLabelsBridgeStats),
		"bridge_client_persistent_bytes_received":       NewSemDesc("bridge_client_persistent_bytes_received", NoSempV2Ready, "Bridge Client Persistent Bytes Received", variableLabelsBridgeStats),
		"bridge_client_persistent_bytes_sent":           NewSemDesc("bridge_client_persistent_bytes_sent", NoSempV2Ready, "Bridge Client Persistent Bytes Sent", variableLabelsBridgeStats),
		"bridge_client_nonpersistent_bytes_received":    NewSemDesc("bridge_client_nonpersistent_bytes_received", NoSempV2Ready, "Bridge Client Non-Persistent Bytes Received", variableLabelsBridgeStats),
		"bridge_client_nonpersistent_bytes_sent":        NewSemDesc("bridge_client_nonpersistent_bytes_sent", NoSempV2Ready, "Bridge Client Non-Persistent Bytes Sent", variableLabelsBridgeStats),
		"bridge_client_direct_bytes_received":           NewSemDesc("bridge_client_direct_bytes_received", NoSempV2Ready, "Bridge Client Direct Bytes Received", variableLabelsBridgeStats),
		"bridge_client_direct_bytes_sent":               NewSemDesc("bridge_client_direct_bytes_sent", NoSempV2Ready, "Bridge Client Direct Bytes Sent", variableLabelsBridgeStats),
		"bridge_client_large_messages_received":         NewSemDesc("bridge_client_large_messages_received", NoSempV2Ready, "Bridge Client Large Messages received", variableLabelsBridgeStats),
		"bridge_denied_duplicate_clients":               NewSemDesc("bridge_denied_duplicate_clients", NoSempV2Ready, "Bridge Denied Duplicate Clients", variableLabelsBridgeStats),
		"bridge_not_enough_space_msgs_sent":             NewSemDesc("bridge_not_enough_space_msgs_sent", NoSempV2Ready, "Bridge Not Enough Space Messages Sent", variableLabelsBridgeStats),
		"bridge_max_exceeded_msgs_sent":                 NewSemDesc("bridge_max_exceeded_msgs_sent", NoSempV2Ready, "Bridge Max Exceeded Messages Sent", variableLabelsBridgeStats),
		"bridge_subscribe_client_not_found":             NewSemDesc("bridge_subscribe_client_not_found", NoSempV2Ready, "Bridge Subscriber Client Not Found", variableLabelsBridgeStats),
		"bridge_not_found_msgs_sent":                    NewSemDesc("bridge_not_found_msgs_sent", NoSempV2Ready, "Bridge Not Found Messages Sent", variableLabelsBridgeStats),
		"bridge_current_ingress_rate_per_second":        NewSemDesc("bridge_current_ingress_rate_per_second", NoSempV2Ready, "Current Ingress Rate / s", variableLabelsBridgeStats),
		"bridge_current_egress_rate_per_second":         NewSemDesc("bridge_current_egress_rate_per_second", NoSempV2Ready, "Current Egress Rate / s", variableLabelsBridgeStats),
		"bridge_total_ingress_discards":                 NewSemDesc("bridge_total_ingress_discards", NoSempV2Ready, "Total Ingress Discards", variableLabelsBridgeStats),
		"bridge_total_egress_discards":                  NewSemDesc("bridge_total_egress_discards", NoSempV2Ready, "Total Egress Discards", variableLabelsBridgeStats),
	},
	"QueueRates": {
		"queue_rx_msg_rate":      NewSemDesc("queue_rx_msg_rate", NoSempV2Ready, "Rate of received messages.", variableLabelsVpnQueue),
		"queue_tx_msg_rate":      NewSemDesc("queue_tx_msg_rate", NoSempV2Ready, "Rate of transmitted messages.", variableLabelsVpnQueue),
		"queue_rx_byte_rate":     NewSemDesc("queue_rx_byte_rate", NoSempV2Ready, "Rate of received bytes.", variableLabelsVpnQueue),
		"queue_tx_byte_rate":     NewSemDesc("queue_tx_byte_rate", NoSempV2Ready, "Rate of transmitted bytes.", variableLabelsVpnQueue),
		"queue_rx_msg_rate_avg":  NewSemDesc("queue_rx_msg_rate_avg", NoSempV2Ready, "Average rate of received messages.", variableLabelsVpnQueue),
		"queue_tx_msg_rate_avg":  NewSemDesc("queue_tx_msg_rate_avg", NoSempV2Ready, "Average rate of transmitted messages.", variableLabelsVpnQueue),
		"queue_rx_byte_rate_avg": NewSemDesc("queue_rx_byte_rate_avg", NoSempV2Ready, "Average rate of received bytes.", variableLabelsVpnQueue),
		"queue_tx_byte_rate_avg": NewSemDesc("queue_tx_byte_rate_avg", NoSempV2Ready, "Average rate of transmitted bytes.", variableLabelsVpnQueue),
	},
	"QueueDetails": {
		"queue_spool_quota_bytes": NewSemDesc("queue_spool_quota_bytes", NoSempV2Ready, "Queue spool configured max disk usage in bytes.", variableLabelsVpnQueue),
		"queue_spool_usage_bytes": NewSemDesc("queue_spool_usage_bytes", NoSempV2Ready, "The size in bytes of all messages currently in the Queue.", variableLabelsVpnQueue),
		"queue_spool_usage_msgs":  NewSemDesc("queue_spool_usage_msgs", NoSempV2Ready, "The count of all messages currently in the Queue.", variableLabelsVpnQueue),
		"queue_binds":             NewSemDesc("queue_binds", NoSempV2Ready, "Number of clients bound to queue.", variableLabelsVpnQueue),
		"queue_subscriptions":     NewSemDesc("queue_subscriptions", NoSempV2Ready, "Number of subscriptions of the queue.", variableLabelsVpnQueue),
	},
	"QueueStats":   QueueStats,
	"QueueStatsV2": QueueStats,
	"TopicEndpointRates": {
		"rx_msg_rate":      NewSemDesc("topic_endpoint_rx_msg_rate", NoSempV2Ready, "Rate of received messages.", variableLabelsVpnTopicEndpoint),
		"tx_msg_rate":      NewSemDesc("topic_endpoint_tx_msg_rate", NoSempV2Ready, "Rate of transmitted messages.", variableLabelsVpnTopicEndpoint),
		"rx_byte_rate":     NewSemDesc("topic_endpoint_rx_byte_rate", NoSempV2Ready, "Rate of received bytes.", variableLabelsVpnTopicEndpoint),
		"tx_byte_rate":     NewSemDesc("topic_endpoint_tx_byte_rate", NoSempV2Ready, "Rate of transmitted bytes.", variableLabelsVpnTopicEndpoint),
		"rx_msg_rate_avg":  NewSemDesc("topic_endpoint_rx_msg_rate_avg", NoSempV2Ready, "Average rate of received messages.", variableLabelsVpnTopicEndpoint),
		"tx_msg_rate_avg":  NewSemDesc("topic_endpoint_tx_msg_rate_avg", NoSempV2Ready, "Average rate of transmitted messages.", variableLabelsVpnTopicEndpoint),
		"rx_byte_rate_avg": NewSemDesc("topic_endpoint_rx_byte_rate_avg", NoSempV2Ready, "Average rate of received bytes.", variableLabelsVpnTopicEndpoint),
		"tx_byte_rate_avg": NewSemDesc("topic_endpoint_tx_byte_rate_avg", NoSempV2Ready, "Average rate of transmitted bytes.", variableLabelsVpnTopicEndpoint),
	},
	"TopicEndpointDetails": {
		"spool_quota_bytes": NewSemDesc("topic_endpoint_spool_quota_bytes", NoSempV2Ready, "Topic Endpoint spool configured max disk usage in bytes.", variableLabelsVpnTopicEndpoint),
		"spool_usage_bytes": NewSemDesc("topic_endpoint_spool_usage_bytes", NoSempV2Ready, "Topic Endpoint spool usage in bytes.", variableLabelsVpnTopicEndpoint),
		"spool_usage_msgs":  NewSemDesc("topic_endpoint_spool_usage_msgs", NoSempV2Ready, "Topic Endpoint spooled number of messages.", variableLabelsVpnTopicEndpoint),
		"binds":             NewSemDesc("topic_endpoint_binds", NoSempV2Ready, "Number of clients bound to topic-endpoint.", variableLabelsVpnTopicEndpoint),
	},
	"TopicEndpointStats": {
		"total_bytes_spooled":             NewSemDesc("topic_endpoint_byte_spooled", NoSempV2Ready, "Topic Endpoint spool total of all spooled messages in bytes.", variableLabelsVpnTopicEndpoint),
		"total_messages_spooled":          NewSemDesc("topic_endpoint_msg_spooled", NoSempV2Ready, "Topic Endpoint spool total of all spooled messages.", variableLabelsVpnTopicEndpoint),
		"messages_redelivered":            NewSemDesc("topic_endpoint_msg_redelivered", NoSempV2Ready, "Topic Endpoint total msg redeliveries.", variableLabelsVpnTopicEndpoint),
		"messages_transport_retransmited": NewSemDesc("topic_endpoint_msg_retransmited", NoSempV2Ready, "Topic Endpoint total msg retransmitted on transport.", variableLabelsVpnTopicEndpoint),
		"spool_usage_exceeded":            NewSemDesc("topic_endpoint_msg_spool_usage_exceeded", NoSempV2Ready, "Topic Endpoint total number of messages exceeded the spool usage.", variableLabelsVpnTopicEndpoint),
		"max_message_size_exceeded":       NewSemDesc("topic_endpoint_msg_max_msg_size_exceeded", NoSempV2Ready, "Topic Endpoint total number of messages exceeded the max message size.", variableLabelsVpnTopicEndpoint),
		"total_deleted_messages":          NewSemDesc("topic_endpoint_msg_total_deleted", NoSempV2Ready, "Topic Endpoint total number that was deleted.", variableLabelsVpnTopicEndpoint),
	},
	"ClusterLinks": {
		"enabled":     NewSemDesc("cluster_link_enabled", NoSempV2Ready, "Cluster link is enabled.", variableLabelsCluserLink),
		"oper_up":     NewSemDesc("cluster_link_operational", NoSempV2Ready, "Cluster link is operational.", variableLabelsCluserLink),
		"oper_uptime": NewSemDesc("cluster_link_uptime", NoSempV2Ready, "Cluster link uptime in seconds.", variableLabelsCluserLink),
	},
	"ClientConnections": {
		"connection_is_zip":                 NewSemDesc("connection_is_zip", NoSempV2Ready, "Connection is zip compressed.", variableLabelsVpnClient),
		"connection_is_ssl":                 NewSemDesc("connection_is_ssl", NoSempV2Ready, "Connection is ssl encrypted.", variableLabelsVpnClient),
		"connection_receive_queue_bytes":    NewSemDesc("connection_receive_queue_bytes", NoSempV2Ready, "The number of bytes currently in the event broker receive queue for the TCP connection.", variableLabelsVpnClient),
		"connection_send_queue_bytes":       NewSemDesc("connection_send_queue_bytes", NoSempV2Ready, "The number of bytes currently in the event broker receive queue for the TCP connection.", variableLabelsVpnClient),
		"connection_receive_queue_segments": NewSemDesc("connection_receive_queue_segments", NoSempV2Ready, "The number of bytes currently queued for the client in both the client’s egress queues and the TCP send queue.", variableLabelsVpnClient),
		"connection_send_queue_segments":    NewSemDesc("connection_send_queue_segments", NoSempV2Ready, "The number of messages currently queued for the client in its egress queues.", variableLabelsVpnClient),
		"connection_maximum_segment_size":   NewSemDesc("connection_maximum_segment_size", NoSempV2Ready, "The maximum segment size (MSS) configured for the client connection. The MSS is configured in the client profile. See RFC 879 for further details.", variableLabelsVpnClient),
		"connection_sent_bytes":             NewSemDesc("connection_sent_bytes", NoSempV2Ready, "The number of bytes sent by the event broker on the TCP connection", variableLabelsVpnClient),
		"connection_received_bytes":         NewSemDesc("connection_received_bytes", NoSempV2Ready, "The number of bytes received by the event broker on the TCP connection", variableLabelsVpnClient),

		"connection_retransmit_milliseconds":     NewSemDesc("connection_retransmit_milliseconds", NoSempV2Ready, "The retransmission timeout (RTO) in milliseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient),
		"connection_roundtrip_smth_microseconds": NewSemDesc("connection_roundtrip_smth_microseconds", NoSempV2Ready, "The smoothed round-trip time (SRTT) in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient),
		"connection_roundtrip_min_microseconds":  NewSemDesc("connection_roundtrip_min_microseconds", NoSempV2Ready, "The minimum round-trip time in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient),
		"connection_roundtrip_var_microseconds":  NewSemDesc("connection_roundtrip_var_microseconds", NoSempV2Ready, "The round-trip time variation (RTTVAR) in microseconds for the TCP connection. See RFC 2988 for further details.", variableLabelsVpnClient),

		"connection_advertised_window": NewSemDesc("connection_advertised_window", NoSempV2Ready, "The receive window size in bytes advertised to the client on the remote end of the TCP connection. See RFC 793 for further details.", variableLabelsVpnClient),
		"connection_transmit_window":   NewSemDesc("connection_transmit_window", NoSempV2Ready, "The send window size in bytes. See RFC 793 for further details.", variableLabelsVpnClient),
		"connection_congestion_window": NewSemDesc("connection_congestion_window", NoSempV2Ready, "The congestion window size in bytes (cwnd). See RFC 5681 for further details.", variableLabelsVpnClient),

		"connection_slow_start_threshold": NewSemDesc("connection_slow_start_threshold", NoSempV2Ready, "The slow start threshold in bytes (ssthresh). See RFC 5681 for further details.", variableLabelsVpnClient),
		"connection_received_outoforder":  NewSemDesc("connection_received_outoforder", NoSempV2Ready, "The number of TCP segments received out of order.", variableLabelsVpnClient),
		"connection_fast_retransmit":      NewSemDesc("connection_fast_retransmit", NoSempV2Ready, "The number of TCP segments retransmitted due to the receipt of duplicate acknowledgments (‘ACKs’). See RFC 5681 for further details.", variableLabelsVpnClient),
		"connection_timed_retransmit":     NewSemDesc("connection_timed_retransmit", NoSempV2Ready, "The number of TCP segments re-transmitted due to timeout awaiting an ACK. See RFC 793 for further details.", variableLabelsVpnClient),
	},
}
