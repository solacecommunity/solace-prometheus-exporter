// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/ini.v1"
)

const (
	namespace = "solace" // For Prometheus metrics.
)

var (
	solaceExporterVersion = float64(1004005)

	variableLabelsUp               = []string{"error"}
	variableLabelsRedundancy       = []string{"mate_name"}
	variableLabelsReplication      = []string{"mate_name"}
	variableLabelsVpn              = []string{"vpn_name"}
	variableLabelsClientInfo       = []string{"vpn_name", "client_name", "client_address"}
	variableLabelsVpnClient        = []string{"vpn_name", "client_name", "client_username"}
	variableLabelsVpnClientDetail  = []string{"vpn_name", "client_name", "client_username", "client_profile", "acl_profile"}
	variableLabelsVpnClientFlow    = []string{"vpn_name", "client_name", "client_username", "client_profile", "acl_profile", "flow_id"}
	variableLabelsVpnQueue         = []string{"vpn_name", "queue_name"}
	variableLabelsVpnTopicEndpoint = []string{"vpn_name", "topic_endpoint_name"}
	variableLabelsCluserLink       = []string{"cluster", "node_name", "remote_cluster", "remote_node_name"}
	variableLabelsBridge           = []string{"vpn_name", "bridge_name"}
	variableLabelsConfigSyncTable  = []string{"table_name"}
	variableLabelsStorageElement   = []string{"path", "device_name", "element_name"}
	variableLabelsDisk             = []string{"path", "device_name"}
	variableLabelsInterface        = []string{"interface_name"}
)

type Metrics map[string]*prometheus.Desc

// Redirect callback, re-insert basic auth string into header
func (e *Exporter) redirectPolicyFunc(req *http.Request, _ []*http.Request) error {
	req.SetBasicAuth(e.config.username, e.config.password)
	return nil
}

// Call http post for the supplied uri and body
func (e *Exporter) postHTTP(uri string, _ string, body string) (io.ReadCloser, error) {
	var proxy func(req *http.Request) (*url.URL, error)
	if e.config.useSystemProxy {
		proxy = http.ProxyFromEnvironment
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !e.config.sslVerify}, Proxy: proxy}
	client := http.Client{
		Timeout:       e.config.timeout,
		Transport:     tr,
		CheckRedirect: e.redirectPolicyFunc,
	}

	req, err := http.NewRequest("GET", uri, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(e.config.username, e.config.password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}

var metricDesc = map[string]Metrics{
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
		"system_spool_usage_bytes":                         prometheus.NewDesc(namespace+"_"+"system_spool_usage_bytes", "Spool total persisted usage.", nil, nil),
		"system_spool_usage_msgs":                          prometheus.NewDesc(namespace+"_"+"system_spool_usage_msgs", "Spool total number of persisted messages.", nil, nil),
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
		"client_rx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_rx_msgs_total", "Number of received messages.", variableLabelsVpnClient, nil),
		"client_tx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_tx_msgs_total", "Number of transmitted messages.", variableLabelsVpnClient, nil),
		"client_rx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_rx_bytes_total", "Number of received bytes.", variableLabelsVpnClient, nil),
		"client_tx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_tx_bytes_total", "Number of transmitted bytes.", variableLabelsVpnClient, nil),
		"client_rx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_rx_discarded_msgs_total", "Number of discarded received messages.", variableLabelsVpnClient, nil),
		"client_tx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_tx_discarded_msgs_total", "Number of discarded transmitted messages.", variableLabelsVpnClient, nil),
		"client_slow_subscriber":         prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClient, nil),
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
		"message_confirmed_delivered":           prometheus.NewDesc(namespace+"_"+"client_egress_message_confirmed_delivered", "Number of messacess succesfully delivered.", variableLabelsVpnClientFlow, nil),
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
		"bridge_client_num_subscriptions":               prometheus.NewDesc(namespace+"_"+"bridge_client_num_subscriptions", "Bridge Client Subscription", variableLabelsBridge, nil),
		"bridge_client_slow_subscriber":                 prometheus.NewDesc(namespace+"_"+"bridge_client_slow_subscriber", "Bridge Slow Subscriber", variableLabelsBridge, nil),
		"bridge_total_client_messages_received":         prometheus.NewDesc(namespace+"_"+"bridge_total_client_messages_received", "Bridge Total Client Messages Received", variableLabelsBridge, nil),
		"bridge_total_client_messages_sent":             prometheus.NewDesc(namespace+"_"+"bridge_total_client_messages_sent", "Bridge Total Client Messages sent", variableLabelsBridge, nil),
		"bridge_client_data_messages_received":          prometheus.NewDesc(namespace+"_"+"bridge_client_data_messages_received", "Bridge Client Data Msgs Received", variableLabelsBridge, nil),
		"bridge_client_data_messages_sent":              prometheus.NewDesc(namespace+"_"+"bridge_client_data_messages_sent", "Bridge Client Data Msgs Sent", variableLabelsBridge, nil),
		"bridge_client_persistent_messages_received":    prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_messages_received", "Bridge Client Persistent Msgs Received", variableLabelsBridge, nil),
		"bridge_client_persistent_messages_sent":        prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_messages_sent", "Bridge Client Persistent Msgs Sent", variableLabelsBridge, nil),
		"bridge_client_nonpersistent_messages_received": prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_messages_received", "Bridge Client Non-Persistent Msgs Received", variableLabelsBridge, nil),
		"bridge_client_nonpersistent_messages_sent":     prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_messages_sent", "Bridge Client Non-Persistent Msgs Sent", variableLabelsBridge, nil),
		"bridge_client_direct_messages_received":        prometheus.NewDesc(namespace+"_"+"bridge_client_direct_messages_received", "Bridge Client Direct Msgs Received", variableLabelsBridge, nil),
		"bridge_client_direct_messages_sent":            prometheus.NewDesc(namespace+"_"+"bridge_client_direct_messages_sent", "Bridge Client Direct Msgs Sent", variableLabelsBridge, nil),
		"bridge_total_client_bytes_received":            prometheus.NewDesc(namespace+"_"+"bridge_total_client_bytes_received", "Bridge Total Client Bytes Received", variableLabelsBridge, nil),
		"bridge_total_client_bytes_sent":                prometheus.NewDesc(namespace+"_"+"bridge_total_client_bytes_sent", "Bridge Total Client Bytes sent", variableLabelsBridge, nil),
		"bridge_client_data_bytes_received":             prometheus.NewDesc(namespace+"_"+"bridge_client_data_bytes_received", "Bridge Client Data Bytes Received", variableLabelsBridge, nil),
		"bridge_client_data_bytes_sent":                 prometheus.NewDesc(namespace+"_"+"bridge_client_data_bytes_sent", "Bridge Client Data Bytes Sent", variableLabelsBridge, nil),
		"bridge_client_persistent_bytes_received":       prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_bytes_received", "Bridge Client Persistent Bytes Received", variableLabelsBridge, nil),
		"bridge_client_persistent_bytes_sent":           prometheus.NewDesc(namespace+"_"+"bridge_client_persistent_bytes_sent", "Bridge Client Persistent Bytes Sent", variableLabelsBridge, nil),
		"bridge_client_nonpersistent_bytes_received":    prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_bytes_received", "Bridge Client Non-Persistent Bytes Received", variableLabelsBridge, nil),
		"bridge_client_nonpersistent_bytes_sent":        prometheus.NewDesc(namespace+"_"+"bridge_client_nonpersistent_bytes_sent", "Bridge Client Non-Persistent Bytes Sent", variableLabelsBridge, nil),
		"bridge_client_direct_bytes_received":           prometheus.NewDesc(namespace+"_"+"bridge_client_direct_bytes_received", "Bridge Client Direct Bytes Received", variableLabelsBridge, nil),
		"bridge_client_direct_bytes_sent":               prometheus.NewDesc(namespace+"_"+"bridge_client_direct_bytes_sent", "Bridge Client Direct Bytes Sent", variableLabelsBridge, nil),
		"bridge_client_large_messages_received":         prometheus.NewDesc(namespace+"_"+"bridge_client_large_messages_received", "Bridge Client Large Messages received", variableLabelsBridge, nil),
		"bridge_denied_duplicate_clients":               prometheus.NewDesc(namespace+"_"+"bridge_denied_duplicate_clients", "Bridge Deneid Duplicate Clients", variableLabelsBridge, nil),
		"bridge_not_enough_space_msgs_sent":             prometheus.NewDesc(namespace+"_"+"bridge_not_enough_space_msgs_sent", "Bridge Not Enough Space Messages Sent", variableLabelsBridge, nil),
		"bridge_max_exceeded_msgs_sent":                 prometheus.NewDesc(namespace+"_"+"bridge_max_exceeded_msgs_sent", "Bridge Max Exceeded Messages Sent", variableLabelsBridge, nil),
		"bridge_subscribe_client_not_found":             prometheus.NewDesc(namespace+"_"+"bridge_subscribe_client_not_found", "Bridge Subscriber Client Not Found", variableLabelsBridge, nil),
		"bridge_not_found_msgs_sent":                    prometheus.NewDesc(namespace+"_"+"bridge_not_found_msgs_sent", "Bridge Not Found Messages Sent", variableLabelsBridge, nil),
		"bridge_current_ingress_rate_per_second":        prometheus.NewDesc(namespace+"_"+"bridge_current_ingress_rate_per_second", "Current Ingress Rate / s", variableLabelsBridge, nil),
		"bridge_current_egress_rate_per_second":         prometheus.NewDesc(namespace+"_"+"bridge_current_egress_rate_per_second", "Current Egress Rate / s", variableLabelsBridge, nil),
		"bridge_total_ingress_discards":                 prometheus.NewDesc(namespace+"_"+"bridge_total_ingress_discards", "Total Ingress Discards", variableLabelsBridge, nil),
		"bridge_total_egress_discards":                  prometheus.NewDesc(namespace+"_"+"bridge_total_egress_discards", "Total Egress Discards", variableLabelsBridge, nil),
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
		"total_bytes_spooled":             prometheus.NewDesc(namespace+"_"+"queue_byte_spooled", "Queue spool total of all spooled messages in bytes.", variableLabelsVpnQueue, nil),
		"total_messages_spooled":          prometheus.NewDesc(namespace+"_"+"queue_msg_spooled", "Queue spool total of all spooled messages.", variableLabelsVpnQueue, nil),
		"messages_redelivered":            prometheus.NewDesc(namespace+"_"+"queue_msg_redelivered", "Queue total msg redeliveries.", variableLabelsVpnQueue, nil),
		"messages_transport_retransmited": prometheus.NewDesc(namespace+"_"+"queue_msg_retransmited", "Queue total msg retransmitted on transport.", variableLabelsVpnQueue, nil),
		"spool_usage_exceeded":            prometheus.NewDesc(namespace+"_"+"queue_msg_spool_usage_exceeded", "Queue total number of messages exceeded the spool usage.", variableLabelsVpnQueue, nil),
		"max_message_size_exceeded":       prometheus.NewDesc(namespace+"_"+"queue_msg_max_msg_size_exceeded", "Queue total number of messages exceeded the max message size.", variableLabelsVpnQueue, nil),
		"total_deleted_messages":          prometheus.NewDesc(namespace+"_"+"queue_msg_total_deleted", "Queuse total number that was deleted.", variableLabelsVpnQueue, nil),
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
}

// Get version of broker
func (e *Exporter) getVersionSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Version struct {
					Description string `xml:"description"`
					CurrentLoad string `xml:"current-load"`
					Uptime      struct {
						Days      float64 `xml:"days"`
						Hours     float64 `xml:"hours"`
						Mins      float64 `xml:"mins"`
						Secs      float64 `xml:"secs"`
						TotalSecs float64 `xml:"total-secs"`
					} `xml:"uptime"`
				} `xml:"version"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><version/></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape getVersionSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml getVersionSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "Unexpected result for getVersionSemp1", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	// remember this for the label
	vmrVersion := strings.TrimPrefix(target.RPC.Show.Version.CurrentLoad, "soltr_")
	// compute a version number so it can be measured by Prometheus
	var vmrVersionStrBuffer bytes.Buffer
	for _, s := range strings.Split(vmrVersion, ".") {
		vmrVersionStrBuffer.WriteString(fmt.Sprintf("%03v", s))
	}
	var vmrVersionNr float64
	vmrVersionNr, _ = strconv.ParseFloat(vmrVersionStrBuffer.String(), 64)

	ch <- prometheus.MustNewConstMetric(metricDesc["Version"]["system_version_currentload"], prometheus.GaugeValue, vmrVersionNr)
	ch <- prometheus.MustNewConstMetric(metricDesc["Version"]["system_version_uptime_totalsecs"], prometheus.GaugeValue, target.RPC.Show.Version.Uptime.TotalSecs)
	ch <- prometheus.MustNewConstMetric(metricDesc["Version"]["exporter_version_current"], prometheus.GaugeValue, solaceExporterVersion)

	return 1, nil
}

// Get system health information
func (e *Exporter) getHealthSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				System struct {
					Health struct {
						DiskLatencyMinimumValue     float64 `xml:"disk-latency-minimum-value"`
						DiskLatencyMaximumValue     float64 `xml:"disk-latency-maximum-value"`
						DiskLatencyAverageValue     float64 `xml:"disk-latency-average-value"`
						DiskLatencyCurrentValue     float64 `xml:"disk-latency-current-value"`
						ComputeLatencyMinimumValue  float64 `xml:"compute-latency-minimum-value"`
						ComputeLatencyMaximumValue  float64 `xml:"compute-latency-maximum-value"`
						ComputeLatencyAverageValue  float64 `xml:"compute-latency-average-value"`
						ComputeLatencyCurrentValue  float64 `xml:"compute-latency-current-value"`
						MateLinkLatencyMinimumValue float64 `xml:"mate-link-latency-minimum-value"`
						MateLinkLatencyMaximumValue float64 `xml:"mate-link-latency-maximum-value"`
						MateLinkLatencyAverageValue float64 `xml:"mate-link-latency-average-value"`
						MateLinkLatencyCurrentValue float64 `xml:"mate-link-latency-current-value"`
					} `xml:"health"`
				} `xml:"system"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><system><health/></system></show ></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape HealthSemp1. Attention this is only supported by software broker not by appliances", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml HealthSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_disk_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_disk_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_disk_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_disk_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyCurrentValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_compute_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_compute_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_compute_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_compute_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyCurrentValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_mate_link_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_mate_link_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_mate_link_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricDesc["Health"]["system_mate_link_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyCurrentValue/1e6)

	return 1, nil
}

// Get system storage-element information (for Software Broker)
func (e *Exporter) getStorageElementSemp1(ch chan<- prometheus.Metric, storageElementFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				StorageElements struct {
					StorageElement []struct {
						Name        string  `xml:"name"`
						Path        string  `xml:"path"`
						DeviceName  string  `xml:"device-name"`
						TotalBlocks float64 `xml:"total-blocks"`
						UsedBlocks  float64 `xml:"used-blocks"`
						AvailBlocks float64 `xml:"available-blocks"`
						UsedPercent float64 `xml:"used-percentage"`
					} `xml:"storage-element"`
				} `xml:"storage-element"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><storage-element><pattern>" + storageElementFilter + "</pattern></storage-element></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape StorageElementSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml StorageElementSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	blockSize := 1024.0

	for _, element := range target.RPC.Show.StorageElements.StorageElement {
		ch <- prometheus.MustNewConstMetric(metricDesc["StorageElement"]["system_storage_used_percent"], prometheus.GaugeValue, element.UsedPercent, element.Path, element.DeviceName, element.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["StorageElement"]["system_storage_used_bytes"], prometheus.GaugeValue, element.UsedBlocks*blockSize, element.Path, element.DeviceName, element.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["StorageElement"]["system_storage_avail_bytes"], prometheus.GaugeValue, element.AvailBlocks*blockSize, element.Path, element.DeviceName, element.Name)
	}

	return 1, nil
}

// Get system disk information (for Appliance)
func (e *Exporter) getDiskSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Disk struct {
					DiskInfos struct {
						DiskInfo []struct {
							Path        string  `xml:"mounted-on"`
							DeviceName  string  `xml:"file-system"`
							TotalBlocks float64 `xml:"blocks"`
							UsedBlocks  float64 `xml:"used"`
							AvailBlocks float64 `xml:"available"`
							UsedPercent string  `xml:"use"`
						} `xml:"disk-info"`
					} `xml:"disk-infos"`
				} `xml:"disk"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><disk><detail/></disk></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape DiskSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml DiskSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	blockSize := 1024.0
	for _, disk := range target.RPC.Show.Disk.DiskInfos.DiskInfo {
		var usedPercent float64
		usedPercent, _ = strconv.ParseFloat(strings.Trim(disk.UsedPercent, "%"), 64)
		ch <- prometheus.MustNewConstMetric(metricDesc["Disk"]["system_disk_used_percent"], prometheus.GaugeValue, usedPercent, disk.Path, disk.DeviceName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Disk"]["system_disk_used_bytes"], prometheus.GaugeValue, disk.UsedBlocks*blockSize, disk.Path, disk.DeviceName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Disk"]["system_disk_avail_bytes"], prometheus.GaugeValue, disk.AvailBlocks*blockSize, disk.Path, disk.DeviceName)
	}

	return 1, nil
}

// Get system memory information
func (e *Exporter) getMemorySemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Memory struct {
					PhysicalUsagePercent     float64 `xml:"physical-memory-usage-percent"`
					SubscriptionUsagePercent float64 `xm:"subscription-memory-usage-percent"`
					SlotInfos                struct {
						SlotInfo []struct {
							Slot             string  `xml:"slot"`
							NabBufLoadFactor float64 `xml:"nab-buffer-load-factor"`
						} `xml:"slot-info"`
					} `xml:"slot-infos"`
				} `xml:"memory"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><memory/></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape MemorySemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml MemorySemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- prometheus.MustNewConstMetric(metricDesc["Memory"]["system_memory_physical_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.PhysicalUsagePercent)
	ch <- prometheus.MustNewConstMetric(metricDesc["Memory"]["system_memory_subscription_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.SubscriptionUsagePercent)
	ch <- prometheus.MustNewConstMetric(metricDesc["Memory"]["system_nab_buffer_load_factor"], prometheus.GaugeValue, target.RPC.Show.Memory.SlotInfos.SlotInfo[0].NabBufLoadFactor)

	return 1, nil
}

// Get interface information
func (e *Exporter) getInterfaceSemp1(ch chan<- prometheus.Metric, interfaceFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Interface struct {
					Interfaces struct {
						Interface []struct {
							Name    string `xml:"phy-interface"`
							Enabled string `xml:"enabled"`
							State   string `xml:"oper-status"`
							Stats   struct {
								RxBytes float64 `xml:"rx-bytes"`
								TxBytes float64 `xml:"tx-bytes"`
							} `xml:"stats"`
						} `xml:"interface"`
					} `xml:"interfaces"`
				} `xml:"interface"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><interface><phy-interface>" + interfaceFilter + "</phy-interface></interface></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape InterfaceSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml InterfaceSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, intf := range target.RPC.Show.Interface.Interfaces.Interface {
		ch <- prometheus.MustNewConstMetric(metricDesc["Interface"]["network_if_rx_bytes"], prometheus.CounterValue, intf.Stats.RxBytes, intf.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Interface"]["network_if_tx_bytes"], prometheus.CounterValue, intf.Stats.TxBytes, intf.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Interface"]["network_if_state"], prometheus.GaugeValue, encodeMetricMulti(intf.State, []string{"Down", "Up"}), intf.Name)
	}

	return 1, nil
}

// Get global stats information
func (e *Exporter) getGlobalStatsSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Stats struct {
					Client struct {
						Global struct {
							Stats struct {
								ClientsConnected float64 `xml:"total-clients-connected"`
								DataRxMsgCount   float64 `xml:"client-data-messages-received"`
								DataTxMsgCount   float64 `xml:"client-data-messages-sent"`
								DataRxByteCount  float64 `xml:"client-data-bytes-received"`
								DataTxByteCount  float64 `xml:"client-data-bytes-sent"`
								RxMsgsRate       float64 `xml:"current-ingress-rate-per-second"`
								TxMsgsRate       float64 `xml:"current-egress-rate-per-second"`
								RxBytesRate      float64 `xml:"current-ingress-byte-rate-per-second"`
								TxBytesRate      float64 `xml:"current-egress-byte-rate-per-second"`
								IngressDiscards  struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"global"`
					} `xml:"client"`
				} `xml:"stats"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><stats><client/></stats></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape GlobalStatsSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml GlobalStatsSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_total_clients_connected"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.ClientsConnected)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_rx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxMsgCount)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_tx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxMsgCount)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_rx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxByteCount)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_tx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxByteCount)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_total_rx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.IngressDiscards.DiscardedRxMsgCount)
	ch <- prometheus.MustNewConstMetric(metricDesc["GlobalStats"]["system_total_tx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.EgressDiscards.DiscardedTxMsgCount)

	return 1, nil
}

// Get system-wide spool information
func (e *Exporter) getSpoolSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						QuotaDiskUsage           float64 `xml:"max-disk-usage"`
						QuotaMsgCount            string  `xml:"max-message-count"`
						PersistUsage             float64 `xml:"current-persist-usage"`
						PersistMsgCount          float64 `xml:"total-messages-currently-spooled"`
						ActiveDiskPartitionUsage string  `xml:"active-disk-partition-usage"` // May be "-"
					} `xml:"message-spool-info"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool></message-spool></show ></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape Solace", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- prometheus.MustNewConstMetric(metricDesc["Spool"]["system_spool_quota_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.QuotaDiskUsage*1048576.0))
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	f1, err3 := strconv.ParseFloat(s1, 64)
	if err3 == nil {
		ch <- prometheus.MustNewConstMetric(metricDesc["Spool"]["system_spool_quota_msgs"], prometheus.GaugeValue, f1*1000000)
	}
	ActiveDiskPartitionUsage, err4 := strconv.ParseFloat(target.RPC.Show.Spool.Info.ActiveDiskPartitionUsage, 64)
	if err4 == nil {
		ch <- prometheus.MustNewConstMetric(metricDesc["Spool"]["system_spool_disk_partition_usage_active_percent"], prometheus.GaugeValue, math.Round(ActiveDiskPartitionUsage))
	}

	ch <- prometheus.MustNewConstMetric(metricDesc["Spool"]["system_spool_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.PersistUsage*1048576.0))
	ch <- prometheus.MustNewConstMetric(metricDesc["Spool"]["system_spool_usage_msgs"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistMsgCount)

	return 1, nil
}

// Get system-wide basic redundancy information for HA triples
func (e *Exporter) getRedundancySemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	var f float64

	type Data struct {
		RPC struct {
			Show struct {
				Red struct {
					ConfigStatus      string `xml:"config-status"`
					RedundancyStatus  string `xml:"redundancy-status"`
					OperatingMode     string `xml:"operating-mode"`
					ActiveStandbyRole string `xml:"active-standby-role"`
					MateRouterName    string `xml:"mate-router-name"`
					VirtualRouters    struct {
						Primary struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"primary"`
						Backup struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"backup"`
					} `xml:"virtual-routers"`
				} `xml:"redundancy"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><redundancy/></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape RedundancySemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml RedundancySemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	mateRouterName := "" + target.RPC.Show.Red.MateRouterName
	ch <- prometheus.MustNewConstMetric(metricDesc["Redundancy"]["system_redundancy_config"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ConfigStatus, []string{"Disabled", "Enabled", "Shutdown"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(metricDesc["Redundancy"]["system_redundancy_up"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyStatus, []string{"Down", "Up"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(metricDesc["Redundancy"]["system_redundancy_role"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ActiveStandbyRole, []string{"Backup", "Primary", "Monitor", "Undefined"}), mateRouterName)

	if target.RPC.Show.Red.ActiveStandbyRole == "Primary" && target.RPC.Show.Red.VirtualRouters.Primary.Status.Activity == "Local Active" ||
		target.RPC.Show.Red.ActiveStandbyRole == "Backup" && target.RPC.Show.Red.VirtualRouters.Backup.Status.Activity == "Local Active" {
		f = 1
	} else {
		f = 0
	}
	ch <- prometheus.MustNewConstMetric(metricDesc["Redundancy"]["system_redundancy_local_active"], prometheus.GaugeValue, f, mateRouterName)

	return 1, nil
}

// Get DR replication statistics
func (e *Exporter) getReplicationStatsSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Repl struct {
					Mate struct {
						Name string `xml:"router-name"`
					} `xml:"mate"`
					ConfigSync struct {
						Bridge struct {
							AdminState string `xml:"admin-state"`
							State      string `xml:"state"`
						} `xml:"bridge"`
					} `xml:"config-sync"`
					Stats struct {
						ActiveStats struct {
							MsgProcessing struct {
								SyncQ2Standby      float64 `xml:"sync-msgs-queued-to-standby"`
								SyncQ2StandbyAsync float64 `xml:"sync-msgs-queued-to-standby-as-async"`
								AsyncQ2Standby     float64 `xml:"async-msgs-queued-to-standby"`
								PromotedQ2Standby  float64 `xml:"promoted-msgs-queued-to-standby"`
								PrunedConsumed     float64 `xml:"pruned-locally-consumed-msgs"`
							} `xml:"message-processing"`
							SyncRepl struct {
								Trans2Ineligible float64 `xml:"transitions-to-ineligible"`
							} `xml:"sync-replication"`
							AckPropagation struct {
								TxMsgToStandby   float64 `xml:"msgs-tx-to-standby"`
								RxReqFromStandby float64 `xml:"rec-req-from-standby"`
							} `xml:"ack-propagation"`
						} `xml:"active-stats"`
						StandbyStats struct {
							MsgProcessing struct {
								RxMsgFromActive float64 `xml:"msgs-rx-from-active"`
							} `xml:"message-processing"`
							AckPropagation struct {
								RxAckPropMsgs float64 `xml:"ack-prop-msgs-rx"`
								TxReconReq    float64 `xml:"recon-req-tx"`
								RxOutOfSeq    float64 `xml:"out-of-seq-rx"`
							} `xml:"ack-propagation"`
							XaRepl struct {
								XaReq                float64 `xml:"transaction-request"`
								XaReqSuccess         float64 `xml:"transaction-request-success"`
								XaReqSuccessPrepare  float64 `xml:"transaction-request-success-prepare"`
								XaReqSuccessCommit   float64 `xml:"transaction-request-success-commit"`
								XaReqSuccessRollback float64 `xml:"transaction-request-success-rollback"`
								XaReqFail            float64 `xml:"transaction-request-fail"`
								XaReqFailPrepare     float64 `xml:"transaction-request-fail-prepare"`
								XaReqFailCommit      float64 `xml:"transaction-request-fail-commit"`
								XaReqFailRollback    float64 `xml:"transaction-request-fail-rollback"`
							} `xml:"transaction-replication"`
						} `xml:"standby-stats"`
					} `xml:"stats"`
				} `xml:"replication"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><replication><stats/></replication></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape ReplicationStatsSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml ReplicationStatsSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	replMateName := "" + target.RPC.Show.Repl.Mate.Name
	if replMateName != "" {
		replBridge := target.RPC.Show.Repl.ConfigSync.Bridge
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_bridge_admin_state"], prometheus.GaugeValue, encodeMetricMulti(replBridge.AdminState, []string{"Disabled", "Enabled", "-"}), replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_bridge_state"], prometheus.GaugeValue, encodeMetricMulti(replBridge.State, []string{"down", "up", "n/a"}), replMateName)
		//Active stats
		activeStats := target.RPC.Show.Repl.Stats.ActiveStats
		//Message processing
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_sync_msgs_queued_to_standby"], prometheus.GaugeValue, activeStats.MsgProcessing.SyncQ2Standby, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_sync_msgs_queued_to_standby_as_async"], prometheus.GaugeValue, activeStats.MsgProcessing.SyncQ2StandbyAsync, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_async_msgs_queued_to_standby"], prometheus.GaugeValue, activeStats.MsgProcessing.AsyncQ2Standby, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_promoted_msgs_queued_to_standby"], prometheus.GaugeValue, activeStats.MsgProcessing.PromotedQ2Standby, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_pruned_locally_consumed_msgs"], prometheus.GaugeValue, activeStats.MsgProcessing.PrunedConsumed, replMateName)
		//Sync replication
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_transitions_to_ineligible"], prometheus.GaugeValue, activeStats.SyncRepl.Trans2Ineligible, replMateName)
		//Ack propagation
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_msgs_tx_to_standby"], prometheus.GaugeValue, activeStats.AckPropagation.TxMsgToStandby, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_rec_req_from_standby"], prometheus.GaugeValue, activeStats.AckPropagation.RxReqFromStandby, replMateName)
		//Standby stats
		standbyStats := target.RPC.Show.Repl.Stats.StandbyStats
		//Message processing
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_msgs_rx_from_active"], prometheus.GaugeValue, standbyStats.MsgProcessing.RxMsgFromActive, replMateName)
		//Ack propagation
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_ack_prop_msgs_rx"], prometheus.GaugeValue, standbyStats.AckPropagation.RxAckPropMsgs, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_recon_req_tx"], prometheus.GaugeValue, standbyStats.AckPropagation.TxReconReq, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_out_of_seq_rx"], prometheus.GaugeValue, standbyStats.AckPropagation.RxOutOfSeq, replMateName)
		//Transaction replication
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req"], prometheus.GaugeValue, standbyStats.XaRepl.XaReq, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_success"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqSuccess, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_success_prepare"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqSuccessPrepare, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_success_commit"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqSuccessCommit, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_success_rollback"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqSuccessRollback, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_fail"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqFail, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_fail_prepare"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqFailPrepare, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_fail_commit"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqFailCommit, replMateName)
		ch <- prometheus.MustNewConstMetric(metricDesc["ReplicationStats"]["system_replication_xa_req_fail_rollback"], prometheus.GaugeValue, standbyStats.XaRepl.XaReqFailRollback, replMateName)
	}

	return 1, nil
}

// Config Sync Status for Broker and Vpn
func (e *Exporter) getConfigSyncRouterSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Database struct {
						Local struct {
							Tables struct {
								Table []struct {
									Type               string  `xml:"type"`
									TimeInStateSeconds float64 `xml:"time-in-state-seconds"`
									Name               string  `xml:"name"`
									Ownership          string  `xml:"ownership"`
									SyncState          string  `xml:"sync-state"`
									TimeInState        string  `xml:"time-in-state"`
								} `xml:"table"`
							} `xml:"tables"`
						} `xml:"local"`
					} `xml:"database"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><config-sync><database/><router/></config-sync></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncRouter"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncRouter"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncRouter"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncRouter"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
	}

	return 1, nil
}

// Get info of all vpn's
func (e *Exporter) getVpnSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					ManagementMessageVpn string `xml:"management-message-vpn"`
					Vpn                  []struct {
						Name                           string  `xml:"name"`
						IsManagementMessageVpn         bool    `xml:"is-management-message-vpn"`
						Enabled                        bool    `xml:"enabled"`
						Operational                    bool    `xml:"operational"`
						LocallyConfigured              bool    `xml:"locally-configured"`
						LocalStatus                    string  `xml:"local-status"`
						UniqueSubscriptions            float64 `xml:"unique-subscriptions"`
						TotalLocalUniqueSubscriptions  float64 `xml:"total-local-unique-subscriptions"`
						TotalRemoteUniqueSubscriptions float64 `xml:"total-remote-unique-subscriptions"`
						TotalUniqueSubscriptions       float64 `xml:"total-unique-subscriptions"`
						Connections                    float64 `xml:"connections"`
						QuotaConnections               float64 `xml:"max-connections"`
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "Unexpected result for VpnSemp1", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_is_management_vpn"], prometheus.GaugeValue, encodeMetricBool(vpn.IsManagementMessageVpn), vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_enabled"], prometheus.GaugeValue, encodeMetricBool(vpn.Enabled), vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_operational"], prometheus.GaugeValue, encodeMetricBool(vpn.Operational), vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_locally_configured"], prometheus.GaugeValue, encodeMetricBool(vpn.LocallyConfigured), vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_unique_subscriptions"], prometheus.GaugeValue, vpn.UniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_total_local_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalLocalUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_total_remote_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalRemoteUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_total_unique_subscriptions"], prometheus.GaugeValue, vpn.TotalUniqueSubscriptions, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_connections"], prometheus.GaugeValue, vpn.Connections, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["Vpn"]["vpn_quota_connections"], prometheus.GaugeValue, vpn.QuotaConnections, vpn.Name)
	}

	return 1, nil
}

// Replication Config and status
func (e *Exporter) getVpnReplicationSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Replication struct {
						MessageVpns struct {
							MessageVpn []struct {
								VpnName                    string `xml:"vpn-name"`
								AdminState                 string `xml:"admin-state"`
								ConfigState                string `xml:"config-state"`
								TransactionReplicationMode string `xml:"transaction-replication-mode"`
							} `xml:"message-vpn"`
						} `xml:"message-vpns"`
					} `xml:"replication"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><replication/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Replication.MessageVpns.MessageVpn {
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnReplication"]["vpn_replication_admin_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.AdminState, []string{"shutdown", "enabled", "n/a"}), vpn.VpnName)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnReplication"]["vpn_replication_config_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.ConfigState, []string{"standby", "active", "n/a"}), vpn.VpnName)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnReplication"]["vpn_replication_transaction_replication_mode"], prometheus.GaugeValue, encodeMetricMulti(vpn.TransactionReplicationMode, []string{"async", "sync", "n/a"}), vpn.VpnName)
	}

	return 1, nil
}

// Config Sync Status for Broker and Vpn
func (e *Exporter) getConfigSyncVpnSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Database struct {
						Local struct {
							Tables struct {
								Table []struct {
									Type               string  `xml:"type"`
									TimeInStateSeconds float64 `xml:"time-in-state-seconds"`
									Name               string  `xml:"name"`
									Ownership          string  `xml:"ownership"`
									SyncState          string  `xml:"sync-state"`
									TimeInState        string  `xml:"time-in-state"`
								} `xml:"table"`
							} `xml:"tables"`
						} `xml:"local"`
					} `xml:"database"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><config-sync><database/><message-vpn/><vpn-name>" + vpnFilter + "</vpn-name></config-sync></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncVpn"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncVpn"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncVpn"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["ConfigSyncVpn"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
	}

	return 1, nil
}

// Get status of bridges for all vpns
func (e *Exporter) getBridgeSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						NumTotalBridgesValue                 float64 `xml:"num-total-bridges"`
						MaxNumTotalBridgesValue              float64 `xml:"max-num-total-bridges"`
						NumLocalBridgesValue                 float64 `xml:"num-local-bridges"`
						MaxNumLocalBridgesValue              float64 `xml:"max-num-local-bridges"`
						NumRemoteBridgesValue                float64 `xml:"num-remote-bridges"`
						MaxNumRemoteBridgesValue             float64 `xml:"max-num-remote-bridges"`
						NumTotalRemoteBridgeSubscriptions    float64 `xml:"num-total-remote-bridge-subscriptions"`
						MaxNumTotalRemoteBridgeSubscriptions float64 `xml:"max-num-total-remote-bridge-subscriptions"`
						Bridge                               []struct {
							BridgeName                      string  `xml:"bridge-name"`
							LocalVpnName                    string  `xml:"local-vpn-name"`
							ConnectedRemoteVpnName          string  `xml:"connected-remote-vpn-name"`
							ConnectedRemoteRouterName       string  `xml:"connected-remote-router-name"`
							AdminState                      string  `xml:"admin-state"`
							ConnectionEstablisher           string  `xml:"connection-establisher"`
							InboundOperationalState         string  `xml:"inbound-operational-state"`
							InboundOperationalFailureReason string  `xml:"inbound-operational-failure-reason"`
							OutboundOperationalState        string  `xml:"outbound-operational-state"`
							QueueOperationalState           string  `xml:"queue-operational-state"`
							Redundancy                      string  `xml:"redundancy"`
							ConnectionUptimeInSeconds       float64 `xml:"connection-uptime-in-seconds"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><bridge><bridge-name-pattern>" + itemFilter + "</bridge-name-pattern><vpn-name-pattern>" + vpnFilter + "</vpn-name-pattern></bridge></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape BridgeSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml BridgeSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_num_total_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_max_num_total_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_num_local_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumLocalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_max_num_local_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumLocalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_num_remote_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumRemoteBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_max_num_remote_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumRemoteBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_num_total_remote_bridge_subscriptions"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalRemoteBridgeSubscriptions)
	ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridges_max_num_total_remote_bridge_subscriptions"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalRemoteBridgeSubscriptions)
	opStates := []string{"Init", "Shutdown", "NoShutdown", "Prepare", "Prepare-WaitToConnect",
		"Prepare-FetchingDNS", "NotReady", "NotReady-Connecting", "NotReady-Handshaking", "NotReady-WaitNext",
		"NotReady-WaitReuse", "NotRead-WaitBridgeVersionMismatch", "NotReady-WaitCleanup", "Ready", "Ready-Subscribing",
		"Ready-InSync", "NotApplicable", "Invalid"}
	failReasons := []string{"Bridge disabled", "No remote message-vpns configured", "SMF service is disabled", "Msg Backbone is disabled",
		"Local message-vpn is disabled", "Active-Standby Role Mismatch", "Invalid Active-Standby Role", "Redundancy Disabled", "Not active",
		"Replication standby", "Remote message-vpns disabled", "Enforce-trusted-common-name but empty trust-common-name list", "SSL transport used but cipher-suite list is empty", "Authentication Scheme is Client-Certificate but no certificate is configured",
		"Client-Certificate Authentication Scheme used but not all Remote Message VPNs use SSL", "Basic Authentication Scheme used but Basic Client Username not configured", "Cluster Down", "Cluster Link Down", ""}
	for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
		bridgeName := bridge.BridgeName
		vpnName := bridge.LocalVpnName
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_admin_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.AdminState, []string{"Enabled", "Disabled", "-"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_connection_establisher"], prometheus.GaugeValue, encodeMetricMulti(bridge.ConnectionEstablisher, []string{"NotApplicable", "Local", "Remote", "Invalid"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_inbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalState, opStates), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_inbound_operational_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalFailureReason, failReasons), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_outbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.OutboundOperationalState, opStates), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_queue_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.QueueOperationalState, []string{"NotApplicable", "Bound", "Unbound"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_redundancy"], prometheus.GaugeValue, encodeMetricMulti(bridge.Redundancy, []string{"NotApplicable", "auto", "primary", "backup", "static", "none"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["Bridge"]["bridge_connection_uptime_in_seconds"], prometheus.GaugeValue, bridge.ConnectionUptimeInSeconds, vpnName, bridgeName)
	}
	return 1, nil
}

// Get summary for each client of vpns
// This can result in heavy system load when lots of clients are connected
func (e *Exporter) getClientSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientAddress    string  `xml:"client-address"`
							ClientName       string  `xml:"name"`
							MsgVpnName       string  `xml:"message-vpn"`
							NumSubscriptions float64 `xml:"num-subscriptions"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><connected/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIp := strings.Split(client.ClientAddress, ":")[0]
			ch <- prometheus.MustNewConstMetric(metricDesc["Client"]["client_num_subscriptions"], prometheus.GaugeValue, client.NumSubscriptions, client.MsgVpnName, client.ClientName, clientIp)
		}
		body.Close()
	}

	return 1, nil
}

// Get slow subscriber client of vpns
// This can result in heavy system load when lots of clients are connected
func (e *Exporter) getClientSlowSubscriberSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientAddress string `xml:"client-address"`
							ClientName    string `xml:"name"`
							MsgVpnName    string `xml:"message-vpn"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><slow-subscriber/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientSlowSubscriberSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientSlowSubscriberSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC
		var slowSubscriber float64
		slowSubscriber = 1

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			clientIp := strings.Split(client.ClientAddress, ":")[0]
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientSlowSubscriber"]["client_slow_subscriber"], prometheus.GaugeValue, slowSubscriber, client.MsgVpnName, client.ClientName, clientIp)
		}
		body.Close()
	}

	return 1, nil
}

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Exporter) getClientStatsSemp1(ch chan<- prometheus.Metric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName     string `xml:"name"`
							ClientUsername string `xml:"client-username"`
							MsgVpnName     string `xml:"message-vpn"`
							SlowSubscriber bool   `xml:"slow-subscriber"`
							Stats          struct {
								DataRxByteCount   float64 `xml:"client-data-bytes-received"`
								DataRxMsgCount    float64 `xml:"client-data-messages-received"`
								DataTxByteCount   float64 `xml:"client-data-bytes-sent"`
								DataTxMsgCount    float64 `xml:"client-data-messages-sent"`
								AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
								AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
								AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
								AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
								RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
								RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
								TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
								TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								IngressDiscards   struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><stats/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientStatSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientStatSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_rx_msgs_total"], prometheus.CounterValue, client.Stats.DataRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_tx_msgs_total"], prometheus.CounterValue, client.Stats.DataTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_rx_bytes_total"], prometheus.CounterValue, client.Stats.DataRxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_tx_bytes_total"], prometheus.CounterValue, client.Stats.DataTxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_rx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.IngressDiscards.DiscardedRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_tx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.EgressDiscards.DiscardedTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientStats"]["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername)
		}
		body.Close()
	}

	return 1, nil
}

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Exporter) getClientMessageSpoolStatsSemp1(ch chan<- prometheus.Metric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName        string  `xml:"name"`
							ClientUsername    string  `xml:"client-username"`
							MsgVpnName        string  `xml:"message-vpn"`
							ClientProfile     string  `xml:"profile"`
							AclProfile        string  `xml:"acl-profile"`
							SlowSubscriber    bool    `xml:"slow-subscriber"`
							ElidingTopics     float64 `xml:"eliding-topics"`
							FlowsIngress      float64 `xml:"total-ingress-flows"`
							FlowsEgress       float64 `xml:"total-egress-flows"`
							MessageSpoolStats struct {
								IngressFlowStats []struct {
									SpoolingNotReady               float64 `xml:"spooling-not-ready"`
									OutOfOrderMessagesReceived     float64 `xml:"out-of-order-messages-received"`
									DuplicateMessagesReceived      float64 `xml:"duplicate-messages-received"`
									NoEligibleDestinations         float64 `xml:"no-eligible-destinations"`
									GuaranteedMessages             float64 `xml:"guaranteed-messages"`
									NoLocalDelivery                float64 `xml:"no-local-delivery"`
									SeqNumRollover                 float64 `xml:"seq-num-rollover"`
									SeqNumMessagesDiscarded        float64 `xml:"seq-num-messages-discarded"`
									TransactedMessagesNotSequenced float64 `xml:"transacted-messages-not-sequenced"`
									DestinationGroupError          float64 `xml:"destination-group-error"`
									SmfTtlExceeded                 float64 `xml:"smf-ttl-exceeded"`
									PublishAclDenied               float64 `xml:"publish-acl-denied"`
								} `xml:"ingress-flow-stats"`
								EgressFlowStats []struct {
									WindowSize                        float64 `xml:"window-size"`
									UsedWindow                        float64 `xml:"used-window"`
									WindowClosed                      float64 `xml:"window-closed"`
									MessageRedelivered                float64 `xml:"message-redelivered"`
									MessageTransportRetransmit        float64 `xml:"message-transport-retransmit"`
									MessageConfirmedDelivered         float64 `xml:"message-confirmed-delivered"`
									ConfirmedDeliveredStoreAndForward float64 `xml:"confirmed-delivered-store-and-forward"`
									ConfirmedDeliveredCutThrough      float64 `xml:"confirmed-delivered-cut-through"`
									UnackedMessages                   float64 `xml:"unacked-messages"`
								} `xml:"egress-flow-stats"`
							} `xml:"message-spool-stats"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><message-spool-stats/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientMessageSpoolStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientMessageSpoolStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["client_flows_ingress"], prometheus.GaugeValue, client.FlowsIngress, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["client_flows_egress"], prometheus.GaugeValue, client.FlowsEgress, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)

			for flowId, ingressFlow := range client.MessageSpoolStats.IngressFlowStats {
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["spooling_not_ready"], prometheus.CounterValue, ingressFlow.SpoolingNotReady, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["out_of_order_messages_received"], prometheus.CounterValue, ingressFlow.OutOfOrderMessagesReceived, client.MsgVpnName, client.ClientName, client.ClientProfile, client.AclProfile, client.ClientUsername, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["duplicate_messages_received"], prometheus.CounterValue, ingressFlow.DuplicateMessagesReceived, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["no_eligible_destinations"], prometheus.CounterValue, ingressFlow.NoEligibleDestinations, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["guaranteed_messages"], prometheus.CounterValue, ingressFlow.GuaranteedMessages, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["no_local_delivery"], prometheus.CounterValue, ingressFlow.NoLocalDelivery, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["seq_num_rollover"], prometheus.CounterValue, ingressFlow.SeqNumRollover, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["seq_num_messages_discarded"], prometheus.CounterValue, ingressFlow.SeqNumMessagesDiscarded, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["transacted_messages_not_sequenced"], prometheus.CounterValue, ingressFlow.TransactedMessagesNotSequenced, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["destination_group_error"], prometheus.CounterValue, ingressFlow.DestinationGroupError, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["smf_ttl_exceeded"], prometheus.CounterValue, ingressFlow.SmfTtlExceeded, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["publish_acl_denied"], prometheus.CounterValue, ingressFlow.PublishAclDenied, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
			}
			for flowId, egressFlow := range client.MessageSpoolStats.EgressFlowStats {
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["window_size"], prometheus.CounterValue, egressFlow.WindowSize, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["used_window"], prometheus.CounterValue, egressFlow.UsedWindow, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["window_closed"], prometheus.CounterValue, egressFlow.WindowClosed, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["message_redelivered"], prometheus.CounterValue, egressFlow.MessageRedelivered, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["message_transport_retransmit"], prometheus.CounterValue, egressFlow.MessageTransportRetransmit, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["message_confirmed_delivered"], prometheus.CounterValue, egressFlow.MessageConfirmedDelivered, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["confirmed_delivered_store_and_forward"], prometheus.CounterValue, egressFlow.ConfirmedDeliveredStoreAndForward, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["confirmed_delivered_cut_through"], prometheus.CounterValue, egressFlow.ConfirmedDeliveredCutThrough, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(metricDesc["ClientMessageSpoolStats"]["unacked_messages"], prometheus.CounterValue, egressFlow.UnackedMessages, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
			}
		}

		body.Close()
	}

	return 1, nil
}

// Get statistics of all vpn's
func (e *Exporter) getVpnStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Vpn []struct {
						Name        string  `xml:"name"`
						LocalStatus string  `xml:"local-status"`
						Connections float64 `xml:"connections"`
						Stats       struct {
							DataRxByteCount   float64 `xml:"client-data-bytes-received"`
							DataRxMsgCount    float64 `xml:"client-data-messages-received"`
							DataTxByteCount   float64 `xml:"client-data-bytes-sent"`
							DataTxMsgCount    float64 `xml:"client-data-messages-sent"`
							AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
							AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
							AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
							AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
							RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
							RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
							TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
							TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
							IngressDiscards   struct {
								DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
							} `xml:"ingress-discards"`
							EgressDiscards struct {
								DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
							} `xml:"egress-discards"`
						} `xml:"stats"`
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><stats/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_rx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_tx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataTxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_rx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataRxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_tx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataTxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_rx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.IngressDiscards.DiscardedRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnStats"]["vpn_tx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.EgressDiscards.DiscardedTxMsgCount, vpn.Name)
	}

	return 1, nil
}

// Get statistics of bridges for all vpns
func (e *Exporter) getBridgeStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						Bridge []struct {
							BridgeName                string `xml:"bridge-name"`
							LocalVpnName              string `xml:"local-vpn-name"`
							ConnectedRemoteVpnName    string `xml:"connected-remote-vpn-name"`
							ConnectedRemoteRouterName string `xml:"connected-remote-router-name"`
							ConnectedViaAddr          string `xml:"connected-via-addr"`
							ConnectedViaInterface     string `xml:"connected-via-interface"`
							Redundancy                string `xml:"redundancy"`
							AdminState                string `xml:"admin-state"`
							ConnectionEstablisher     string `xml:"connection-establisher"`
							Client                    struct {
								ClientAddress    string  `xml:"client-address"`
								Name             string  `xml:"name"`
								NumSubscriptions float64 `xml:"num-subscriptions"`
								ClientId         float64 `xml:"client-id"`
								MessageVpn       string  `xml:"message-vpn"`
								SlowSubscriber   bool    `xml:"slow-subscriber"`
								ClientUsername   string  `xml:"client-username"`
								Stats            struct {
									TotalClientMessagesReceived         float64 `xml:"total-client-messages-received"`
									TotalClientMessagesSent             float64 `xml:"total-client-messages-sent"`
									ClientDataMessagesReceived          float64 `xml:"client-data-messages-received"`
									ClientDataMessagesSent              float64 `xml:"client-data-messages-sent"`
									ClientPersistentMessagesReceived    float64 `xml:"client-persistent-messages-received"`
									ClientPersistentMessagesSent        float64 `xml:"client-persistent-messages-sent"`
									ClientNonPersistentMessagesReceived float64 `xml:"client-non-persistent-messages-received"`
									ClientNonPersistentMessagesSent     float64 `xml:"client-non-persistent-messages-sent"`
									ClientDirectMessagesReceived        float64 `xml:"client-direct-messages-received"`
									ClientDirectMessagesSent            float64 `xml:"client-direct-messages-sent"`

									TotalClientBytesReceived         float64 `xml:"total-client-bytes-received"`
									TotalClientBytesSent             float64 `xml:"total-client-bytes-sent"`
									ClientDataBytesReceived          float64 `xml:"client-data-bytes-received"`
									ClientDataBytesSent              float64 `xml:"client-data-bytes-sent"`
									ClientPersistentBytesReceived    float64 `xml:"client-persistent-bytes-received"`
									ClientPersistentBytesSent        float64 `xml:"client-persistent-bytes-sent"`
									ClientNonPersistentBytesReceived float64 `xml:"client-non-persistent-bytes-received"`
									ClientNonPersistentBytesSent     float64 `xml:"client-non-persistent-bytes-sent"`
									ClientDirectBytesReceived        float64 `xml:"client-direct-bytes-received"`
									ClientDirectBytesSent            float64 `xml:"client-direct-bytes-sent"`

									LargeMessagesReceived       float64 `xml:"large-messages-received"`
									DeniedDuplicateClients      float64 `xml:"denied-duplicate-clients"`
									NotEnoughSpaceMsgsSent      float64 `xml:"not-enough-space-msgs-sent"`
									MaxExceededMsgsSent         float64 `xml:"max-exceeded-msgs-sent"`
									SubscribeClientNotFound     float64 `xml:"subscribe-client-not-found"`
									NotFoundMsgsSent            float64 `xml:"not-found-msgs-sent"`
									CurrentIngressRatePerSecond float64 `xml:"current-ingress-rate-per-second"`
									CurrentEgressRatePerSecond  float64 `xml:"current-egress-rate-per-second"`
									IngressDiscards             struct {
										TotalIngressDiscards       float64 `xml:"total-ingress-discards"`
										NoSubscriptionMatch        float64 `xml:"no-subscription-match"`
										TopicParseError            float64 `xml:"topic-parse-error"`
										ParseError                 float64 `xml:"parse-error"`
										MsgTooBig                  float64 `xml:"msg-too-big"`
										TtlExceeded                float64 `xml:"ttl-exceeded"`
										WebParseError              float64 `xml:"web-parse-error"`
										PublishTopicAcl            float64 `xml:"publish-topic-acl"`
										MsgSpoolDiscards           float64 `xml:"msg-spool-discards"`
										MessagePromotionCongestion float64 `xml:"message-promotion-congestion"`
										MessageSpoolCongestion     float64 `xml:"message-spool-congestion"`
									} `xml:"ingress-discards"`
									EgressDiscards struct {
										TotalEgressDiscards        float64 `xml:"total-egress-discards"`
										TransmitCongestion         float64 `xml:"transmit-congestion"`
										CompressionCongestion      float64 `xml:"compression-congestion"`
										MessageElided              float64 `xml:"message-elided"`
										TtlExceeded                float64 `xml:"ttl-exceeded"`
										PayloadCouldNotBeFormatted float64 `xml:"payload-could-not-be-formatted"`
										MessagePromotionCongestion float64 `xml:"message-promotion-congestion"`
										MessageSpoolCongestion     float64 `xml:"message-spool-congestion"`
										ClientNotConnected         float64 `xml:"client-not-connected"`
									} `xml:"egress-discards"`
									ManagedSubscriptions struct {
										AddBySubscriptionManager    float64 `xml:"add-by-subscription-manager"`
										RemoveBySubscriptionManager float64 `xml:"remove-by-subscription-manager"`
									} `xml:"managed-subscriptions"`
								} `xml:"stats"`
							} `xml:"client"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><bridge><bridge-name-pattern>" + itemFilter + "</bridge-name-pattern><vpn-name-pattern>" + vpnFilter + "</vpn-name-pattern><stats/></bridge></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape BridgeSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml BridgeSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}
	for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
		bridgeName := bridge.BridgeName
		vpnName := bridge.LocalVpnName
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_num_subscriptions"], prometheus.GaugeValue, bridge.Client.NumSubscriptions, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(bridge.Client.SlowSubscriber), vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_client_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_client_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_data_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_data_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_persistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_persistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_nonpersistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_nonpersistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_direct_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_direct_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesSent, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_client_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_client_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_data_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_data_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_persistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_persistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_nonpersistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_nonpersistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_direct_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_direct_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesSent, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_client_large_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.LargeMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_denied_duplicate_clients"], prometheus.GaugeValue, bridge.Client.Stats.DeniedDuplicateClients, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_not_enough_space_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotEnoughSpaceMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_max_exceeded_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.MaxExceededMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_subscribe_client_not_found"], prometheus.GaugeValue, bridge.Client.Stats.SubscribeClientNotFound, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_not_found_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotFoundMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_current_ingress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentIngressRatePerSecond, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_current_egress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentEgressRatePerSecond, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_ingress_discards"], prometheus.GaugeValue, bridge.Client.Stats.IngressDiscards.TotalIngressDiscards, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricDesc["BridgeStats"]["bridge_total_egress_discards"], prometheus.GaugeValue, bridge.Client.Stats.EgressDiscards.TotalEgressDiscards, vpnName, bridgeName)
	}
	return 1, nil
}

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
// Deprecated: in facor of: getQueueStatsSemp1
func (e *Exporter) getQueueRatesSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Rates struct {
								Qendpt struct {
									AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
									AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
									AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
									AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
									RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
									RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
									TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
									TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								} `xml:"qendpt-data-rates"`
							} `xml:"rates"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueRatesSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueRatesSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_rx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_tx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_rx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_tx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_rx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_tx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_rx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueRates"]["queue_tx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxByteRate, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Stats struct {
								MessageSpoolStats struct {
									TotalByteSpooled       float64 `xml:"total-bytes-spooled"`
									TotalMsgSpooled        float64 `xml:"total-messages-spooled"`
									MsgRedelivered         float64 `xml:"messages-redelivered"`
									MsgRetransmit          float64 `xml:"messages-transport-retransmit"`
									SpoolUsageExceeded     float64 `xml:"spool-usage-exceeded"`
									MsgSizeExceeded        float64 `xml:"max-message-size-exceeded"`
									SpoolShutdownDiscard   float64 `xml:"spool-shutdown-discard"`
									DestinationGroupError  float64 `xml:"destination-group-error"`
									LowPrioMsgDiscard      float64 `xml:"low-priority-msg-congestion-discard"`
									Deleted                float64 `xml:"total-deleted-messages"`
									TtlDisacarded          float64 `xml:"total-ttl-expired-discard-messages"`
									TtlDmq                 float64 `xml:"total-ttl-expired-to-dmq-messages"`
									TtlDmqFailed           float64 `xml:"total-ttl-expired-to-dmq-failures"`
									MaxRedeliveryDiscarded float64 `xml:"max-redelivery-exceeded-discard-messages"`
									MaxRedeliveryDmq       float64 `xml:"max-redelivery-exceeded-to-dmq-messages"`
									MaxRedeliveryDmqFailed float64 `xml:"max-redelivery-exceeded-to-dmq-failures"`
								} `xml:"message-spool-stats"`
							} `xml:"stats"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><stats/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC
		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["total_bytes_spooled"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.TotalByteSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["total_messages_spooled"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.TotalMsgSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["messages_redelivered"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgRedelivered, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["messages_transport_retransmited"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgRetransmit, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["spool_usage_exceeded"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.SpoolUsageExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["max_message_size_exceeded"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgSizeExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueStats"]["total_deleted_messages"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.Deleted, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}

// Get some statistics for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueDetailsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName      string  `xml:"message-vpn"`
								Quota           float64 `xml:"quota"`
								Usage           float64 `xml:"current-spool-usage-in-mb"`
								SpooledMsgCount float64 `xml:"num-messages-spooled"`
								BindCount       float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueDetails"]["queue_spool_quota_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Quota*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueDetails"]["queue_spool_usage_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Usage*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueDetails"]["queue_spool_usage_msgs"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricDesc["QueueDetails"]["queue_binds"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}

// Get rates for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topic-endpoints
// Deprecated: in facor of: getTopicEndpointStatsSemp1
func (e *Exporter) getTopicEndpointRatesSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Rates struct {
								Qendpt struct {
									AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
									AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
									AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
									AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
									RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
									RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
									TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
									TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								} `xml:"qendpt-data-rates"`
							} `xml:"rates"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointRatesSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointRatesSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["rx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["tx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["rx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["tx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["rx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["tx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["rx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointRates"]["tx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}

// Get rates for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topc-endpoints
func (e *Exporter) getTopicEndpointStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Stats struct {
								MessageSpoolStats struct {
									TotalByteSpooled       float64 `xml:"total-bytes-spooled"`
									TotalMsgSpooled        float64 `xml:"total-messages-spooled"`
									MsgRedelivered         float64 `xml:"messages-redelivered"`
									MsgRetransmit          float64 `xml:"messages-transport-retransmit"`
									SpoolUsageExceeded     float64 `xml:"spool-usage-exceeded"`
									MsgSizeExceeded        float64 `xml:"max-message-size-exceeded"`
									SpoolShutdownDiscard   float64 `xml:"spool-shutdown-discard"`
									DestinationGroupError  float64 `xml:"destination-group-error"`
									LowPrioMsgDiscard      float64 `xml:"low-priority-msg-congestion-discard"`
									Deleted                float64 `xml:"total-deleted-messages"`
									TtlDisacarded          float64 `xml:"total-ttl-expired-discard-messages"`
									TtlDmq                 float64 `xml:"total-ttl-expired-to-dmq-messages"`
									TtlDmqFailed           float64 `xml:"total-ttl-expired-to-dmq-failures"`
									MaxRedeliveryDiscarded float64 `xml:"max-redelivery-exceeded-discard-messages"`
									MaxRedeliveryDmq       float64 `xml:"max-redelivery-exceeded-to-dmq-messages"`
									MaxRedeliveryDmqFailed float64 `xml:"max-redelivery-exceeded-to-dmq-failures"`
								} `xml:"message-spool-stats"`
							} `xml:"stats"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><stats/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointStatsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["total_bytes_spooled"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.TotalByteSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["total_messages_spooled"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.TotalMsgSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["messages_redelivered"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgRedelivered, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["messages_transport_retransmited"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgRetransmit, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["spool_usage_exceeded"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.SpoolUsageExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["max_message_size_exceeded"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.MsgSizeExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointStats"]["total_deleted_messages"], prometheus.GaugeValue, topicEndpoint.Stats.MessageSpoolStats.Deleted, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}

// Get some statistics for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topic endpoints
func (e *Exporter) getTopicEndpointDetailsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
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
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", e.config.scrapeURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointDetails"]["spool_quota_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Quota*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointDetails"]["spool_usage_bytes"], prometheus.GaugeValue, math.Round(topicEndpoint.Info.Usage*1048576.0), topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointDetails"]["spool_usage_msgs"], prometheus.GaugeValue, topicEndpoint.Info.SpooledMsgCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(metricDesc["TopicEndpointDetails"]["binds"], prometheus.GaugeValue, topicEndpoint.Info.BindCount, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}

// Replication Config and status
func (e *Exporter) getVpnSpoolSemp1(ch chan<- prometheus.Metric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageSpool struct {
					MessageVpn struct {
						Vpn []struct {
							Name                string  `xml:"name"`
							SpooledMsgCount     float64 `xml:"current-messages-spooled"`
							SpoolUsageCurrentMb float64 `xml:"current-spool-usage-mb"`
							SpoolUsageMaxMb     float64 `xml:"maximum-spool-usage-mb"`
						} `xml:"vpn"`
					} `xml:"message-vpn"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool><vpn-name>" + vpnFilter + "</vpn-name></message-spool></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, vpn := range target.RPC.Show.MessageSpool.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnSpool"]["vpn_spool_quota_bytes"], prometheus.GaugeValue, vpn.SpoolUsageMaxMb*1024*1024, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnSpool"]["vpn_spool_usage_bytes"], prometheus.GaugeValue, vpn.SpoolUsageCurrentMb*1024*1024, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricDesc["VpnSpool"]["vpn_spool_usage_msgs"], prometheus.GaugeValue, vpn.SpooledMsgCount, vpn.Name)
	}

	return 1, nil
}

// Cluster link states of broker
func (e *Exporter) getClusterLinksSemp1(ch chan<- prometheus.Metric, clusterFilter string, linkFilter string) (ok float64, err error) {
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
		} `xml:"execute-result"`
	}

	command := "<rpc><show><cluster><cluster-name-pattern>" + clusterFilter + "</cluster-name-pattern><link-name-pattern>" + linkFilter + "</link-name-pattern></cluster></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape ClusterLinksSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml ClusterLinksSemp1", "err", err, "broker", e.config.scrapeURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.config.scrapeURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, cluster := range target.RPC.Show.Cluster.Clusters.Cluster {
		for _, link := range cluster.Links.Link {
			ch <- prometheus.MustNewConstMetric(metricDesc["ClusterLinks"]["enabled"], prometheus.GaugeValue, encodeMetricMulti(link.Enabled, []string{"false", "true", "n/a"}), cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClusterLinks"]["oper_up"], prometheus.GaugeValue, encodeMetricMulti(link.Operational, []string{"false", "true", "n/a"}), cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
			ch <- prometheus.MustNewConstMetric(metricDesc["ClusterLinks"]["oper_uptime"], prometheus.GaugeValue, link.UptimeInSeconds, cluster.ClusterName, cluster.NodeName, link.RemoteClusterName, link.RemoteNodeName)
		}
	}

	return 1, nil
}

// Encodes string to 0,1,2,... metric
func encodeMetricMulti(item string, refItems []string) float64 {
	uItem := strings.ToUpper(item)
	for i, s := range refItems {
		if uItem == strings.ToUpper(s) {
			return float64(i)
		}
	}
	return -1
}

// Encodes bool into 0,1 metric
func encodeMetricBool(item bool) float64 {
	if item {
		return 1
	}
	return 0
}

// Collection of configs
type config struct {
	listenAddr     string
	enableTLS      bool
	certificate    string
	privateKey     string
	scrapeURI      string
	username       string
	password       string
	sslVerify      bool
	useSystemProxy bool
	timeout        time.Duration
	dataSource     []DataSource
}

// getListenURI returns the `listenAddr` with proper protocol (http/https),
// based on the `enableTLS` configuration parameter
func (c *config) getListenURI() string {
	if c.enableTLS {
		return "https://" + c.listenAddr
	} else {
		return "http://" + c.listenAddr
	}
}

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config    config
	logger    log.Logger
	lastError error
}

type DataSource struct {
	name       string
	vpnFilter  string
	itemFilter string
}

func (dataSource DataSource) String() string {
	return fmt.Sprintf("%s=%s|%s", dataSource.name, dataSource.vpnFilter, dataSource.itemFilter)
}

func logDataSource(dataSources []DataSource) string {
	dS := make([]string, len(dataSources))
	for index, dataSource := range dataSources {
		dS[index] = dataSource.String()
	}
	return strings.Join(dS, "&")
}

func parseConfigBool(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) bool {
	var ok = true
	s := parseConfigString(cfg, logger, iniSection, iniKey, envKey, &ok)
	if ok {
		val, err := strconv.ParseBool(s)
		if err == nil {
			return val
		}
		_ = level.Error(logger).Log("msg", "Config param invalid", "iniKey", iniKey, "envKey", envKey)
	}
	*okp = false
	return false
}

func parseConfigDuration(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) time.Duration {
	var ok = true
	s := parseConfigString(cfg, logger, iniSection, iniKey, envKey, &ok)
	if ok {
		val, err := time.ParseDuration(s)
		if err == nil {
			return val
		}
		_ = level.Error(logger).Log("msg", "Config param invalid", "iniKey", iniKey, "envKey", envKey)
	}
	*okp = false
	return 0
}

func parseConfigString(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) string {
	s := os.Getenv(envKey)
	if len(s) > 0 {
		return s
	}

	if cfg != nil {
		s := cfg.Section(iniSection).Key(iniKey).String()
		if len(s) > 0 {
			return s
		}
	}

	_ = level.Error(logger).Log("msg", "Config param missing", "iniKey", iniKey, "envKey", envKey)
	*okp = false
	return ""
}

func parseConfig(configFile string, conf *config, logger log.Logger) (bool, map[string][]DataSource) {
	var cfg *ini.File = nil
	var err interface{}
	var oki = true

	if len(configFile) > 0 {
		opts := ini.LoadOptions{
			AllowBooleanKeys: true,
		}
		cfg, err = ini.LoadSources(opts, configFile)
		if err != nil {
			_ = level.Error(logger).Log("msg", "Can't open config file", "err", err)
			return false, nil
		}
	}

	conf.listenAddr = parseConfigString(cfg, logger, "solace", "listenAddr", "SOLACE_LISTEN_ADDR", &oki)
	conf.enableTLS = parseConfigBool(cfg, logger, "solace", "enableTLS", "SOLACE_LISTEN_TLS", &oki)
	conf.certificate = parseConfigString(cfg, logger, "solace", "certificate", "SOLACE_SERVER_CERT", &oki)
	conf.privateKey = parseConfigString(cfg, logger, "solace", "privateKey", "SOLACE_PRIVATE_KEY", &oki)
	conf.scrapeURI = parseConfigString(cfg, logger, "solace", "scrapeUri", "SOLACE_SCRAPE_URI", &oki)
	conf.username = parseConfigString(cfg, logger, "solace", "username", "SOLACE_USERNAME", &oki)
	conf.password = parseConfigString(cfg, logger, "solace", "password", "SOLACE_PASSWORD", &oki)
	conf.timeout = parseConfigDuration(cfg, logger, "solace", "timeout", "SOLACE_TIMEOUT", &oki)
	conf.sslVerify = parseConfigBool(cfg, logger, "solace", "sslVerify", "SOLACE_SSL_VERIFY", &oki)

	endpoints := make(map[string][]DataSource)
	if cfg != nil {
		var scrapeTargetRe = regexp.MustCompile(`^(\w+)(\.\d+)?$`)
		for _, section := range cfg.Sections() {
			if strings.HasPrefix(section.Name(), "endpoint.") {
				endpointName := strings.TrimPrefix(section.Name(), "endpoint.")

				var dataSource []DataSource
				for _, key := range section.Keys() {
					scrapeTarget := scrapeTargetRe.ReplaceAllString(key.Name(), `$1`)

					parts := strings.Split(key.String(), "|")
					if len(parts) != 2 {
						_ = level.Error(logger).Log("msg", "Exactly one | expected. Use VPN wildcard. |. Item wildcard.", "endpointName", endpointName, "key", key.Name(), "value", key.String())
					} else {
						dataSource = append(dataSource, DataSource{
							name:       scrapeTarget,
							vpnFilter:  parts[0],
							itemFilter: parts[1],
						})
					}
				}

				endpoints[endpointName] = dataSource
			}
		}
	}

	return oki, endpoints
}

// NewExporter returns an initialized Exporter.
func NewExporter(logger log.Logger, conf config) *Exporter {
	return &Exporter{
		logger:    logger,
		config:    conf,
		lastError: nil,
	}
}

// Describe describes all the metrics ever exported by the Solace exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, dataSource := range e.config.dataSource {
		if metricDescItems, ok := metricDesc[dataSource.name]; ok {
			for _, m := range metricDescItems {
				ch <- m
			}
		} else {
			permittedNames := make([]string, 0, len(metricDesc))
			for index := range metricDesc {
				permittedNames = append(permittedNames, index)
			}
			_ = level.Error(e.logger).Log("msg", "Unexpected data source name: "+dataSource.name, "permitted", strings.Join(permittedNames, ","))
		}

	}
}

// Collect fetches the stats from configured Solace location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	var up float64 = 1
	var err error = nil

	for _, dataSource := range e.config.dataSource {
		if up < 1 {
			if err != nil {
				ch <- prometheus.MustNewConstMetric(metricDesc["Global"]["up"], prometheus.GaugeValue, 0, err.Error())
			} else {
				ch <- prometheus.MustNewConstMetric(metricDesc["Global"]["up"], prometheus.GaugeValue, 0, "Unknown")
			}
			return
		}

		switch dataSource.name {
		case "Version":
			up, err = e.getVersionSemp1(ch)
		case "Health":
			up, err = e.getHealthSemp1(ch)
		case "StorageElement":
			up, err = e.getStorageElementSemp1(ch, dataSource.itemFilter)
		case "Disk":
			up, err = e.getDiskSemp1(ch)
		case "Memory":
			up, err = e.getMemorySemp1(ch)
		case "Interface":
			up, err = e.getInterfaceSemp1(ch, dataSource.itemFilter)
		case "GlobalStats":
			up, err = e.getGlobalStatsSemp1(ch)
		case "Spool":
			up, err = e.getSpoolSemp1(ch)
		case "Redundancy":
			up, err = e.getRedundancySemp1(ch)
		case "ReplicationStats":
			up, err = e.getReplicationStatsSemp1(ch)
		case "ConfigSyncRouter":
			up, err = e.getConfigSyncRouterSemp1(ch)
		case "Vpn":
			up, err = e.getVpnSemp1(ch, dataSource.vpnFilter)
		case "VpnReplication":
			up, err = e.getVpnReplicationSemp1(ch, dataSource.vpnFilter)
		case "ConfigSyncVpn":
			up, err = e.getConfigSyncVpnSemp1(ch, dataSource.vpnFilter)
		case "Bridge":
			up, err = e.getBridgeSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "VpnSpool":
			up, err = e.getVpnSpoolSemp1(ch, dataSource.vpnFilter)
		case "Client":
			up, err = e.getClientSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "ClientSlowSubscriber":
			up, err = e.getClientSlowSubscriberSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "ClientStats":
			up, err = e.getClientStatsSemp1(ch, dataSource.vpnFilter)
		case "ClientMessageSpoolStats":
			up, err = e.getClientMessageSpoolStatsSemp1(ch, dataSource.vpnFilter)
		case "ClusterLinks":
			up, err = e.getClusterLinksSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)

		case "VpnStats":
			up, err = e.getVpnStatsSemp1(ch, dataSource.vpnFilter)
		case "BridgeStats":
			up, err = e.getBridgeStatsSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "QueueRates":
			up, err = e.getQueueRatesSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "QueueStats":
			up, err = e.getQueueStatsSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "QueueDetails":
			up, err = e.getQueueDetailsSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "TopicEndpointRates":
			up, err = e.getTopicEndpointRatesSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "TopicEndpointStats":
			up, err = e.getTopicEndpointStatsSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		case "TopicEndpointDetails":
			up, err = e.getTopicEndpointDetailsSemp1(ch, dataSource.vpnFilter, dataSource.itemFilter)
		}
	}
	ch <- prometheus.MustNewConstMetric(metricDesc["Global"]["up"], prometheus.GaugeValue, 1, "")
}

func main() {

	kingpin.HelpFlag.Short('h')

	promlogConfig := promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}
	_ = promlogConfig.Level.Set("info")
	_ = promlogConfig.Format.Set("logfmt")
	flag.AddFlags(kingpin.CommandLine, &promlogConfig)

	configFile := kingpin.Flag(
		"config-file",
		"Path and name of ini file with configuration settings. See sample file solace_prometheus_exporter.ini.",
	).String()
	enableTLS := kingpin.Flag(
		"enable-tls",
		"Set to true, to start listenAddr as TLS port. Make sure to provide valid server certificate and private key files.",
	).Bool()
	certfile := kingpin.Flag(
		"certificate",
		"If using TLS, you must provide a valid server certificate in PEM format. Can be set via config file, cli parameter or env variable.",
	).ExistingFile()
	privateKey := kingpin.Flag(
		"private-key",
		"If using TLS, you must provide a valid private key in PEM format. Can be set via config file, cli parameter or env variable.",
	).ExistingFile()

	kingpin.Parse()

	logger := promlog.New(&promlogConfig)

	var conf config
	oki, endpoints := parseConfig(*configFile, &conf, logger)
	if !oki {
		os.Exit(1)
	}

	_ = level.Info(logger).Log("msg", "Starting solace_prometheus_exporter", "version", solaceExporterVersion)
	_ = level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	if *enableTLS {
		conf.enableTLS = *enableTLS
	}
	if len(*certfile) > 0 {
		conf.certificate = *certfile
	}
	if len(*privateKey) > 0 {
		conf.privateKey = *privateKey
	}

	_ = level.Info(logger).Log("msg", "Scraping",
		"listenAddr", conf.getListenURI(),
		"scrapeURI", conf.scrapeURI,
		"username", conf.username,
		"sslVerify", conf.sslVerify,
		"timeout", conf.timeout)

	// Test scrape to check if broker can be accessed. If it fails it prints a warn to the log file.
	// Note that failure is not fatal, as broker might not have started up yet.
	conf.timeout, _ = time.ParseDuration("2s") // Don't delay startup too much

	// Configure endpoints
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		doHandle(w, r, nil, conf, logger)
	})

	declareHandlerFromConfig := func(urlPath string, dataSource []DataSource) {
		_ = level.Info(logger).Log("msg", "Register handler from config", "handler", "/"+urlPath, "dataSource", logDataSource(dataSource))
		http.HandleFunc("/"+urlPath, func(w http.ResponseWriter, r *http.Request) {
			doHandle(w, r, dataSource, conf, logger)
		})
	}
	for urlPath, dataSource := range endpoints {
		declareHandlerFromConfig(urlPath, dataSource)
	}

	http.HandleFunc("/solace", func(w http.ResponseWriter, r *http.Request) {
		var err = r.ParseForm()
		if err != nil {
			_ = level.Error(logger).Log("msg", "Can not parse the request parameter", "err", err)
			return
		}

		var dataSource []DataSource
		for key, values := range r.Form {
			if strings.HasPrefix(key, "m.") {
				for _, value := range values {
					parts := strings.Split(value, "|")
					if len(parts) != 2 {
						_ = level.Error(logger).Log("msg", "Exactly one | expected. Use VPN wildcard. |. Item wildcard.", "key", key, "value", value)
					} else {
						dataSource = append(dataSource, DataSource{
							name:       strings.TrimPrefix(key, "m."),
							vpnFilter:  parts[0],
							itemFilter: parts[1],
						})
					}
				}
			}
		}

		doHandle(w, r, dataSource, conf, logger)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var endpointsDoc bytes.Buffer
		for urlPath, dataSources := range endpoints {
			endpointsDoc.WriteString("<li><a href='/" + urlPath + "'>Custom Exporter " + urlPath + " -> " + logDataSource(dataSources) + "</a></li>")
		}

		_, _ = w.Write([]byte(`<html>
            <head><title>Solace Exporter</title></head>
            <body>
            <h1>Solace Exporter</h1>
            <ul style="list-style: none;">
                <li><a href='` + "/metrics" + `'>Exporter Metrics</a></li>
				` + endpointsDoc.String() + `
				<li><a href='` + "/solace?m.ClientStats=*|*&m.VpnStats=*|*&m.BridgeStats=*|*&m.QueueRates=*|*" + `'>Solace Broker</a>
				<br>
				<p>Configure the data you want ot receive, via HTTP GET parameters.
				<br>Please use in format &quot;m.ClientStats=*|*&m.VpnStats=*|*&quot; 
				<br>Here is &quot;m.&quot; the prefix.
				<br>Here is &quot;ClientStats&quot; the scrape target.
				<br>The first asterisk the VPN filter and the second asterisk the item filter.
				Not all scrape targets support filter.
				<br>Scrape targets:<br>
				<table><tr><th>scape target</th><th>vpn filter supports</th><th>item filter supported</th><th>performance</th><tr>
					<tr><td>Version</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Health</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>StorageElement</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Disk</td><td>no</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>Memory</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Interface</td><td>no</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>GlobalStats</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Spool</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Redundancy (only for HA broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ConfigSyncRouter (only for HA broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ReplicationStats (only for DR replication broker)</td><td>no</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Vpn</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>VpnReplication</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>ConfigSyncVpn (only for HA broker)</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Bridge</td><td>yes</td><td>yes</td><td>dont harm broker</td></tr>
					<tr><td>VpnSpool</td><td>yes</td><td>no</td><td>dont harm broker</td></tr>
					<tr><td>Client</td><td>yes</td><td>yes</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientSlowSubscriber</td><td>yes</td><td>yes</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientStats</td><td>yes</td><td>no</td><td>may harm broker if many clients</td></tr>
					<tr><td>ClientMessageSpoolStats</td><td>yes</td><td>yes</td><td>no</td></tr>
					<tr><td>ClusterLinks</td><td>yes</td><td>no</td><td>may harm broker if many clients</td></tr>
					<tr><td>VpnStats</td><td>yes</td><td>no</td><td>has a very small performance down site</td></tr>
					<tr><td>BridgeStats</td><td>yes</td><td>yes</td><td>has a very small performance down site</td></tr>
					<tr><td>QueueRates</td><td>yes</td><td>yes</td><td>DEPRECATED: may harm broker if many queues</td></tr>
					<tr><td>QueueStats</td><td>yes</td><td>yes</td><td>may harm broker if many queues</td></tr>
					<tr><td>QueueDetails</td><td>yes</td><td>yes</td><td>may harm broker if many queues</td></tr>
					<tr><td>TopicEndpointRates</td><td>yes</td><td>yes</td><td>DEPRECATED: may harm broker if many topic-endpoints</td></tr>
					<tr><td>TopicEndpointStats</td><td>yes</td><td>yes</td><td>may harm broker if many topic-endpoints</td></tr>
					<tr><td>TopicEndpointDetails</td><td>yes</td><td>yes</td><td>may harm broker if many topic-endpoints</td></tr>
				</table>
				<br>
				</p>
				</li>
            <ul>
            </body>
            </html>`))
	})

	// start server
	if conf.enableTLS {
		if err := http.ListenAndServeTLS(conf.listenAddr, conf.certificate, conf.privateKey, nil); err != nil {
			_ = level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	} else {
		if err := http.ListenAndServe(conf.listenAddr, nil); err != nil {
			_ = level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
			os.Exit(2)
		}
	}

}

func doHandle(w http.ResponseWriter, r *http.Request, dataSource []DataSource, conf config, logger log.Logger) (resultCode string) {

	if dataSource == nil {
		handler := promhttp.Handler()
		handler.ServeHTTP(w, r)
	} else {
		// Exporter for endpoint
		conf.dataSource = dataSource
		username := r.FormValue("username")
		password := r.FormValue("password")
		scrapeURI := r.FormValue("scrapeURI")
		timeout := r.FormValue("timeout")
		if len(username) > 0 {
			conf.username = username
		}
		if len(password) > 0 {
			conf.password = password
		}
		if len(scrapeURI) > 0 {
			conf.scrapeURI = scrapeURI
		}
		if len(timeout) > 0 {
			var err error
			conf.timeout, err = time.ParseDuration(timeout)
			if err != nil {
				_ = level.Error(logger).Log("msg", "Per HTTP given timeout parameter is not valid", "err", err, "timeout", timeout)
			}
		}

		_ = level.Info(logger).Log("msg", "handle http request", "dataSource", logDataSource(dataSource), "scrapeURI", conf.scrapeURI)

		exporter := NewExporter(logger, conf)
		registry := prometheus.NewRegistry()
		registry.MustRegister(exporter)
		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	}
	return w.Header().Get("status")
}
