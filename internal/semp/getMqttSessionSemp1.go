package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

func (semp *Semp) GetMqttSessionSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Mqtt struct {
						MqttSessions struct {
							MqttSession []struct {
								ClientId         string  `xml:"client-id"`
								Owner            string  `xml:"owner"`
								MsgVpnName       string  `xml:"message-vpn"`
								NumSubscriptions float64 `xml:"num-subscriptions"`
								Enabled          string  `xml:"enabled"`
								Clean            string  `xml:"clean"`
								Durable          string  `xml:"durable"`
								Uptime           struct {
									Days  float64 `xml:"days"`
									Hours float64 `xml:"hours"`
									Mins  float64 `xml:"mins"`
									Secs  float64 `xml:"secs"`
								} `xml:"uptime"`
							} `xml:"mqtt-session"`
						} `xml:"mqtt-sessions"`
					} `xml:"mqtt"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastSessionKey = ""

	command := fmt.Sprintf("<rpc><show><message-vpn><vpn-name>"+vpnFilter+"</vpn-name><mqtt/><mqtt-session/><client-id-pattern>"+itemFilter+"</client-id-pattern><count/><num-elements>%d</num-elements></message-vpn></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "MqttSessionSemp1", func(target *Data) {
		for _, session := range target.RPC.Show.MessageVpn.Mqtt.MqttSessions.MqttSession {
			sessionKey := session.MsgVpnName + "___" + session.ClientId
			if sessionKey == lastSessionKey {
				continue
			}
			lastSessionKey = sessionKey

			uptimeSeconds := (session.Uptime.Days * 86400) + (session.Uptime.Hours * 3600) + (session.Uptime.Mins * 60) + session.Uptime.Secs

			ch <- semp.NewMetric(MetricDesc["MqttSession"]["mqtt_session_info"], prometheus.GaugeValue, 1.0, session.MsgVpnName, session.ClientId, session.Owner, session.Clean, session.Durable, session.Enabled)
			ch <- semp.NewMetric(MetricDesc["MqttSession"]["mqtt_session_subscriptions"], prometheus.GaugeValue, session.NumSubscriptions, session.MsgVpnName, session.ClientId, session.Owner)
			ch <- semp.NewMetric(MetricDesc["MqttSession"]["mqtt_session_uptime_seconds"], prometheus.GaugeValue, uptimeSeconds, session.MsgVpnName, session.ClientId, session.Owner)
		}
	})
}
