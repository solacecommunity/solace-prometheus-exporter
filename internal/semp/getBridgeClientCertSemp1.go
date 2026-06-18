package semp

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"

	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetBridgeClientCertSemp1 Get client certificate validity for all bridges
// SEMPv1 returns an openssl-text style dump (not PEM); the first chain entry is the leaf cert
func (semp *Semp) GetBridgeClientCertSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						Bridge []struct {
							BridgeName                string `xml:"bridge-name"`
							LocalVpnName              string `xml:"local-vpn-name"`
							ConnectedRemoteRouterName string `xml:"connected-remote-router-name"`
							ClientCertificate         struct {
								CertificateChain struct {
									CertificateContent []string `xml:"certificate-content"`
								} `xml:"certificate-chain"`
							} `xml:"client-certificate"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var page = 1
	var lastBridgeName = ""
	for command := fmt.Sprintf("<rpc><show><bridge><bridge-name-pattern>"+itemFilter+"</bridge-name-pattern><vpn-name-pattern>"+vpnFilter+"</vpn-name-pattern><client-certificate/><count/><num-elements>%d</num-elements></bridge></show></rpc>", sempPageSize); command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "BridgeClientCertSemp1", page)
		page++

		if err != nil {
			semp.logger.Error("Can't scrape BridgeClientCertSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer func() { _ = body.Close() }()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			semp.logger.Error("Can't decode Xml BridgeClientCertSemp1", "err", err, "broker", semp.brokerURI)
			_ = body.Close()
			return 0, err
		}
		if err := target.ExecuteResult.OK(); err != nil {
			semp.logger.Error(
				"unexpected result",
				"command", command,
				"result", target.ExecuteResult.Result,
				"reason", target.ExecuteResult.Reason,
				"broker", semp.brokerURI,
			)
			return 0, err
		}

		semp.logger.Debug("Result of BridgeClientCertSemp1", "results", len(target.RPC.Show.Bridge.Bridges.Bridge), "page", page-1)
		command = target.MoreCookie.RPC

		for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
			bridgeName := bridge.BridgeName
			vpnName := bridge.LocalVpnName
			connectedRemoteRouter := bridge.ConnectedRemoteRouterName

			bridgeKey := vpnName + "___" + bridgeName
			if bridgeKey == lastBridgeName {
				continue
			}
			lastBridgeName = bridgeKey

			chain := bridge.ClientCertificate.CertificateChain.CertificateContent
			if len(chain) == 0 {
				// No cert configured: emit configured=0 but no expiry gauge, a 0 timestamp would alert as expired
				ch <- semp.NewMetric(MetricDesc["BridgeClientCert"]["bridge_client_cert_configured"], prometheus.GaugeValue, 0, vpnName, bridgeName, connectedRemoteRouter, "")
				continue
			}

			leaf := chain[0]
			notAfter, errA := parseCertTextTime(leaf, "Not After")
			notBefore, errB := parseCertTextTime(leaf, "Not Before")
			commonName := parseCertTextCN(leaf)

			if errA != nil {
				semp.logger.Error("Can't parse bridge client cert notAfter", "err", errA, "vpn", vpnName, "bridge", bridgeName, "broker", semp.brokerURI)
				ch <- semp.NewMetric(MetricDesc["BridgeClientCert"]["bridge_client_cert_configured"], prometheus.GaugeValue, 0, vpnName, bridgeName, connectedRemoteRouter, commonName)
				continue
			}

			ch <- semp.NewMetric(MetricDesc["BridgeClientCert"]["bridge_client_cert_configured"], prometheus.GaugeValue, 1, vpnName, bridgeName, connectedRemoteRouter, commonName)
			ch <- semp.NewMetric(MetricDesc["BridgeClientCert"]["bridge_client_cert_expiry_timestamp_seconds"], prometheus.GaugeValue, float64(notAfter.Unix()), vpnName, bridgeName, connectedRemoteRouter, commonName)
			if errB == nil {
				ch <- semp.NewMetric(MetricDesc["BridgeClientCert"]["bridge_client_cert_not_before_timestamp_seconds"], prometheus.GaugeValue, float64(notBefore.Unix()), vpnName, bridgeName, connectedRemoteRouter, commonName)
			}
		}
		_ = body.Close()
	}
	return 1, nil
}

// openssl-text validity layout, e.g. "Sep  9 13:17:49 2026 GMT" (day is space-padded)
const certTextValidityLayout = "Jan _2 15:04:05 2006 MST"

// anchored to the Subject block so the issuer's CN (listed first) is not matched
var certTextSubjectCNRegexp = regexp.MustCompile(`(?s)\n\s*Subject:.*?\n\s*CN=(.+?)\s*\n`)

// label is matched without the colon (e.g. "Not After"); openssl space-pads the
// labels for alignment ("Not After :" vs "Not Before:"), so split on ":" instead
// of relying on the exact spacing
func parseCertTextTime(certText string, label string) (time.Time, error) {
	for _, line := range strings.Split(certText, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || strings.Join(strings.Fields(key), " ") != label {
			continue
		}
		value = strings.TrimSpace(value)
		t, err := time.Parse(certTextValidityLayout, value)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse validity %q: %w", value, err)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("validity label %q not found in certificate text", label)
}

func parseCertTextCN(certText string) string {
	m := certTextSubjectCNRegexp.FindStringSubmatch(certText)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
