package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetVpnStatsSemp1 Get statistics of all VPNs
func (semp *Semp) GetVpnStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Vpn []struct {
						Name                      string  `xml:"name"`
						LocalStatus               string  `xml:"local-status"`
						Connections               float64 `xml:"connections"`
						QuotaConnections          float64 `xml:"max-connections"`
						QuotaConnectionsSmf       float64 `xml:"max-connections-service-smf"`
						QuotaConnectionsWeb       float64 `xml:"max-connections-service-web"`
						QuotaConnectionsMqtt      float64 `xml:"max-connections-service-mqtt"`
						QuotaConnectionsAmqp      float64 `xml:"max-connections-service-amqp"`
						QuotaConnectionsRestIn    float64 `xml:"max-connections-service-rest-incoming"`
						QuotaConnectionsRestOut   float64 `xml:"max-connections-service-rest-outgoing"`
						ConnectionsAmqService     float64 `xml:"connections-service-amqp"`
						ConnectionsSmfService     float64 `xml:"connections-service-smf"`
						ConnectionsWebService     float64 `xml:"connections-service-web"`
						ConnectionsMqttService    float64 `xml:"connections-service-mqtt"`
						ConnectionsRestInService  float64 `xml:"connections-service-rest-incoming"`
						ConnectionsRestOutService float64 `xml:"connections-service-rest-outgoing"`
						Stats                     struct {
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
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><stats/></message-vpn></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "VpnStatsSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape VpnSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_rx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataRxMsgCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_tx_msgs_total"], prometheus.CounterValue, vpn.Stats.DataTxMsgCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_rx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataRxByteCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_tx_bytes_total"], prometheus.CounterValue, vpn.Stats.DataTxByteCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_rx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.IngressDiscards.DiscardedRxMsgCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_tx_discarded_msgs_total"], prometheus.CounterValue, vpn.Stats.EgressDiscards.DiscardedTxMsgCount, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections"], prometheus.GaugeValue, vpn.Connections, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_amqp"], prometheus.GaugeValue, vpn.ConnectionsAmqService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_mqtt"], prometheus.GaugeValue, vpn.ConnectionsMqttService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_smf"], prometheus.GaugeValue, vpn.ConnectionsSmfService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_web"], prometheus.GaugeValue, vpn.ConnectionsWebService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_rest_in"], prometheus.GaugeValue, vpn.ConnectionsRestInService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_connections_service_rest_out"], prometheus.GaugeValue, vpn.ConnectionsRestOutService, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections"], prometheus.GaugeValue, vpn.QuotaConnections, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_smf"], prometheus.GaugeValue, vpn.QuotaConnectionsSmf, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_web"], prometheus.GaugeValue, vpn.QuotaConnectionsWeb, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_amqp"], prometheus.GaugeValue, vpn.QuotaConnectionsAmqp, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_mqtt"], prometheus.GaugeValue, vpn.QuotaConnectionsMqtt, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_rest_in"], prometheus.GaugeValue, vpn.QuotaConnectionsRestIn, vpn.Name)
		ch <- semp.NewMetric(MetricDesc["VpnStats"]["vpn_quota_connections_rest_out"], prometheus.GaugeValue, vpn.QuotaConnectionsRestOut, vpn.Name)
	}

	return 1, nil
}
