package ocpp

// MeterValuesRequest is the minimal shape of a MeterValues.req payload, shared by
// OCPP 1.6 and 2.0.1/2.1, needed to locate signed meter readings.
type MeterValuesRequest struct {
	MeterValue []MeterValue `json:"meterValue"`
}

// MeterValue is a single entry of MeterValuesRequest.MeterValue.
type MeterValue struct {
	SampledValue []SampledValue `json:"sampledValue"`
}

// SampledValue is a single entry of MeterValue.SampledValue.
//
// OCPP 1.6 declares "value" as a string; OCPP 2.0.1/2.1 declare it as a number. Value is
// typed as any here so one type covers both; callers type-assert to the shape they expect.
type SampledValue struct {
	Value any `json:"value,omitempty"`

	// SignedMeterValue is only present in OCPP 2.0.1/2.1, which carry a signed reading
	// in this dedicated sub-object rather than directly in Value.
	SignedMeterValue *SignedMeterValue `json:"signedMeterValue,omitempty"`
}

// SignedMeterValue mirrors OCPP 2.0.1/2.1's SignedMeterValueType.
type SignedMeterValue struct {
	// SignedMeterData is base64-encoded, in the format named by EncodingMethod.
	SignedMeterData string `json:"signedMeterData,omitempty"`
	// EncodingMethod names the format used to produce SignedMeterData, e.g. "OCMF", "EDL".
	EncodingMethod string `json:"encodingMethod,omitempty"`
}
