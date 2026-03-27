package exporter

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/promslog"
	"software.sslmate.com/src/go-pkcs12"
)

func ListenAndServeTLS(conf *Config) {
	promlogConfig := promslog.Config{
		Level:  promslog.NewLevel(),
		Format: promslog.NewFormat(),
	}
	promlogConfig.Level.Set("info")
	promlogConfig.Format.Set("logfmt")

	logger := promslog.New(&promlogConfig)

	var tlsCert tls.Certificate

	if strings.ToUpper(conf.CertType) == CertTypePKCS12 {
		// Read byte data from pkcs12 keystore
		p12Data, err := os.ReadFile(conf.Pkcs12File)
		if err != nil {
			logger.Error("Error reading PKCS12 file", "err", err)
			return
		}

		// Extract cert and key from pkcs12 keystore
		privateKey, leafCert, caCerts, err := pkcs12.DecodeChain(p12Data, conf.Pkcs12Pass)
		if err != nil {
			logger.Error("PKCS12 - Error decoding chain", "err", err)
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
			logger.Error("PEM - Error loading keypair", "err", err)
			return
		}
	}

	cfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		Certificates: []tls.Certificate{tlsCert},
	}
	httpServer := &http.Server{
		Addr:              conf.ListenAddr,
		TLSConfig:         cfg,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		logger.Error("Error starting HTTP server", "err", err)
		os.Exit(2)
	}
}
