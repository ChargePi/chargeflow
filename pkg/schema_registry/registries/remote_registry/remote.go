package remote_registry

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/schema_registry/registries/file_registry"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

type remoteRegistryConfig struct {
	url string
	// Cache settings
	cacheRefresh time.Duration
	cache        Cache
	timeout      time.Duration
	auth         authConfig
}

// SchemaRegistry fetches schemas from a remote schema registry service and caches them locally to reduce latency and network calls.
type SchemaRegistry struct {
	logger *zap.Logger

	config     remoteRegistryConfig
	httpClient *http.Client
	baseURL    string

	cache    Cache
	compiler *jsonschema.Compiler
}

// applyAuthHeaders adds authentication headers to the request based on the auth config.
func (r *SchemaRegistry) applyAuthHeaders(req *http.Request) {
	switch r.config.auth.authType {
	case authTypeBasic:
		credentials := base64.StdEncoding.EncodeToString([]byte(r.config.auth.username + ":" + r.config.auth.password))
		req.Header.Set("Authorization", "Basic "+credentials)
	case authTypeBearer:
		req.Header.Set("Authorization", "Bearer "+r.config.auth.bearerToken)
	case authTypeAPIKey:
		req.Header.Set(r.config.auth.apiKeyHeader, r.config.auth.apiKey)
	case authTypeCustomHeader:
		req.Header.Set(r.config.auth.customHeaderName, r.config.auth.customHeaderValue)
	default:
		// No auth
	}
}

// doRequest performs an HTTP request with authentication and logging.
func (r *SchemaRegistry) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	fullURL, err := url.JoinPath(r.baseURL, path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build URL for path %s", path)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	r.logger.Debug("Executing request",
		zap.String("method", method),
		zap.String("url", fullURL))

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create request for %s %s", method, path)
	}

	// Apply authentication headers
	r.applyAuthHeaders(req)

	// Set content type for POST/PUT/PATCH requests
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json, application/vnd.schemaregistry+json, application/json")

	return r.httpClient.Do(req)
}

func NewRemoteSchemaRegistry(baseURL string, logger *zap.Logger, opts ...Options) (*SchemaRegistry, error) {
	// Default configuration
	config := remoteRegistryConfig{
		url:          baseURL,
		cacheRefresh: 10 * time.Minute,
		timeout:      5 * time.Second,
		auth:         authConfig{authType: authTypeNone},
	}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Validate the URL
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid base URL")
	}

	// Ensure baseURL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.timeout,
	}

	cache := config.cache
	if cache == nil {
		cache = NewMemoryCache(config.cacheRefresh)
	}

	registry := &SchemaRegistry{
		config:     config,
		httpClient: httpClient,
		baseURL:    baseURL,
		cache:      cache,
		compiler:   jsonschema.NewCompiler(),
		logger:     logger,
	}

	// Pre-load OCPP schemas

	return registry, nil
}

// buildSubjectName constructs a subject name from OCPP version and action.
// Format: ocpp-{version}-{action}
// Example: ocpp-1.6-BootNotificationRequest
func buildSubjectName(ocppVersion ocpp.Version, action string) string {
	versionStr := strings.ReplaceAll(ocppVersion.String(), ".", "-")
	return fmt.Sprintf("ocpp-%s-%s", versionStr, action)
}

// getLatestVersion fetches the latest version number for a subject from the remote registry.
func (r *SchemaRegistry) getLatestVersion(ctx context.Context, subject string) (int, error) {
	path := fmt.Sprintf("subjects/%s/versions", url.PathEscape(subject))
	resp, err := r.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get versions for subject %s", subject)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read response body for subject %s", subject)
	}

	var versions []int
	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(bodyBytes, &versions); err != nil {
			return 0, errors.Wrapf(err, "failed to parse versions response for subject %s", subject)
		}
	case http.StatusNotFound:
		return 0, errors.Errorf("subject %s not found", subject)
	case http.StatusInternalServerError:
		return 0, errors.Errorf("internal server error when fetching versions for subject %s", subject)
	default:
		return 0, errors.Errorf("unexpected status code %d when fetching versions for subject %s", resp.StatusCode, subject)
	}

	if len(versions) == 0 {
		return 0, errors.Errorf("no versions found for subject %s", subject)
	}

	// Find the latest version (maximum version number)
	return slices.Max(versions), nil
}

type schemaResponse struct {
	Schema string `json:"schema"`
}

// fetchSchemaFromRemote fetches a schema from the remote registry for a given subject and version.
func (r *SchemaRegistry) fetchSchemaFromRemote(ctx context.Context, subject string, version int) (json.RawMessage, error) {
	versionStr := strconv.Itoa(version)
	path := fmt.Sprintf("subjects/%s/versions/%s/schema", url.PathEscape(subject), url.PathEscape(versionStr))
	resp, err := r.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch schema for subject %s version %d", subject, version)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body for subject %s version %d", subject, version)
	}

	var response schemaResponse

	switch resp.StatusCode {
	case http.StatusOK:
		// Try to parse as structured response first
		if err := json.Unmarshal(bodyBytes, &response); err == nil && response.Schema != "" {
			return json.RawMessage(response.Schema), nil
		}
		// If structured parsing fails, try as direct string
		var schemaStr string
		if err := json.Unmarshal(bodyBytes, &schemaStr); err == nil {
			return json.RawMessage(schemaStr), nil
		}
		// If both fail, return raw bytes
		return bodyBytes, nil
	case http.StatusNotFound:
		return nil, errors.Errorf("schema not found for subject %s version %d", subject, version)
	case http.StatusUnprocessableEntity:
		return nil, errors.Errorf("invalid request for subject %s version %d", subject, version)
	case http.StatusInternalServerError:
		return nil, errors.Errorf("internal server error when fetching schema for subject %s version %d", subject, version)
	default:
		return nil, errors.Errorf("unexpected status code %d when fetching schema for subject %s version %d", resp.StatusCode, subject, version)
	}
}

type RegisterSchemaRequest struct {
	Schema     string `json:"schema"`
	SchemaType string `json:"schemaType"`
}

func (r *SchemaRegistry) RegisterSchema(ctx context.Context, ocppVersion ocpp.Version, action string, rawSchema json.RawMessage) error {
	logger := r.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Registering schema to remote registry")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, "Request") || strings.HasSuffix(action, "Response")) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", action)
	}

	subject := buildSubjectName(ocppVersion, action)

	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()

	// Validate and normalize the schema before sending
	// First, try to compile it to ensure it's valid JSON Schema
	_, err := r.compiler.Compile(rawSchema)
	if err != nil {
		return errors.Wrapf(err, "invalid JSON schema format for subject %s", subject)
	}

	// Clear any formatting by unmarshaling and re-marshaling as compact JSON
	// This ensures the schema is normalized without any whitespace/formatting or escaping issues
	var schemaObj interface{}
	if err := json.Unmarshal(rawSchema, &schemaObj); err != nil {
		return errors.Wrapf(err, "failed to parse schema JSON for subject %s", subject)
	}

	// Marshal back as compact JSON (no formatting/whitespace)
	// This produces clean, unescaped JSON bytes
	normalizedBytes, err := json.Marshal(schemaObj)
	if err != nil {
		return errors.Wrapf(err, "failed to normalize schema JSON for subject %s", subject)
	}

	// The schema must be sent as a JSON string
	// Convert normalized bytes to string - this is raw JSON without any escaping
	// json.Marshal will properly escape this string when serializing the request body

	logger.Debug("Schema string prepared for registration", zap.String("subject", subject))

	// Create the request payload
	schemaStr := string(normalizedBytes)
	payload := RegisterSchemaRequest{
		Schema:     schemaStr,
		SchemaType: "JSON",
	}

	// Serialize the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to serialize request payload for subject %s", subject)
	}

	// Make the request
	path := fmt.Sprintf("subjects/%s/versions", url.PathEscape(subject))
	resp, err := r.doRequest(ctx, http.MethodPost, path, payloadBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to register schema for subject %s", subject)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response body for subject %s", subject)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Success - schema registered
	case http.StatusConflict:
		return errors.Errorf("schema already exists for subject %s", subject)
	case http.StatusUnprocessableEntity:
		// Try to get more details from the error response
		var errorResponse struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil && errorResponse.Message != "" {
			return errors.Errorf("invalid schema format for subject %s: %s", subject, errorResponse.Message)
		}
		return errors.Errorf("invalid schema format for subject %s", subject)
	case http.StatusInternalServerError:
		return errors.Errorf("internal server error when registering schema for subject %s", subject)
	default:
		return errors.Errorf("unexpected status code %d when registering schema for subject %s", resp.StatusCode, subject)
	}

	// Invalidate cache for this schema
	r.cache.Delete(ctx, ocppVersion, action)

	logger.Debug("Successfully registered schema to remote registry")
	return nil
}

func (r *SchemaRegistry) DeleteSchema(ctx context.Context, ocppVersion ocpp.Version, action string) error {
	logger := r.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Deleting schema from remote registry")

	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	if !(strings.HasSuffix(action, file_registry.RequestSuffix) || strings.HasSuffix(action, file_registry.ResponseSuffix)) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", action)
	}

	subject := buildSubjectName(ocppVersion, action)

	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()

	path := fmt.Sprintf("subjects/%s", url.PathEscape(subject))
	resp, err := r.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to delete schema for subject %s", subject)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response body for subject %s", subject)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Success
	case http.StatusNotFound:
		return errors.Errorf("schema not found for subject %s", subject)
	case http.StatusUnprocessableEntity:
		var errorResponse struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil && errorResponse.Message != "" {
			return errors.Errorf("cannot delete schema for subject %s: %s", subject, errorResponse.Message)
		}
		return errors.Errorf("cannot delete schema for subject %s", subject)
	case http.StatusInternalServerError:
		return errors.Errorf("internal server error when deleting schema for subject %s", subject)
	default:
		return errors.Errorf("unexpected status code %d when deleting schema for subject %s", resp.StatusCode, subject)
	}

	r.cache.Delete(ctx, ocppVersion, action)

	logger.Debug("Successfully deleted schema from remote registry")
	return nil
}

func (r *SchemaRegistry) GetSchema(ctx context.Context, ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool) {
	logger := r.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Getting schema")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		logger.Warn("Invalid OCPP version")
		return nil, false
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, "Request") || strings.HasSuffix(action, "Response")) {
		logger.Warn("Invalid action name")
		return nil, false
	}

	// Check cache first
	if schema, ok := r.cache.Get(ctx, ocppVersion, action); ok {
		logger.Debug("Returning schema from cache")
		return schema, true
	}

	// Cache miss or expired - fetch from remote
	subject := buildSubjectName(ocppVersion, action)
	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()

	// Get the latest version
	latestVersion, err := r.getLatestVersion(ctx, subject)
	if err != nil {
		logger.Warn("Failed to get latest version", zap.Error(err))
		return nil, false
	}

	// Fetch the schema
	rawSchema, err := r.fetchSchemaFromRemote(ctx, subject, latestVersion)
	if err != nil {
		logger.Warn("Failed to fetch schema from remote", zap.Error(err))
		return nil, false
	}

	// Compile the schema
	schema, err := r.compiler.Compile(rawSchema)
	if err != nil {
		logger.Warn("Failed to compile schema", zap.Error(err))
		return nil, false
	}

	// Update cache
	r.cache.Set(ctx, ocppVersion, action, schema)

	logger.Debug("Successfully fetched and cached schema from remote")
	return schema, true
}

func (r *SchemaRegistry) Type() string {
	return "remote"
}
