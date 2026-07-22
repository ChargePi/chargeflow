package ocmf

// Record is a fully parsed OCMF record: "OCMF|<Payload>|<Signature>", split on
// its header and the two JSON sections.
type Record struct {
	Header    string    `json:"header"`
	Payload   Payload   `json:"payload"`
	Signature Signature `json:"signature"`
}

// Payload is the signed payload data.
type Payload struct {
	FV string    `json:"FV,omitempty"`
	GI string    `json:"GI,omitempty"`
	GS string    `json:"GS,omitempty"`
	GV string    `json:"GV,omitempty"`
	PG string    `json:"PG,omitempty"`
	MV string    `json:"MV,omitempty"`
	MM string    `json:"MM,omitempty"`
	MS string    `json:"MS,omitempty"`
	MF string    `json:"MF,omitempty"`
	IS *bool     `json:"IS,omitempty"`
	IL string    `json:"IL,omitempty"`
	IF []string  `json:"IF,omitempty"`
	IT string    `json:"IT,omitempty"`
	ID string    `json:"ID,omitempty"`
	CT string    `json:"CT,omitempty"`
	CI string    `json:"CI,omitempty"`
	RD []Reading `json:"RD,omitempty"`
}

type Reading struct {
	TM string   `json:"TM,omitempty"`
	TX string   `json:"TX,omitempty"`
	RV *float64 `json:"RV,omitempty"`
	RI string   `json:"RI,omitempty"`
	RU string   `json:"RU,omitempty"`
	RT string   `json:"RT,omitempty"`
	EF string   `json:"EF,omitempty"`
	ST string   `json:"ST,omitempty"`
}

// Signature is the signature over the payload data.
type Signature struct {
	SA string `json:"SA,omitempty"`
	SE string `json:"SE,omitempty"`
	SM string `json:"SM,omitempty"`
	SD string `json:"SD,omitempty"`
}
