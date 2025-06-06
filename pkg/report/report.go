package report

type Report struct {
	ParserErrors    []string `json:"parser_errors,omitempty"`
	ValidatorErrors []string `json:"validator_errors,omitempty"`
}
