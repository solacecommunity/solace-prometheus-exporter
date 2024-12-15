package exporter

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"

	"github.com/go-kit/log/level"
	"github.com/prometheus/common/promlog"
	"software.sslmate.com/src/go-pkcs12"
)

func ListenAndServeTLS(conf Config) {

	promlogConfig := promlog.Config{
		Level:  &promlog.AllowedLevel{},
		Format: &promlog.AllowedFormat{},
	}
	promlogConfig.Level.Set("info")
	promlogConfig.Format.Set("logfmt")

	logger := promlog.New(&promlogConfig)

	var tlsCert tls.Certificate

	if strings.ToUpper(conf.CertType) == CERTTYPE_PKCS12 {

		// Read byte data from pkcs12 keystore
		p12Data, err := os.ReadFile(conf.Pkcs12File)
		if err != nil {
			level.Error(logger).Log("Error reading PKCS12 file", err)
			return
		}

		// Extract cert and key from pkcs12 keystore
		privateKey, leafCert, caCerts, err := pkcs12.DecodeChain(p12Data, conf.Pkcs12Pass)
		if err != nil {
			level.Error(logger).Log("PKCS12 - Error decoding chain", err)
			return
		}

		certBytes := [][]byte{leafCert.Raw}
		for _, ca := range caCerts {
			certBytes = append(certBytes, ca.Raw)
		}
		tlsCert = tls.Certificate{
			Certificate: certBytes,
			PrivateKey:  privateKey,
		}
	} else {
		var err error
		tlsCert, err = tls.LoadX509KeyPair(conf.Certificate, conf.PrivateKey)
		if err != nil {
			level.Error(logger).Log("PEM - Error loading keypair", err)
			return
		}
	}

	cfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		Certificates: []tls.Certificate{tlsCert},
	}
	httpServer := &http.Server{
		Addr:         conf.ListenAddr,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(2)
	}

}
