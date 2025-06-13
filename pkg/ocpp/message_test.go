package ocpp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFormatErrorType(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected ErrorCode
	}{
		{
			name:     "OCPP 1.6",
			version:  V16,
			expected: FormatViolationV16,
		},
		{
			name:     "OCPP 2.0",
			version:  V20,
			expected: FormatViolationV2,
		},
		{
			name:    "Invalid Version",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Invalid Version" {
				assert.Panics(t, func() {
					_ = FormatErrorType(tt.version)
				})
			} else {
				result := FormatErrorType(tt.version)
				assert.Equal(t, result, tt.expected)
			}
		})
	}
}

func TestOccurrenceConstraintErrorType(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected ErrorCode
	}{
		{
			name:     "OCPP 1.6",
			version:  V16,
			expected: OccurrenceConstraintViolationV16,
		},
		{
			name:     "OCPP 2.0",
			version:  V20,
			expected: OccurrenceConstraintViolationV2,
		},
		{
			name: "Invalid Version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Invalid Version" {
				assert.Panics(t, func() {
					_ = OccurrenceConstraintErrorType(tt.version)
				})
			} else {
				result := OccurrenceConstraintErrorType(tt.version)
				assert.Equal(t, result, tt.expected)
			}
		})
	}
}

func TestIsErrorCodeValid(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected bool
	}{
		{
			name:     "Not implemented",
			code:     NotImplemented,
			expected: true,
		},
		{
			name:     "Not supported",
			code:     NotSupported,
			expected: true,
		},
		{
			name:     "Internal error",
			code:     InternalError,
			expected: true,
		},
		{
			name:     "Format violation",
			code:     FormatViolationV16,
			expected: true,
		}, {
			name:     "Format violation",
			code:     FormatViolationV2,
			expected: true,
		},
		{
			name:     "Security error",
			code:     SecurityError,
			expected: true,
		},
		{
			name:     "Property constraint violation",
			code:     PropertyConstraintViolation,
			expected: true,
		},
		{
			name:     "Occurrence constraint violation",
			code:     OccurrenceConstraintViolationV16,
			expected: true,
		},
		{
			name:     "Occurrence constraint violation",
			code:     OccurrenceConstraintViolationV2,
			expected: true,
		},
		{
			name:     "Type constraint violation",
			code:     TypeConstraintViolation,
			expected: true,
		},
		{
			name:     "Generic error",
			code:     GenericError,
			expected: true,
		},
		{
			name:     "Unknown error",
			code:     "UnknownError",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsErrorCodeValid(tt.code)
			assert.Equal(t, result, tt.expected)
		})
	}
}
