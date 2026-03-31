package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Kleine Hilfsfunktion, um Solace "yes"/"no" oder "true"/"false" in 1.0/0.0 umzuwandeln
func boolToFloat(val string) float64 {
	v := strings.ToLower(strings.TrimSpace(val))
	if v == "yes" || v == "true" || v == "1" {
		return 1.0
	}
	return 0.0
}

// GetMqttSessionSemp1 holt Details zu MQTT-Sessions inklusive Flags
func (semp *Semp) GetMqttSessionSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var lastSessionKey = ""
	var page = 1

	for command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><mqtt/><mqtt-session/><client-id-pattern>" + itemFilter + "</client-id-pattern><count/><num-elements>100</num-elements></message-vpn></show></rpc>"; command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "MqttSessionSemp1", page)
		page++

		if err != nil {
			semp.logger.Error("Can't scrape MqttSessionSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}

		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			semp.logger.Error("Can't decode MqttSessionSemp1", "err", err, "broker", semp.brokerURI)
			_ = body.Close()
			return 0, err
		}

		if err := target.ExecuteResult.OK(); err != nil {
			semp.logger.Error("unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
			_ = body.Close()
			return 0, err
		}

		command = target.MoreCookie.RPC

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
		_ = body.Close()
	}

	return 1, nil
}
