package ocpp

// OcppContext carries the OCPP protocol version together with optional
// vendor and model identifiers used for vendor/model-specific schema selection.
type OcppContext struct {
	Version Version // Required
	Vendor  string  // Optional. Must be present if model is set.
	Model   string  // Optional.
}
