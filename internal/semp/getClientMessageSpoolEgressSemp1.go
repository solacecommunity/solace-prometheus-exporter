package semp

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"solace_exporter/internal/semp/types"
)

// GetClientMessageSpoolEgressSemp1 emits one client_endpoint_egress_bind_time_seconds
// gauge per (client, endpoint) binding. A single client can be bound to multiple
// endpoints, so each <flow> under <flows-to-client> becomes one series.
//
// itemFilter is the broker-side client-name wildcard (passed verbatim into
// <name>); the broker handles the wildcard. There is no VPN or endpoint-name
// filter at the SEMP level for `show client ... message-spool egress`.
func (semp *Semp) GetClientMessageSpoolEgressSemp1(ch chan<- PrometheusMetric, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName             string `xml:"name"`
							ClientAddress          string `xml:"client-address"`
							ClientID               uint32 `xml:"client-id"`
							MsgVpnName             string `xml:"message-vpn"`
							ClientUsername         string `xml:"client-username"`
							OriginalClientUsername string `xml:"original-client-username"`
							User                   string `xml:"user"`
							Description            string `xml:"description"`
							SoftwareVersion        string `xml:"software-version"`
							Platform               string `xml:"platform"`
							MessageSpool           struct {
								FlowsToClient struct {
									Flow []struct {
										Name            string `xml:"name"`
										Type            string `xml:"type"`
										BindTimeSeconds int64  `xml:"bind-time-seconds"`
									} `xml:"flow"`
								} `xml:"flows-to-client"`
							} `xml:"message-spool"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	// The broker does not support paging (<count/><num-elements>) for
	// `show client ... message-spool egress connected`, so this is a single request.
	command := "<rpc><show><client><name>" + itemFilter + "</name><message-spool/><egress/><connected/></client></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "ClientMessageSpoolEgressSemp1", 1, &target); err != nil {
		return up, err
	}
	for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
		clientIDStr := strconv.FormatUint(uint64(client.ClientID), 10)
		clientIP := strings.Split(client.ClientAddress, ":")[0]
		for _, flow := range client.MessageSpool.FlowsToClient.Flow {
			bindTarget := flow.Type + "=" + flow.Name
			ch <- semp.NewMetric(
				MetricDesc["ClientMessageSpoolEgress"]["client_endpoint_egress_bind_time_seconds"],
				prometheus.GaugeValue,
				float64(flow.BindTimeSeconds),
				client.MsgVpnName, client.ClientName, clientIP,
				clientIDStr, client.ClientUsername, client.OriginalClientUsername,
				client.User, client.Description,
				client.SoftwareVersion, client.Platform,
				flow.Type, flow.Name, bindTarget,
			)
		}
	}
	return 1, nil
}
