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
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
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

type metrics map[string]*prometheus.Desc

var (
	solaceExporterVersion = float64(1002002)

	variableLabelsRedundancy      = []string{"mate_name"}
	variableLabelsVpn             = []string{"vpn_name"}
	variableLabelsVpnClient       = []string{"vpn_name", "client_name", "client_username"}
	variableLabelsVpnQueue        = []string{"vpn_name", "queue_name"}
	variableLabelsBridge          = []string{"vpn_name", "bridge_name"}
	variableLabelsConfigSyncTable = []string{"table_name"}

	solaceUp = prometheus.NewDesc(namespace+"_"+"up", "Was the last scrape of Solace broker successful.", nil, nil)
)

var metricsStd = metrics{
	// version
	"system_version_currentload":      prometheus.NewDesc(namespace+"_"+"system_version_currentload", "Solace Version as WWWXXXYYYZZZ ", nil, nil),
	"system_version_uptime_totalsecs": prometheus.NewDesc(namespace+"_"+"system_version_uptime_totalsecs", "Broker uptime in seconds ", nil, nil),
	"exporter_version_current":        prometheus.NewDesc(namespace+"_"+"exporter_version_current", "Exporter Version as XXXYYYZZZ", nil, nil),

	// redundancy
	"system_redundancy_up":           prometheus.NewDesc(namespace+"_"+"system_redundancy_up", "Is redundancy up? (0=Down, 1=Up).", variableLabelsRedundancy, nil),
	"system_redundancy_config":       prometheus.NewDesc(namespace+"_"+"system_redundancy_config", "Redundancy configuration (0-Disabled, 1-Enabled, 2-Shutdown)", variableLabelsRedundancy, nil),
	"system_redundancy_role":         prometheus.NewDesc(namespace+"_"+"system_redundancy_role", "Redundancy role (0=Backup, 1=Primary, 2=Monitor, 3-Undefined).", variableLabelsRedundancy, nil),
	"system_redundancy_local_active": prometheus.NewDesc(namespace+"_"+"system_redundancy_local_active", "Is local node the active messaging node? (0=not active, 1=active).", variableLabelsRedundancy, nil),

	// system
	"system_spool_quota_bytes":             prometheus.NewDesc(namespace+"_"+"system_spool_quota_bytes", "Spool configured max disk usage.", nil, nil),
	"system_spool_quota_msgs":              prometheus.NewDesc(namespace+"_"+"system_spool_quota_msgs", "Spool configured max number of messages.", nil, nil),
	"system_spool_usage_bytes":             prometheus.NewDesc(namespace+"_"+"system_spool_usage_bytes", "Spool total persisted usage.", nil, nil),
	"system_spool_usage_msgs":              prometheus.NewDesc(namespace+"_"+"system_spool_usage_msgs", "Spool total number of persisted messages.", nil, nil),
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

	// config sync
	"configsync_table_type":               prometheus.NewDesc(namespace+"_"+"configsync_table_type", "Config Sync Resource Type (0-Router, 1-Vpn)", variableLabelsConfigSyncTable, nil),
	"configsync_table_timeinstateseconds": prometheus.NewDesc(namespace+"_"+"configsync_table_timeinstateseconds", "Config Sync Time in State", variableLabelsConfigSyncTable, nil),
	"configsync_table_ownership":          prometheus.NewDesc(namespace+"_"+"configsync_table_ownership", "Config Sync Ownership (0-Master, 1-Slave, 2-Unknown)", variableLabelsConfigSyncTable, nil),
	"configsync_table_syncstate":          prometheus.NewDesc(namespace+"_"+"configsync_table_syncstate", "Config Sync State (0-Down, 1-Up)", variableLabelsConfigSyncTable, nil),

	// vpn
	"vpn_local_status":                             prometheus.NewDesc(namespace+"_"+"vpn_local_status", "Local status (0=Down, 1=Up)", variableLabelsVpn, nil),
	"vpn_connections":                              prometheus.NewDesc(namespace+"_"+"vpn_connections", "Number of connections.", variableLabelsVpn, nil),
	"vpn_rx_msgs_total":                            prometheus.NewDesc(namespace+"_"+"vpn_rx_msgs_total", "Number of received messages.", variableLabelsVpn, nil),
	"vpn_tx_msgs_total":                            prometheus.NewDesc(namespace+"_"+"vpn_tx_msgs_total", "Number of transmitted messages.", variableLabelsVpn, nil),
	"vpn_rx_bytes_total":                           prometheus.NewDesc(namespace+"_"+"vpn_rx_bytes_total", "Number of received bytes.", variableLabelsVpn, nil),
	"vpn_tx_bytes_total":                           prometheus.NewDesc(namespace+"_"+"vpn_tx_bytes_total", "Number of transmitted bytes.", variableLabelsVpn, nil),
	"vpn_rx_discarded_msgs_total":                  prometheus.NewDesc(namespace+"_"+"vpn_rx_discarded_msgs_total", "Number of discarded received messages.", variableLabelsVpn, nil),
	"vpn_tx_discarded_msgs_total":                  prometheus.NewDesc(namespace+"_"+"vpn_tx_discarded_msgs_total", "Number of discarded transmitted messages.", variableLabelsVpn, nil),
	"vpn_replication_admin_state":                  prometheus.NewDesc(namespace+"_"+"vpn_replication_admin_state", "Replication Admin Status (0-shutdown, 1-enabled, 2-n/a)", variableLabelsVpn, nil),
	"vpn_replication_config_state":                 prometheus.NewDesc(namespace+"_"+"vpn_replication_config_state", "Replication Config Status (0-standby, 1-active, 2-n/a)", variableLabelsVpn, nil),
	"vpn_replication_transaction_replication_mode": prometheus.NewDesc(namespace+"_"+"vpn_replication_transaction_replication_mode", "Replication Tx Replication Mode (0-async, 1-sync)", variableLabelsVpn, nil),

	//bridges
	"bridges_num_total_bridges":                         prometheus.NewDesc(namespace+"_"+"bridges_num_total_bridges", "Number of Bridges", nil, nil),
	"bridges_max_num_total_bridges":                     prometheus.NewDesc(namespace+"_"+"bridges_max_num_total_bridges", "Max number of Bridges", nil, nil),
	"bridges_num_local_bridges":                         prometheus.NewDesc(namespace+"_"+"bridges_num_local_bridges", "Number of Local Bridges", nil, nil),
	"bridges_max_num_local_bridges":                     prometheus.NewDesc(namespace+"_"+"bridges_max_num_local_bridges", "Max number of Local Bridges", nil, nil),
	"bridges_num_remote_bridges":                        prometheus.NewDesc(namespace+"_"+"bridges_num_remote_bridges", "Number of Remote Bridges", nil, nil),
	"bridges_max_num_remote_bridges":                    prometheus.NewDesc(namespace+"_"+"bridges_max_num_remote_bridges", "Max number of Remote Bridges", nil, nil),
	"bridges_num_total_remote_bridge_subscriptions":     prometheus.NewDesc(namespace+"_"+"bridges_num_total_remote_bridge_subscriptions", "Total number of Remote Bridge Subscription", nil, nil),
	"bridges_max_num_total_remote_bridge_subscriptions": prometheus.NewDesc(namespace+"_"+"bridges_max_num_total_remote_bridge_subscriptions", "Max total number of Remote Bridge Subscription", nil, nil),

	// bridge
	"bridge_admin_state":                            prometheus.NewDesc(namespace+"_"+"bridge_admin_state", "Bridge Administrative State (0-Enabled 1-Disabled)", variableLabelsBridge, nil),
	"bridge_connection_establisher":                 prometheus.NewDesc(namespace+"_"+"bridge_connection_establisher", "Connection Establisher (0-NotApplicable, 1-Local, 2-Remote)", variableLabelsBridge, nil),
	"bridge_inbound_operational_state":              prometheus.NewDesc(namespace+"_"+"bridge_inbound_operational_state", "Inbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge, nil),
	"bridge_inbound_operational_failure_reason":     prometheus.NewDesc(namespace+"_"+"bridge_inbound_operational_failure_reason", "Inbound Ops Failure Reason (various very long codes)", variableLabelsBridge, nil),
	"bridge_outbound_operational_state":             prometheus.NewDesc(namespace+"_"+"bridge_outbound_operational_state", "Outbound Ops State (0-Init, 1-Shutdown, 2-NoShutdown, 3-Prepare, 4-Prepare-WaitToConnect, 5-Prepare-FetchingDNS, 6-NotReady, 7-NotReady-Connecting, 8-NotReady-Handshaking, 9-NotReady-WaitNext, 10-NotReady-WaitReuse, 11-NotRead-WaitBridgeVersionMismatch, 12-NotReady-WaitCleanup, 13-Ready, 14-Ready-Subscribing, 15-Ready-InSync, 16-NotApplicable, 17-Invalid)", variableLabelsBridge, nil),
	"bridge_queue_operational_state":                prometheus.NewDesc(namespace+"_"+"bridge_queue_operational_state", "Queue Ops State (0-NotApplicable, 1-Bound, 2-Unbound)", variableLabelsBridge, nil),
	"bridge_redundancy":                             prometheus.NewDesc(namespace+"_"+"bridge_redundancy", "Bridge Redundancy (0-NotApplicable, 1-auto, 2-primary, 3-backup, 4-static, 5-none)", variableLabelsBridge, nil),
	"bridge_connection_uptime_in_seconds":           prometheus.NewDesc(namespace+"_"+"bridge_connection_uptime_in_seconds", "Connection Uptime (s)", variableLabelsBridge, nil),
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
	"bridge_total_client_bytess_sent":               prometheus.NewDesc(namespace+"_"+"bridge_total_client_bytess_sent", "Bridge Total Client Bytes sent", variableLabelsBridge, nil),
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
}

var metricsDet = metrics{
	"client_rx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_rx_msgs_total", "Number of received messages.", variableLabelsVpnClient, nil),
	"client_tx_msgs_total":           prometheus.NewDesc(namespace+"_"+"client_tx_msgs_total", "Number of transmitted messages.", variableLabelsVpnClient, nil),
	"client_rx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_rx_bytes_total", "Number of received bytes.", variableLabelsVpnClient, nil),
	"client_tx_bytes_total":          prometheus.NewDesc(namespace+"_"+"client_tx_bytes_total", "Number of transmitted bytes.", variableLabelsVpnClient, nil),
	"client_rx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_rx_discarded_msgs_total", "Number of discarded received messages.", variableLabelsVpnClient, nil),
	"client_tx_discarded_msgs_total": prometheus.NewDesc(namespace+"_"+"client_tx_discarded_msgs_total", "Number of discarded transmitted messages.", variableLabelsVpnClient, nil),
	"client_slow_subscriber":         prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClient, nil),

	"queue_spool_quota_bytes": prometheus.NewDesc(namespace+"_"+"queue_spool_quota_bytes", "Queue spool configured max disk usage in bytes.", variableLabelsVpnQueue, nil),
	"queue_spool_usage_bytes": prometheus.NewDesc(namespace+"_"+"queue_spool_usage_bytes", "Queue spool usage in bytes.", variableLabelsVpnQueue, nil),
	"queue_spool_usage_msgs":  prometheus.NewDesc(namespace+"_"+"queue_spool_usage_msgs", "Queue spooled number of messages.", variableLabelsVpnQueue, nil),
	"queue_binds":             prometheus.NewDesc(namespace+"_"+"queue_binds", "Number of clients bound to queue.", variableLabelsVpnQueue, nil),
	"queue_rx_msg_rate":       prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate", "Rate of received messages.", variableLabelsVpnQueue, nil),
	"queue_tx_msg_rate":       prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpnQueue, nil),
	"queue_rx_byte_rate":      prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate", "Rate of received bytes.", variableLabelsVpnQueue, nil),
	"queue_tx_byte_rate":      prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpnQueue, nil),
	"queue_rx_msg_rate_avg":   prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpnQueue, nil),
	"queue_tx_msg_rate_avg":   prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpnQueue, nil),
	"queue_rx_byte_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpnQueue, nil),
	"queue_tx_byte_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpnQueue, nil),
}

// Redirect callback, re-insert basic auth string into header
func (e *Exporter) redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	req.SetBasicAuth(e.config.username, e.config.password)
	return nil
}

// Call http post for the supplied uri and body
func (e *Exporter) postHTTP(uri string, contentType string, body string) (io.ReadCloser, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !e.config.sslVerify}}
	client := http.Client{
		Timeout:       e.config.timeout,
		Transport:     tr,
		CheckRedirect: e.redirectPolicyFunc,
	}

	req, err := http.NewRequest("GET", uri, strings.NewReader(body))
	req.SetBasicAuth(e.config.username, e.config.password)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}

// Get version of broker
func (e *Exporter) getVersionSemp1(ch chan<- prometheus.Metric) (ok float64) {

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
		level.Error(e.logger).Log("msg", "Can't scrape getVersionSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml getVersionSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
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

	ch <- prometheus.MustNewConstMetric(metricsStd["system_version_currentload"], prometheus.GaugeValue, vmrVersionNr)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_version_uptime_totalsecs"], prometheus.GaugeValue, target.RPC.Show.Version.Uptime.TotalSecs)
	ch <- prometheus.MustNewConstMetric(metricsStd["exporter_version_current"], prometheus.GaugeValue, solaceExporterVersion)

	return 1
}

// Get system health information
func (e *Exporter) getHealthSemp1(ch chan<- prometheus.Metric) (ok float64) {

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
		level.Error(e.logger).Log("msg", "Can't scrape HealthSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml HealthSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyCurrentValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyCurrentValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMinimumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMaximumValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyAverageValue/1e6)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyCurrentValue/1e6)

	return 1
}

// Get system-wide spool information
func (e *Exporter) getSpoolSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						QuotaDiskUsage  float64 `xml:"max-disk-usage"`
						QuotaMsgCount   string  `xml:"max-message-count"`
						PersistUsage    float64 `xml:"current-persist-usage"`
						PersistMsgCount float64 `xml:"total-messages-currently-spooled"`
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
		level.Error(e.logger).Log("msg", "Can't scrape Solace", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_quota_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.QuotaDiskUsage*1048576.0))
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	f1, err3 := strconv.ParseFloat(s1, 64)
	if err3 == nil {
		ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_quota_msgs"], prometheus.GaugeValue, f1*1000000)
	}
	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.PersistUsage*1048576.0))
	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_usage_msgs"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistMsgCount)

	return 1
}

// Get system-wide basic redundancy information for HA triples
func (e *Exporter) getRedundancySemp1(ch chan<- prometheus.Metric) (ok float64) {
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
		level.Error(e.logger).Log("msg", "Can't scrape RedundancySemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml RedundancySemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	mateRouterName := "" + target.RPC.Show.Red.MateRouterName
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_config"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ConfigStatus, []string{"Disabled", "Enabled", "Shutdown"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_up"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyStatus, []string{"Down", "Up"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_role"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ActiveStandbyRole, []string{"Backup", "Primary", "Monitor", "Undefined"}), mateRouterName)

	if target.RPC.Show.Red.ActiveStandbyRole == "Primary" && target.RPC.Show.Red.VirtualRouters.Primary.Status.Activity == "Local Active" ||
		target.RPC.Show.Red.ActiveStandbyRole == "Backup" && target.RPC.Show.Red.VirtualRouters.Backup.Status.Activity == "Local Active" {
		f = 1
	} else {
		f = 0
	}
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_local_active"], prometheus.GaugeValue, f, mateRouterName)

	return 1
}

// Get status of bridges for all vpns
func (e *Exporter) getBridgeSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><bridge><bridge-name-pattern>*</bridge-name-pattern></bridge></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape BridgeSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml BridgeSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_num_total_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_max_num_total_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_num_local_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumLocalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_max_num_local_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumLocalBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_num_remote_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumRemoteBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_max_num_remote_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumRemoteBridgesValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_num_total_remote_bridge_subscriptions"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalRemoteBridgeSubscriptions)
	ch <- prometheus.MustNewConstMetric(metricsStd["bridges_max_num_total_remote_bridge_subscriptions"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalRemoteBridgeSubscriptions)
	opStates := []string{"Init", "Shutdown", "NoShutdown", "Prepare", "Prepare-WaitToConnect", "Prepare-FetchingDNS", "NotReady", "NotReady-Connecting", "NotReady-Handshaking", "NotReady-WaitNext", "NotReady-WaitReuse", "NotRead-WaitBridgeVersionMismatch", "NotReady-WaitCleanup", "Ready", "Ready-Subscribing", "Ready-InSync", "NotApplicable", "Invalid"}
	failReasons := []string{"Bridge disabled", "No remote message-vpns configured", "SMF service is disabled", "Msg Backbone is disabled", "Local message-vpn is disabled", "Active-Standby Role Mismatch", "Invalid Active-Standby Role", "Redundancy Disabled", "Not active", "Replication standby", "Remote message-vpns disabled", "Enforce-trusted-common-name but empty trust-common-name list", "SSL transport used but cipher-suite list is empty", "Authentication Scheme is Client-Certificate but no certificate is configured", "Client-Certificate Authentication Scheme used but not all Remote Message VPNs use SSL", "Basic Authentication Scheme used but Basic Client Username not configured", "Cluster Down", "Cluster Link Down", ""}
	for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
		bridgeName := bridge.BridgeName
		vpnName := bridge.LocalVpnName
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_admin_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.AdminState, []string{"Enabled", "Disabled"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_connection_establisher"], prometheus.GaugeValue, encodeMetricMulti(bridge.ConnectionEstablisher, []string{"NotApplicable", "Local", "Remote"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_inbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalState, opStates), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_inbound_operational_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalFailureReason, failReasons), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_outbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.OutboundOperationalState, opStates), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_queue_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.QueueOperationalState, []string{"NotApplicable", "Bound", "Unbound"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_redundancy"], prometheus.GaugeValue, encodeMetricMulti(bridge.Redundancy, []string{"NotApplicable", "auto", "primary", "backup", "static", "none"}), vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_connection_uptime_in_seconds"], prometheus.GaugeValue, bridge.ConnectionUptimeInSeconds, vpnName, bridgeName)
	}
	return 1
}

// Get statistics of bridges for all vpns
func (e *Exporter) getBridgeStatsSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><bridge><bridge-name-pattern>*</bridge-name-pattern><stats/></bridge></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape BridgeSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml BridgeSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}
	for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
		bridgeName := bridge.BridgeName
		vpnName := bridge.LocalVpnName
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_num_subscriptions"], prometheus.GaugeValue, bridge.Client.NumSubscriptions, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(bridge.Client.SlowSubscriber), vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_client_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_client_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_data_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_data_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_persistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_persistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_nonpersistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_nonpersistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_direct_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_direct_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesSent, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_client_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_client_bytess_sent"], prometheus.GaugeValue, bridge.Client.Stats.TotalClientBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_data_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_data_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_persistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_persistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_nonpersistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_nonpersistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_direct_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_direct_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesSent, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_client_large_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.LargeMessagesReceived, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_denied_duplicate_clients"], prometheus.GaugeValue, bridge.Client.Stats.DeniedDuplicateClients, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_not_enough_space_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotEnoughSpaceMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_max_exceeded_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.MaxExceededMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_subscribe_client_not_found"], prometheus.GaugeValue, bridge.Client.Stats.SubscribeClientNotFound, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_not_found_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotFoundMsgsSent, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_current_ingress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentIngressRatePerSecond, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_current_egress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentEgressRatePerSecond, vpnName, bridgeName)

		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_ingress_discards"], prometheus.GaugeValue, bridge.Client.Stats.IngressDiscards.TotalIngressDiscards, vpnName, bridgeName)
		ch <- prometheus.MustNewConstMetric(metricsStd["bridge_total_egress_discards"], prometheus.GaugeValue, bridge.Client.Stats.EgressDiscards.TotalEgressDiscards, vpnName, bridgeName)
	}
	return 1
}

// Get status and number of connected clients of all vpn's, and some data stats and rates
func (e *Exporter) getVpnStatsSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><message-vpn><vpn-name>*</vpn-name><stats/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_connections"], prometheus.GaugeValue, vpn.Connections, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)

		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataTxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataRxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataTxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.IngressDiscards.DiscardedRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.EgressDiscards.DiscardedTxMsgCount, vpn.Name)
	}

	return 1
}

// Config Sync Status for Broker and Vpn
func (e *Exporter) getConfigSyncSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><config-sync><database/></config-sync></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up"}), table.Name)
	}

	return 1
}

// Config Sync Status for Broker and Vpn
func (e *Exporter) getConfigSyncRouterSemp1(ch chan<- prometheus.Metric) (ok float64) {

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
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up"}), table.Name)
	}

	return 1
}

// Config Sync Status for Broker and Vpn
func (e *Exporter) getConfigSyncVpnSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><config-sync><database/><message-vpn/><vpn-name>*</vpn-name></config-sync></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml ConfigSyncSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up"}), table.Name)
	}

	return 1
}

// Replication Config and status
func (e *Exporter) getVpnReplicationSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	command := "<rpc><show><message-vpn><vpn-name>*</vpn-name><replication/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Replication.MessageVpns.MessageVpn {
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_replication_admin_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.AdminState, []string{"shutdown", "enabled", "n/a"}), vpn.VpnName)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_replication_config_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.ConfigState, []string{"standby", "active", "n/a"}), vpn.VpnName)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_replication_transaction_replication_mode"], prometheus.GaugeValue, encodeMetricMulti(vpn.TransactionReplicationMode, []string{"async", "sync"}), vpn.VpnName)
	}

	return 1
}

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Exporter) getClientStatsSemp1(ch chan<- prometheus.Metric) (ok float64) {

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
					} `xml:"primary-virtual-router"`
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

	for nextRequest := "<rpc><show><client><name>*</name><stats/><count/><num-elements>100</num-elements></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape ClientSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode ClientSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show client stats")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_msgs_total"], prometheus.CounterValue, client.Stats.DataRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_msgs_total"], prometheus.CounterValue, client.Stats.DataTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_bytes_total"], prometheus.CounterValue, client.Stats.DataRxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_bytes_total"], prometheus.CounterValue, client.Stats.DataTxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.IngressDiscards.DiscardedRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.EgressDiscards.DiscardedTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername)
		}
		body.Close()
	}

	return 1
}

// Get some statistics for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueDetailSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	for nextRequest := "<rpc><show><queue><name>*</name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape QueueSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode QueueSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show queue details")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_quota_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Quota*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_usage_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Usage*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_usage_msgs"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_binds"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1
}

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueRatesSemp1(ch chan<- prometheus.Metric) (ok float64) {

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

	for nextRequest := "<rpc><show><queue><name>*</name><rates/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape QueueRatesSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode QueueRatesSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show queue rates")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxByteRate, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1
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

const (
	scopeBrokerStandard   = "broker"
	scopeBrokerStatistics = "brokerStats"
	scopeBrokerDetails    = "brokerDetails"
	scopeVpnStandard      = "vpn"
	scopeVpnStatistics    = "vpnStats"
	scopeVpnDetails       = "vpnDetails"
)

// Collection of configs
type config struct {
	listenAddr string
	scrapeURI  string
	username   string
	password   string
	sslVerify  bool
	timeout    time.Duration
	details    bool
	redundancy bool
	scope      string
}

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config config
	logger log.Logger
}

func parseConfigBool(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) bool {
	var ok bool = true
	s := parseConfigString(cfg, logger, iniSection, iniKey, envKey, &ok)
	if ok {
		val, err := strconv.ParseBool(s)
		if err == nil {
			return val
		}
		level.Error(logger).Log("msg", "Config param invalid", "iniKey", iniKey, "envKey", envKey)
	}
	*okp = false
	return false
}

func parseConfigDuration(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) time.Duration {
	var ok bool = true
	s := parseConfigString(cfg, logger, iniSection, iniKey, envKey, &ok)
	if ok {
		val, err := time.ParseDuration(s)
		if err == nil {
			return val
		}
		level.Error(logger).Log("msg", "Config param invalid", "iniKey", iniKey, "envKey", envKey)
	}
	*okp = false
	return 0
}

func parseConfigString(cfg *ini.File, logger log.Logger, iniSection string, iniKey string, envKey string, okp *bool) string {
	if cfg != nil {
		s := cfg.Section(iniSection).Key(iniKey).String()
		if len(s) > 0 {
			return s
		}
	}
	s := os.Getenv(envKey)
	if len(s) > 0 {
		return s
	}
	level.Error(logger).Log("msg", "Config param missing", "iniKey", iniKey, "envKey", envKey)
	*okp = false
	return ""
}

func parseConfig(configFile string, conf *config, logger log.Logger) (ok bool) {
	var oki bool = true
	var cfg *ini.File = nil
	var err interface{}

	if len(configFile) > 0 {
		opts := ini.LoadOptions{
			AllowBooleanKeys: true,
		}
		cfg, err = ini.LoadSources(opts, configFile)
		if err != nil {
			level.Error(logger).Log("msg", "Can't open config file", "err", err)
			return false
		}
	}

	conf.listenAddr = parseConfigString(cfg, logger, "solace", "listenAddr", "SOLACE_LISTEN_ADDR", &oki)
	conf.scrapeURI = parseConfigString(cfg, logger, "solace", "scrapeUri", "SOLACE_SCRAPE_URI", &oki)
	conf.username = parseConfigString(cfg, logger, "solace", "username", "SOLACE_USERNAME", &oki)
	conf.password = parseConfigString(cfg, logger, "solace", "password", "SOLACE_PASSWORD", &oki)
	conf.timeout = parseConfigDuration(cfg, logger, "solace", "timeout", "SOLACE_TIMEOUT", &oki)
	conf.sslVerify = parseConfigBool(cfg, logger, "solace", "sslVerify", "SOLACE_SSL_VERIFY", &oki)
	conf.redundancy = parseConfigBool(cfg, logger, "solace", "redundancy", "SOLACE_REDUNDANCY", &oki)

	return oki
}

// NewExporter returns an initialized Exporter.
func NewExporter(logger log.Logger, conf config) *Exporter {
	return &Exporter{
		logger: logger,
		config: conf,
	}
}

// Describe describes all the metrics ever exported by the Solace exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	if e.config.details {
		for _, m := range metricsDet {
			ch <- m
		}
	} else {
		for _, m := range metricsStd {
			ch <- m
		}
	}
	ch <- solaceUp
}

// Collect fetches the stats from configured Solace location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	var up float64 = 1

	switch e.config.scope {
	case scopeBrokerStandard:
		if up > 0 {
			up = e.getVersionSemp1(ch)
		}
		if up > 0 {
			up = e.getHealthSemp1(ch)
		}
		if up > 0 {
			up = e.getSpoolSemp1(ch)
		}
		if up > 0 && e.config.redundancy {
			up = e.getRedundancySemp1(ch)
		}
		if up > 0 && e.config.redundancy {
			up = e.getConfigSyncRouterSemp1(ch)
		}
		ch <- prometheus.MustNewConstMetric(solaceUp, prometheus.GaugeValue, up)
	case scopeBrokerStatistics:
		{
		}
	case scopeBrokerDetails:
		{
		}
	case scopeVpnStandard:
		if up > 0 {
			up = e.getVpnReplicationSemp1(ch)
		}
		if up > 0 && e.config.redundancy {
			up = e.getConfigSyncVpnSemp1(ch)
		}
		if up > 0 {
			up = e.getBridgeSemp1(ch)
		}
	case scopeVpnStatistics:
		if up > 0 {
			up = e.getClientStatsSemp1(ch)
		}
		if up > 0 {
			up = e.getVpnStatsSemp1(ch)
		}
		if up > 0 {
			up = e.getBridgeStatsSemp1(ch)
		}
		if up > 0 {
			up = e.getQueueRatesSemp1(ch)
		}
	case scopeVpnDetails:
		if up > 0 {
			up = e.getQueueDetailSemp1(ch)
		}
	default:
		if e.config.details {
			if up > 0 {
				up = e.getClientStatsSemp1(ch)
			}
			if up > 0 {
				up = e.getQueueDetailSemp1(ch)
			}
			if up > 0 {
				up = e.getQueueRatesSemp1(ch)
			}
		} else { // Basic
			if up > 0 {
				up = e.getVersionSemp1(ch)
			}
			if up > 0 && e.config.redundancy {
				up = e.getRedundancySemp1(ch)
			}
			if up > 0 && e.config.redundancy {
				up = e.getConfigSyncSemp1(ch)
			}
			if up > 0 {
				up = e.getSpoolSemp1(ch)
			}
			if up > 0 {
				up = e.getHealthSemp1(ch)
			}
			if up > 0 {
				up = e.getVpnStatsSemp1(ch)
			}
			if up > 0 {
				up = e.getVpnReplicationSemp1(ch)
			}
			if up > 0 {
				up = e.getBridgeSemp1(ch)
			}
			if up > 0 {
				up = e.getBridgeStatsSemp1(ch)
			}
			ch <- prometheus.MustNewConstMetric(solaceUp, prometheus.GaugeValue, up)
		}
	}
}

func main() {

	kingpin.HelpFlag.Short('h')

	promlogConfig := promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}
	promlogConfig.Level.Set("info")
	promlogConfig.Format.Set("logfmt")
	flag.AddFlags(kingpin.CommandLine, &promlogConfig)

	configFile := kingpin.Flag(
		"config-file",
		"Path and name of ini file with configuration settings. See sample file solace_exporter.ini.",
	).String()

	kingpin.Parse()

	logger := promlog.New(&promlogConfig)

	var conf config
	if !parseConfig(*configFile, &conf, logger) {
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "Starting solace_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	level.Info(logger).Log("msg", "Scraping",
		"listenAddr", conf.listenAddr,
		"scrapeURI", conf.scrapeURI,
		"username", conf.username,
		"sslVerify", conf.sslVerify,
		"timeout", conf.timeout,
		"redundancy", conf.redundancy)

	// Exporter for standard endpoint
	conf.details = false
	exporterStd := NewExporter(logger, conf)
	registryStd := prometheus.NewRegistry()
	registryStd.MustRegister(exporterStd)
	registryStd.MustRegister(version.NewCollector("solace_standard"))
	handlerStd := promhttp.HandlerFor(registryStd, promhttp.HandlerOpts{})

	// Exporter for client/queue details endpoint
	conf.details = true
	exporterDet := NewExporter(logger, conf)
	registryDet := prometheus.NewRegistry()
	registryDet.MustRegister(exporterDet)
	registryDet.MustRegister(version.NewCollector("solace_detailed"))
	handlerDet := promhttp.HandlerFor(registryDet, promhttp.HandlerOpts{})

	// Exporter for broker standard endpoint
	conf.details = false
	conf.scope = scopeBrokerStandard
	exporterBrokerStd := NewExporter(logger, conf)
	registryBrokerStd := prometheus.NewRegistry()
	registryBrokerStd.MustRegister(exporterBrokerStd)
	registryBrokerStd.MustRegister(version.NewCollector("solace_broker_standard"))
	handlerBrokerStd := promhttp.HandlerFor(registryBrokerStd, promhttp.HandlerOpts{})

	// Exporter for Vpn standard endpoint
	conf.details = false
	conf.scope = scopeVpnStandard
	exporterVpnStd := NewExporter(logger, conf)
	registryVpnStd := prometheus.NewRegistry()
	registryVpnStd.MustRegister(exporterVpnStd)
	registryVpnStd.MustRegister(version.NewCollector("solace_vpn_standard"))
	handlerVpnStd := promhttp.HandlerFor(registryVpnStd, promhttp.HandlerOpts{})

	// Exporter for broker stats endpoint
	conf.details = false
	conf.scope = scopeBrokerStatistics
	exporterBrokerStats := NewExporter(logger, conf)
	registryBrokerStats := prometheus.NewRegistry()
	registryBrokerStats.MustRegister(exporterBrokerStats)
	registryBrokerStats.MustRegister(version.NewCollector("solace_broker_stats"))
	handlerBrokerStats := promhttp.HandlerFor(registryBrokerStats, promhttp.HandlerOpts{})

	// Exporter for Vpn stats endpoint
	conf.details = false
	conf.scope = scopeVpnStatistics
	exporterVpnStats := NewExporter(logger, conf)
	registryVpnStats := prometheus.NewRegistry()
	registryVpnStats.MustRegister(exporterVpnStats)
	registryVpnStats.MustRegister(version.NewCollector("solace_vpn_stats"))
	handlerVpnStats := promhttp.HandlerFor(registryVpnStats, promhttp.HandlerOpts{})

	// Exporter for broker details endpoint
	conf.details = true
	conf.scope = scopeBrokerDetails
	exporterBrokerDet := NewExporter(logger, conf)
	registryBrokerDet := prometheus.NewRegistry()
	registryBrokerDet.MustRegister(exporterBrokerDet)
	registryBrokerDet.MustRegister(version.NewCollector("solace_broker_details"))
	handlerBrokerDetails := promhttp.HandlerFor(registryBrokerDet, promhttp.HandlerOpts{})

	// Exporter for Vpn details endpoint
	conf.details = true
	conf.scope = scopeVpnDetails
	exporterVpnDet := NewExporter(logger, conf)
	registryVpnDet := prometheus.NewRegistry()
	registryVpnDet.MustRegister(exporterVpnDet)
	registryVpnDet.MustRegister(version.NewCollector("solace_vpn_details"))
	handlerVpnDetails := promhttp.HandlerFor(registryVpnDet, promhttp.HandlerOpts{})

	// Test scrape to check if broker can be accessed. If it fails it prints a warn to the log file.
	// Note that failure is not fatal, as broker might not have started up yet.
	conf.timeout, _ = time.ParseDuration("2s") // Don't delay startup too much

	// Configure the endpoints and start the server
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/solace-std", handlerStd)
	http.Handle("/solace-det", handlerDet)

	http.Handle("/solace-broker-std", handlerBrokerStd)
	http.Handle("/solace-vpn-std", handlerVpnStd)
	http.Handle("/solace-broker-stats", handlerBrokerStats)
	http.Handle("/solace-vpn-stats", handlerVpnStats)
	http.Handle("/solace-broker-det", handlerBrokerDetails)
	http.Handle("/solace-vpn-det", handlerVpnDetails)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Solace Exporter</title></head>
             <body>
             <h1>Solace Exporter</h1>
             <p><a href='` + "/metrics" + `'>Default Metrics</a></p>
             <p><a href='` + "/solace-std" + `'>Solace Standard Metrics (System and VPN)</a></p>
             <p><a href='` + "/solace-det" + `'>Solace Detailed Metrics (Client and Queue)</a></p>
						 <p><a href='` + "/solace-broker-std" + `'>Solace Broker only Standard Metrics (System)</a></p>
						 <p><a href='` + "/solace-broker-stats" + `'>Solace Broker only Statistics (System)</a></p>
						 <p><a href='` + "/solace-broker-det" + `'>Solace Broker only Detailed Metrics (System)</a></p>
             <p><a href='` + "/solace-vpn-std" + `'>Solace Vpn only Standard Metrics (VPN)</a></p>
             <p><a href='` + "/solace-vpn-stats" + `'>Solace Vpn only Statistics (VPN)</a></p>
             <p><a href='` + "/solace-vpn-det" + `'>Solace Vpn only Detailed Metrics (VPN)</a></p>
             </body>
             </html>`))
	})
	if err := http.ListenAndServe(conf.listenAddr, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(2)
	}
}
