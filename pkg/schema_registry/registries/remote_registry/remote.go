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

	"github.com/ChargePi/chargeflow/pkg/ocpp"
	"github.com/ChargePi/chargeflow/pkg/schema_registry"
	"github.com/ChargePi/chargeflow/pkg/schema_registry/registries/file_registry"
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

// buildSubjectName constructs a subject name from OCPP version, action, and optional vendor/model.
// Base format:  ocpp-{version}-{action}
// With vendor/model: {vendor}-{model}-ocpp-{version}-{action}
// Omitted parts are skipped when empty.
func buildSubjectName(ocppVersion ocpp.Version, action, vendor, model string) string {
	versionStr := strings.ReplaceAll(ocppVersion.String(), ".", "-")
	base := fmt.Sprintf("ocpp-%s-%s", versionStr, action)
	if vendor == "" && model == "" {
		return base
	}
	parts := make([]string, 0, 3)
	if vendor != "" {
		parts = append(parts, vendor)
	}
	if model != "" {
		parts = append(parts, model)
	}
	parts = append(parts, base)
	return strings.Join(parts, "-")
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

type registerSchemaPayload struct {
	Schema     string `json:"schema"`
	SchemaType string `json:"schemaType"`
}

func (r *SchemaRegistry) RegisterSchema(ctx context.Context, req schema_registry.CreateSchemaRequest) error {
	logger := r.logger.With(zap.String("ocppVersion", req.OcppContext.Version.String()), zap.String("action", req.Action))
	logger.Debug("Registering schema to remote registry")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(req.OcppContext.Version) {
		return errors.Errorf("invalid OCPP version: %s", req.OcppContext.Version)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(req.Action, "Request") || strings.HasSuffix(req.Action, "Response")) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", req.Action)
	}

	subject := buildSubjectName(req.OcppContext.Version, req.Action, req.OcppContext.Vendor, req.OcppContext.Model)

	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()

	// Validate and normalize the schema before sending
	// First, try to compile it to ensure it's valid JSON Schema
	_, err := r.compiler.Compile(req.Schema)
	if err != nil {
		return errors.Wrapf(err, "invalid JSON schema format for subject %s", subject)
	}

	// Clear any formatting by unmarshaling and re-marshaling as compact JSON
	// This ensures the schema is normalized without any whitespace/formatting or escaping issues
	var schemaObj interface{}
	if err := json.Unmarshal(req.Schema, &schemaObj); err != nil {
		return errors.Wrapf(err, "failed to parse schema JSON for subject %s", subject)
	}

	// Marshal back as compact JSON (no formatting/whitespace)
	// This produces clean, unescaped JSON bytes
	normalizedBytes, err := json.Marshal(schemaObj)
	if err != nil {
		return errors.Wrapf(err, "failed to normalize schema JSON for subject %s", subject)
	}

	logger.Debug("Schema string prepared for registration", zap.String("subject", subject))

	payload := registerSchemaPayload{
		Schema:     string(normalizedBytes),
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
	r.cache.Delete(ctx, subject)

	logger.Debug("Successfully registered schema to remote registry")
	return nil
}

func (r *SchemaRegistry) DeleteSchema(ctx context.Context, req schema_registry.DeleteSchemaRequest) error {
	logger := r.logger.With(zap.String("ocppVersion", req.OcppContext.Version.String()), zap.String("action", req.Action))
	logger.Debug("Deleting schema from remote registry")

	if !ocpp.IsValidProtocolVersion(req.OcppContext.Version) {
		return errors.Errorf("invalid OCPP version: %s", req.OcppContext.Version)
	}

	if !(strings.HasSuffix(req.Action, file_registry.RequestSuffix) || strings.HasSuffix(req.Action, file_registry.ResponseSuffix)) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", req.Action)
	}

	subject := buildSubjectName(req.OcppContext.Version, req.Action, req.OcppContext.Vendor, req.OcppContext.Model)

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

	r.cache.Delete(ctx, subject)

	logger.Debug("Successfully deleted schema from remote registry")
	return nil
}

// fetchAndCacheSchema fetches a schema for the given subject from the remote registry,
// compiles it, caches it, and returns it. Returns nil, false if anything fails.
func (r *SchemaRegistry) fetchAndCacheSchema(ctx context.Context, subject string) (*jsonschema.Schema, bool) {
	latestVersion, err := r.getLatestVersion(ctx, subject)
	if err != nil {
		r.logger.Warn("Failed to get latest version", zap.String("subject", subject), zap.Error(err))
		return nil, false
	}

	rawSchema, err := r.fetchSchemaFromRemote(ctx, subject, latestVersion)
	if err != nil {
		r.logger.Warn("Failed to fetch schema from remote", zap.String("subject", subject), zap.Error(err))
		return nil, false
	}

	schema, err := r.compiler.Compile(rawSchema)
	if err != nil {
		r.logger.Warn("Failed to compile schema", zap.String("subject", subject), zap.Error(err))
		return nil, false
	}

	r.cache.Set(ctx, subject, schema)
	return schema, true
}

func (r *SchemaRegistry) GetSchema(ctx context.Context, req schema_registry.GetSchemaRequest) (*jsonschema.Schema, bool) {
	logger := r.logger.With(
		zap.String("ocppVersion", req.OcppContext.Version.String()),
		zap.String("action", req.Action),
		zap.String("vendor", req.OcppContext.Vendor),
		zap.String("model", req.OcppContext.Model),
	)
	logger.Debug("Getting schema")

	if !ocpp.IsValidProtocolVersion(req.OcppContext.Version) {
		logger.Warn("Invalid OCPP version")
		return nil, false
	}

	if !(strings.HasSuffix(req.Action, "Request") || strings.HasSuffix(req.Action, "Response")) {
		logger.Warn("Invalid action name")
		return nil, false
	}

	ctx, cancel := context.WithTimeout(ctx, r.config.timeout)
	defer cancel()

	// When vendor or model is provided, try the specific subject first.
	if req.OcppContext.Vendor != "" || req.OcppContext.Model != "" {
		specificSubject := buildSubjectName(req.OcppContext.Version, req.Action, req.OcppContext.Vendor, req.OcppContext.Model)
		if schema, ok := r.cache.Get(ctx, specificSubject); ok {
			logger.Debug("Returning vendor/model-specific schema from cache")
			return schema, true
		}
		if schema, ok := r.fetchAndCacheSchema(ctx, specificSubject); ok {
			logger.Debug("Returning vendor/model-specific schema from remote")
			return schema, true
		}
		logger.Debug("No vendor/model-specific schema found, falling back to base schema")
	}

	baseSubject := buildSubjectName(req.OcppContext.Version, req.Action, "", "")
	if schema, ok := r.cache.Get(ctx, baseSubject); ok {
		logger.Debug("Returning base schema from cache")
		return schema, true
	}

	schema, ok := r.fetchAndCacheSchema(ctx, baseSubject)
	if ok {
		logger.Debug("Successfully fetched and cached base schema from remote")
	}
	return schema, ok
}

func (r *SchemaRegistry) Type() string {
	return "remote"
}
