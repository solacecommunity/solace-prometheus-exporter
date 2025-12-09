package semp

import (
	"net/http"

	"github.com/go-kit/log"
)

// Semp API to the solace broker, to collect data
type Semp struct {
	logger                  log.Logger
	httpClient              http.Client
	httpRequestVisitor      func(*http.Request)
	brokerURI               string
	logBrokerToSlowWarnings bool
	isHWBroker              bool
}

// NewSemp returns an initialized Semp.
func NewSemp(logger log.Logger, brokerURI string, httpClient http.Client, httpRequestVisitor func(*http.Request), logBrokerToSlowWarnings bool, isHWBroker bool) *Semp {
	return &Semp{
		logger:                  logger,
		brokerURI:               brokerURI,
		httpClient:              httpClient,
		httpRequestVisitor:      httpRequestVisitor,
		logBrokerToSlowWarnings: logBrokerToSlowWarnings,
		isHWBroker:              isHWBroker,
	}
}
