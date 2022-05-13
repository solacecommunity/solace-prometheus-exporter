package semp

import (
	"github.com/go-kit/kit/log"
	"net/http"
)

// Semp API to the solace broker, to collect data
type Semp struct {
	logger             log.Logger
	httpClient         http.Client
	httpRequestVisitor func(*http.Request)
	brokerURI          string
	exporterVersion    float64
}

// NewSemp returns an initialized Semp.
func NewSemp(logger log.Logger, brokerURI string, httpClient http.Client, httpRequestVisitor func(*http.Request), exporterVersion float64) *Semp {
	return &Semp{
		logger:             logger,
		brokerURI:          brokerURI,
		httpClient:         httpClient,
		httpRequestVisitor: httpRequestVisitor,
		exporterVersion:    exporterVersion,
	}
}
