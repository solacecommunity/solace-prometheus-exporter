package semp

import (
	"testing"
)

func TestEncodeMetricBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		item bool
		want float64
	}{
		{true, 1},
		{false, 0},
	}

	for _, tt := range tests {
		result := encodeMetricBool(tt.item)
		if result != tt.want {
			t.Errorf("encodeMetricBool(%v) = %v; want %v", tt.item, result, tt.want)
		}
	}
}
