package remote

type authConfig struct {
	authType          authType
	username          string
	password          string
	bearerToken       string
	apiKey            string
	apiKeyHeader      string
	customHeaderName  string
	customHeaderValue string
}

type authType int

const (
	authTypeNone authType = iota
	authTypeBasic
	authTypeBearer
	authTypeAPIKey
	authTypeCustomHeader
)
