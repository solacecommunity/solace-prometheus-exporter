package semp

// Encodes bool into 0,1 metric
func encodeMetricBool(item bool) float64 {
	if item {
		return 1
	}
	return 0
}
