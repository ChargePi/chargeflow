package ocmf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validRecord is the worked example from the OCMF v1.0.1-DRAFT spec (section 4.5).
const validRecord = `OCMF|{` +
	`"FV":"1.0","GI":"ABL SBC-301","GS":"808829900001","GV":"1.4p3","PG":"T12345",` +
	`"MV":"Phoenix Contact","MM":"EEM-350-D-MCB","MS":"BQ27400330016","MF":"1.0",` +
	`"IS":true,"IL":"VERIFIED","IF":["RFID_PLAIN","OCPP_RS_TLS"],"IT":"ISO14443","ID":"1F2D3A4F5506C7",` +
	`"RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","TX":"B","RV":2935.6,"RI":"1-b:1.8.0","RU":"kWh","RT":"AC","EF":"","ST":"G"}]` +
	`}|{"SD":"887FABF407AC82782EEFFF2220C2F856AEB0BC22364BBCC6B55761911ED651D1A922BADA88818C9671AFEE7094D7F536"}`

func TestLooksLikeOCMF(t *testing.T) {
	assert.True(t, LooksLikeOCMF(validRecord))
	assert.False(t, LooksLikeOCMF("1234.5"))
	assert.False(t, LooksLikeOCMF(""))
	assert.False(t, LooksLikeOCMF("OCMFwithoutpipe"))
}

func TestParse(t *testing.T) {
	record, err := Parse(validRecord)
	require.NoError(t, err)
	assert.Equal(t, "OCMF", record.Header)
	assert.Equal(t, "1.0", record.Payload.FV)
	assert.Equal(t, "T12345", record.Payload.PG)
	require.Len(t, record.Payload.RD, 1)
	assert.Equal(t, "G", record.Payload.RD[0].ST)
	assert.Equal(t, "887FABF407AC82782EEFFF2220C2F856AEB0BC22364BBCC6B55761911ED651D1A922BADA88818C9671AFEE7094D7F536", record.Signature.SD)

	_, err = Parse("OCMF|{}")
	assert.Error(t, err, "expected error for record with less than 3 sections")

	_, err = Parse("OCMF|not-json|{}")
	assert.Error(t, err, "expected error for non-JSON payload section")

	_, err = Parse("OCMF|{}|not-json")
	assert.Error(t, err, "expected error for non-JSON signature section")
}

func TestValidate_ValidRecord(t *testing.T) {
	result, err := Validate(validRecord)
	require.NoError(t, err)
	assert.True(t, result.IsValid(), "expected spec example to be valid, got errors: %v", result.Errors)
}

func TestValidate_InvalidRecord(t *testing.T) {
	tests := []struct {
		name   string
		record string
	}{
		{
			name:   "RI without paired RU",
			record: `OCMF|{"FV":"1.0","PG":"T1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","ST":"G","RI":"1-b:1.8.0"}]}|{"SD":"AA"}`,
		},
		{
			name:   "missing required ST in reading",
			record: `OCMF|{"FV":"1.0","PG":"T1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S"}]}|{"SD":"AA"}`,
		},
		{
			name:   "malformed pagination",
			record: `OCMF|{"FV":"1.0","PG":"X1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","ST":"G"}]}|{"SD":"AA"}`,
		},
		{
			name:   "missing signature data",
			record: `OCMF|{"FV":"1.0","PG":"T1","RD":[{"TM":"2018-07-24T13:22:04,000+0200 S","ST":"G"}]}|{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Validate(tt.record)
			require.NoError(t, err)
			assert.False(t, result.IsValid())
		})
	}
}

func TestValidate_MalformedRecord(t *testing.T) {
	_, err := Validate("not-an-ocmf-record")
	assert.Error(t, err)
}
