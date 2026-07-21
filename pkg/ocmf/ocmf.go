// Package ocmf detects and validates Open Charge Metering Format (OCMF) records
package ocmf

import (
	"embed"
	"encoding/json"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
)

//go:embed OCMF.json
var schemaFile embed.FS

// Header is the literal string that identifies an OCMF record on the wire.
const Header = "OCMF"

var schema *jsonschema.Schema

func init() {
	data, err := schemaFile.ReadFile("OCMF.json")
	if err != nil {
		panic(errors.Wrap(err, "unable to read embedded OCMF schema"))
	}

	schema, err = jsonschema.NewCompiler().Compile(data)
	if err != nil {
		panic(errors.Wrap(err, "unable to compile embedded OCMF schema"))
	}
}

// LooksLikeOCMF reports whether value appears to be an OCMF record, i.e. it starts
// with the "OCMF|" header used to identify the transfer format.
func LooksLikeOCMF(value string) bool {
	return strings.HasPrefix(value, Header+"|")
}

// Parse splits a raw OCMF record ("OCMF|<payload>|<signature>") into its header,
// payload and signature sections and unmarshals the payload and signature JSON
// objects into their typed Go representations.
func Parse(value string) (*Record, error) {
	sections := strings.SplitN(value, "|", 3)
	if len(sections) != 3 {
		return nil, errors.Errorf("malformed OCMF record: expected 3 '|'-separated sections, got %d", len(sections))
	}

	var payload Payload
	if err := json.Unmarshal([]byte(sections[1]), &payload); err != nil {
		return nil, errors.Wrap(err, "unable to parse OCMF payload section as JSON")
	}

	var signature Signature
	if err := json.Unmarshal([]byte(sections[2]), &signature); err != nil {
		return nil, errors.Wrap(err, "unable to parse OCMF signature section as JSON")
	}

	return &Record{
		Header:    sections[0],
		Payload:   payload,
		Signature: signature,
	}, nil
}

// Validate parses value as an OCMF record and validates it against the OCMF JSON Schema.
func Validate(value string) (*jsonschema.EvaluationResult, error) {
	record, err := Parse(value)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(record)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal parsed OCMF record")
	}

	return schema.ValidateJSON(data), nil
}
