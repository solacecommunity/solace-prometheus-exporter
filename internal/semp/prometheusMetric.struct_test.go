package semp

import (
	"errors"
	"testing"
)

func TestValidateLabelValues(t *testing.T) {
	tests := []struct {
		name                   string
		vals                   []string
		expectedNumberOfValues int
		wantErr                bool
		expectedErr            error
	}{
		{
			name:                   "Correct cardinality",
			vals:                   []string{"val1", "val2"},
			expectedNumberOfValues: 2,
			wantErr:                false,
		},
		{
			name:                   "Inconsistent cardinality - too few",
			vals:                   []string{"val1"},
			expectedNumberOfValues: 2,
			wantErr:                true,
			expectedErr:            errInconsistentCardinality,
		},
		{
			name:                   "Inconsistent cardinality - too many",
			vals:                   []string{"val1", "val2", "val3"},
			expectedNumberOfValues: 2,
			wantErr:                true,
			expectedErr:            errInconsistentCardinality,
		},
		{
			name:                   "Empty values and expected zero",
			vals:                   []string{},
			expectedNumberOfValues: 0,
			wantErr:                false,
		},
		{
			name:                   "Nil values and expected zero",
			vals:                   nil,
			expectedNumberOfValues: 0,
			wantErr:                false,
		},
		{
			name:                   "Invalid UTF-8",
			vals:                   []string{"val1", "\xff"},
			expectedNumberOfValues: 2,
			wantErr:                true,
		},
		{
			name:                   "All valid UTF-8",
			vals:                   []string{"valid", "also valid", "你好"},
			expectedNumberOfValues: 3,
			wantErr:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLabelValues(tt.vals, tt.expectedNumberOfValues)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLabelValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("validateLabelValues() error = %v, expectedErr %v", err, tt.expectedErr)
				}
			}
		})
	}
}
