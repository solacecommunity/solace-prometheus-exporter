package semp

import (
	"testing"
)

func TestEncodeMetricMulti(t *testing.T) {
	t.Parallel()

	tests := []struct {
		item     string
		refItems []string
		want     float64
	}{
		{"apple", []string{"apple", "banana", "cherry"}, 0},
		{"banana", []string{"apple", "banana", "cherry"}, 1},
		{"cherry", []string{"apple", "banana", "cherry"}, 2},
		{"APPLE", []string{"apple", "banana", "cherry"}, 0},  // Case-insensitive match
		{"Banana", []string{"apple", "banana", "cherry"}, 1}, // Case-insensitive match
		{"grape", []string{"apple", "banana", "cherry"}, -1}, // Item not in list
		{"", []string{"apple", "banana", "cherry"}, -1},      // Empty string
	}

	for _, tt := range tests {
		result := encodeMetricMulti(tt.item, tt.refItems)
		if result != tt.want {
			t.Errorf("encodeMetricMulti(%q, %v) = %v; want %v", tt.item, tt.refItems, result, tt.want)
		}
	}
}
