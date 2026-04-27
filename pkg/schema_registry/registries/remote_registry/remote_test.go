package remote_registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

const bootNotificationSchema = `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:BootNotificationRequest",
		"title": "BootNotificationRequest",
		"type": "object",
		"properties": {
			"chargePointVendor": {
				"type": "string",
				"maxLength": 20
			},
			"chargePointModel": {
				"type": "string",
				"maxLength": 20
			}
		},
		"additionalProperties": false,
		"required": ["chargePointVendor", "chargePointModel"]
	}`

const authorizationReqSchema = `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:AuthorizeRequest",
		"title": "AuthorizeRequest",
		"type": "object",
		"properties": {
			"idTag": {
				"type": "string",
				"maxLength": 20
			}
		},
		"additionalProperties": false,
		"required": ["idTag"]
	}`

const statusNotificationSchema = `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:StatusNotificationRequest",
		"title": "StatusNotificationRequest",
		"type": "object",
		"properties": {
			"connectorId": {
				"type": "integer"
			},
			"status": {
				"type": "string",
				"enum": ["Available", "Preparing", "Charging", "SuspendedEVSE", "SuspendedEV", "Finishing", "Reserved", "Unavailable", "Faulted"]
			}
		},
		"additionalProperties": false,
		"required": ["connectorId", "status"]
	}`

type remoteRegistryIntegrationTestSuite struct {
	suite.Suite
	redpandaContainer testcontainers.Container
	registryURL       string
	logger            *zap.Logger
}

func (s *remoteRegistryIntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	// s.logger = zaptest.NewLogger(s.T())
	s.logger, _ = zap.NewDevelopment()

	// Start Redpanda container with Schema Registry enabled
	redpandaContainer, err := redpanda.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v23.1.7")
	s.Require().NoError(err, "Failed to start Redpanda container")

	s.redpandaContainer = redpandaContainer

	// Get the schema registry URL
	schemaRegistryURL, err := redpandaContainer.SchemaRegistryAddress(ctx)
	s.Require().NoError(err, "Failed to get schema registry address")

	s.registryURL = schemaRegistryURL
}

func (s *remoteRegistryIntegrationTestSuite) TearDownSuite() {
	if s.redpandaContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := s.redpandaContainer.Terminate(ctx)
		s.NoError(err, "Failed to terminate Redpanda container")
	}
}

func (s *remoteRegistryIntegrationTestSuite) TestRegisterSchema() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	validSchema := json.RawMessage(bootNotificationSchema)

	tests := []struct {
		name        string
		ocppVersion ocpp.Version
		action      string
		schema      json.RawMessage
		expectError bool
	}{
		{
			name:        "Register valid schema for OCPP 1.6",
			ocppVersion: ocpp.V16,
			action:      "BootNotificationRequest",
			schema:      validSchema,
			expectError: false,
		},
		{
			name:        "Register valid schema for OCPP 2.0",
			ocppVersion: ocpp.V20,
			action:      "BootNotificationRequest",
			schema:      validSchema,
			expectError: false,
		},
		{
			name:        "Register schema with Response suffix",
			ocppVersion: ocpp.V16,
			action:      "BootNotificationResponse",
			schema:      validSchema,
			expectError: false,
		},
		{
			name:        "Invalid OCPP version",
			ocppVersion: ocpp.Version("unsupported"),
			action:      "BootNotificationRequest",
			schema:      validSchema,
			expectError: true,
		},
		{
			name:        "Invalid action suffix",
			ocppVersion: ocpp.V16,
			action:      "BootNotification",
			schema:      validSchema,
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := registry.RegisterSchema(ctx, tt.ocppVersion, tt.action, tt.schema)
			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	validSchema := json.RawMessage(authorizationReqSchema)

	// First register a schema
	err = registry.RegisterSchema(ctx, ocpp.V16, "AuthorizeRequest", validSchema)
	s.Require().NoError(err)

	// Test getting the schema
	schema, found := registry.GetSchema(ctx, ocpp.V16, "AuthorizeRequest")
	s.True(found, "Schema should be found")
	s.NotNil(schema, "Schema should not be nil")

	// Test getting non-existent schema
	_, found = registry.GetSchema(ctx, ocpp.V16, "NonExistentRequest")
	s.False(found, "Non-existent schema should not be found")

	// Test getting schema for non-existent OCPP version
	_, found = registry.GetSchema(ctx, ocpp.V20, "AuthorizeRequest")
	s.False(found, "Schema for different OCPP version should not be found")
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema_Caching() {
	ctx := context.Background()

	// Use a short cache refresh duration for testing
	cacheRefresh := 2 * time.Second
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
		WithCacheRefreshDuration(cacheRefresh),
	)
	s.Require().NoError(err)

	validSchema := json.RawMessage(statusNotificationSchema)

	// Register the schema
	err = registry.RegisterSchema(ctx, ocpp.V16, "StatusNotificationRequest", validSchema)
	s.Require().NoError(err)

	// First fetch - should fetch from remote
	schema1, found := registry.GetSchema(ctx, ocpp.V16, "StatusNotificationRequest")
	s.True(found)
	s.NotNil(schema1)

	// Second fetch immediately - should use cache
	schema2, found := registry.GetSchema(ctx, ocpp.V16, "StatusNotificationRequest")
	s.True(found)
	s.NotNil(schema2)
	s.Equal(schema1, schema2, "Should return the same schema instance from cache")

	// Wait for cache to expire
	time.Sleep(cacheRefresh + 500*time.Millisecond)

	// Third fetch after cache expiry - should fetch from remote again
	schema3, found := registry.GetSchema(ctx, ocpp.V16, "StatusNotificationRequest")
	s.True(found)
	s.NotNil(schema3)
	// Note: schema3 will be a new instance, but should validate the same data
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema_MultipleVersions() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	schemaV1 := json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:HeartbeatRequest",
		"title": "HeartbeatRequest",
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	schemaV2 := json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:HeartbeatRequest",
		"title": "HeartbeatRequest",
		"type": "object",
		"properties": {
			"timestamp": {
				"type": "string",
				"format": "date-time"
			}
		},
		"additionalProperties": false
	}`)

	// Register first version
	err = registry.RegisterSchema(ctx, ocpp.V16, "HeartbeatRequest", schemaV1)
	s.Require().NoError(err)

	// Register second version (should create a new version in the registry)
	err = registry.RegisterSchema(ctx, ocpp.V16, "HeartbeatRequest", schemaV2)
	s.Require().NoError(err)

	// GetSchema should return the latest version
	schema, found := registry.GetSchema(ctx, ocpp.V16, "HeartbeatRequest")
	s.True(found)
	s.NotNil(schema)
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema_InvalidInputs() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	tests := []struct {
		name        string
		ocppVersion ocpp.Version
		action      string
	}{
		{
			name:        "Invalid OCPP version",
			ocppVersion: ocpp.Version("invalid"),
			action:      "BootNotificationRequest",
		},
		{
			name:        "Invalid action suffix",
			ocppVersion: ocpp.V16,
			action:      "InvalidAction",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			schema, found := registry.GetSchema(ctx, tt.ocppVersion, tt.action)
			s.False(found, "Should not find schema for invalid input")
			s.Nil(schema, "Schema should be nil for invalid input")
		})
	}
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema_DifferentOCPPVersions() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	schema16 := json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:StartTransactionRequest",
		"title": "StartTransactionRequest",
		"type": "object",
		"properties": {
			"connectorId": {
				"type": "integer"
			}
		},
		"additionalProperties": false,
		"required": ["connectorId"]
	}`)

	schema20 := json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:2.0:2019:12:StartTransactionRequest",
		"title": "StartTransactionRequest",
		"type": "object",
		"properties": {
			"evseId": {
				"type": "integer"
			}
		},
		"additionalProperties": false,
		"required": ["evseId"]
	}`)

	// Register schemas for different OCPP versions
	err = registry.RegisterSchema(ctx, ocpp.V16, "StartTransactionRequest", schema16)
	s.Require().NoError(err)

	err = registry.RegisterSchema(ctx, ocpp.V20, "StartTransactionRequest", schema20)
	s.Require().NoError(err)

	// Verify both schemas can be retrieved independently
	schema1, found1 := registry.GetSchema(ctx, ocpp.V16, "StartTransactionRequest")
	s.True(found1, "OCPP 1.6 schema should be found")
	s.NotNil(schema1)

	schema2, found2 := registry.GetSchema(ctx, ocpp.V20, "StartTransactionRequest")
	s.True(found2, "OCPP 2.0 schema should be found")
	s.NotNil(schema2)

	// Schemas should be different
	s.NotEqual(schema1, schema2, "Schemas for different OCPP versions should be different")
}

func (s *remoteRegistryIntegrationTestSuite) TestGetSchema_ResponseSuffix() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
	)
	s.Require().NoError(err)

	responseSchema := json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"id": "urn:OCPP:1.6:2019:12:BootNotificationResponse",
		"title": "BootNotificationResponse",
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["Accepted", "Pending", "Rejected"]
			},
			"currentTime": {
				"type": "string",
				"format": "date-time"
			},
			"interval": {
				"type": "integer"
			}
		},
		"additionalProperties": false,
		"required": ["status", "currentTime"]
	}`)

	// Register response schema
	err = registry.RegisterSchema(ctx, ocpp.V16, "BootNotificationResponse", responseSchema)
	s.Require().NoError(err)

	// Retrieve response schema
	schema, found := registry.GetSchema(ctx, ocpp.V16, "BootNotificationResponse")
	s.True(found, "Response schema should be found")
	s.NotNil(schema, "Response schema should not be nil")
}

func (s *remoteRegistryIntegrationTestSuite) TestDeleteSchema() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(s.registryURL, s.logger, WithTimeout(10*time.Second))
	s.Require().NoError(err)

	validSchema := json.RawMessage(bootNotificationSchema)

	tests := []struct {
		name        string
		setup       func()
		ocppVersion ocpp.Version
		action      string
		expectError bool
	}{
		{
			name: "Delete existing schema",
			setup: func() {
				err := registry.RegisterSchema(ctx, ocpp.V16, "MeterValuesRequest", validSchema)
				s.Require().NoError(err)
			},
			ocppVersion: ocpp.V16,
			action:      "MeterValuesRequest",
			expectError: false,
		},
		{
			name:        "Delete non-existent schema",
			ocppVersion: ocpp.V16,
			action:      "NonExistentRequest",
			expectError: true,
		},
		{
			name:        "Invalid OCPP version",
			ocppVersion: ocpp.Version("unsupported"),
			action:      "MeterValuesRequest",
			expectError: true,
		},
		{
			name:        "Invalid action suffix",
			ocppVersion: ocpp.V16,
			action:      "MeterValues",
			expectError: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setup != nil {
				tt.setup()
			}

			err := registry.DeleteSchema(ctx, tt.ocppVersion, tt.action)
			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *remoteRegistryIntegrationTestSuite) TestDeleteSchema_CacheInvalidation() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(
		s.registryURL,
		s.logger,
		WithTimeout(10*time.Second),
		WithCacheRefreshDuration(5*time.Minute),
	)
	s.Require().NoError(err)

	validSchema := json.RawMessage(authorizationReqSchema)

	err = registry.RegisterSchema(ctx, ocpp.V16, "DataTransferRequest", validSchema)
	s.Require().NoError(err)

	schema, found := registry.GetSchema(ctx, ocpp.V16, "DataTransferRequest")
	s.True(found, "schema should be found after registration")
	s.NotNil(schema)

	err = registry.DeleteSchema(ctx, ocpp.V16, "DataTransferRequest")
	s.Require().NoError(err)

	_, found = registry.GetSchema(ctx, ocpp.V16, "DataTransferRequest")
	s.False(found, "schema should not be found after deletion — cache must be invalidated")
}

func (s *remoteRegistryIntegrationTestSuite) TestRegisterAndDeleteLifecycle() {
	ctx := context.Background()
	registry, err := NewRemoteSchemaRegistry(s.registryURL, s.logger, WithTimeout(10*time.Second))
	s.Require().NoError(err)

	validSchema := json.RawMessage(bootNotificationSchema)

	err = registry.RegisterSchema(ctx, ocpp.V16, "ClearCacheRequest", validSchema)
	s.Require().NoError(err)

	schema, found := registry.GetSchema(ctx, ocpp.V16, "ClearCacheRequest")
	s.True(found, "schema should be retrievable after registration")
	s.NotNil(schema)

	err = registry.DeleteSchema(ctx, ocpp.V16, "ClearCacheRequest")
	s.Require().NoError(err)

	_, found = registry.GetSchema(ctx, ocpp.V16, "ClearCacheRequest")
	s.False(found, "schema should not be retrievable after deletion")

	// Re-registration after deletion must succeed
	err = registry.RegisterSchema(ctx, ocpp.V16, "ClearCacheRequest", validSchema)
	s.NoError(err)
}

func TestRemoteRegistryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	suite.Run(t, new(remoteRegistryIntegrationTestSuite))
}

func TestAuthOptions(t *testing.T) {
	tests := []struct {
		name           string
		opts           []Options
		expectedHeader string
		expectedValue  string
	}{
		{
			name: "Basic Auth",
			opts: []Options{
				WithBasicAuth("testuser", "testpass"),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass")),
		},
		{
			name: "Bearer Token",
			opts: []Options{
				WithBearerToken("test-token-123"),
			},
			expectedHeader: "Authorization",
			expectedValue:  "Bearer test-token-123",
		},
		{
			name: "API Key with default header",
			opts: []Options{
				WithAPIKey("test-api-key", ""),
			},
			expectedHeader: "X-API-Key",
			expectedValue:  "test-api-key",
		},
		{
			name: "API Key with custom header",
			opts: []Options{
				WithAPIKey("test-api-key", "X-Custom-API-Key"),
			},
			expectedHeader: "X-Custom-API-Key",
			expectedValue:  "test-api-key",
		},
		{
			name: "Custom Header",
			opts: []Options{
				WithCustomHeader("X-Custom-Auth", "custom-value"),
			},
			expectedHeader: "X-Custom-Auth",
			expectedValue:  "custom-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a config and apply options
			config := remoteRegistryConfig{
				auth: authConfig{authType: authTypeNone},
			}
			for _, opt := range tt.opts {
				opt(&config)
			}

			// Create a test request
			req, err := http.NewRequest("GET", "http://example.com/test", nil)
			assert.NoError(t, err)

			// Verify the header was set correctly
			actualValue := req.Header.Get(tt.expectedHeader)
			assert.Equal(t, tt.expectedValue, actualValue, "Header value should match expected value")
		})
	}
}
