package registries

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
	"strings"
	"sync"
	"time"

	"github.com/kaptinlin/jsonschema"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ChargePi/chargeflow/pkg/ocpp"
)

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

type remoteRegistryConfig struct {
	url string
	// Cache settings
	cacheRefresh time.Duration
	timeout      time.Duration
	auth         authConfig
}

type RemoteOptions func(*remoteRegistryConfig)

func WithCacheRefreshDuration(d time.Duration) RemoteOptions {
	return func(c *remoteRegistryConfig) {
		c.cacheRefresh = d
	}
}

func WithTimeout(d time.Duration) RemoteOptions {
	return func(c *remoteRegistryConfig) {
		c.timeout = d
	}
}

// WithBasicAuth configures basic authentication with username and password.
func WithBasicAuth(username, password string) RemoteOptions {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType: authTypeBasic,
			username: username,
			password: password,
		}
	}
}

// WithBearerToken configures bearer token authentication.
func WithBearerToken(token string) RemoteOptions {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType:    authTypeBearer,
			bearerToken: token,
		}
	}
}

// WithAPIKey configures API key authentication with a custom header name.
// If headerName is empty, it defaults to "X-API-Key".
func WithAPIKey(apiKey, headerName string) RemoteOptions {
	if headerName == "" {
		headerName = "X-API-Key"
	}
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType:     authTypeAPIKey,
			apiKey:       apiKey,
			apiKeyHeader: headerName,
		}
	}
}

// WithCustomHeader configures a custom header for authentication.
func WithCustomHeader(headerName, headerValue string) RemoteOptions {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType:          authTypeCustomHeader,
			customHeaderName:  headerName,
			customHeaderValue: headerValue,
		}
	}
}

type cachedSchema struct {
	schema   *jsonschema.Schema
	cachedAt time.Time
}

// RemoteSchemaRegistry fetches schemas from a remote schema registry service and caches them locally to reduce latency and network calls.
type RemoteSchemaRegistry struct {
	logger *zap.Logger

	config     remoteRegistryConfig
	httpClient *http.Client
	baseURL    string

	mu sync.RWMutex // Protects concurrent access to cache
	// Map of cached schemas per OCPP version and action
	cache map[ocpp.Version]map[string]*cachedSchema

	compiler *jsonschema.Compiler
}

// applyAuthHeaders adds authentication headers to the request based on the auth config.
func (r *RemoteSchemaRegistry) applyAuthHeaders(req *http.Request) {
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
	}
}

// logRequestBody logs the request body if present.
func (r *RemoteSchemaRegistry) logRequestBody(method, url string, bodyBytes []byte) {
	if len(bodyBytes) == 0 {
		r.logger.Info("Executing request",
			zap.String("method", method),
			zap.String("url", url))
		return
	}

	// Try to pretty-print JSON if possible, otherwise use raw string
	var bodyStr string
	var jsonBody interface{}
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		if prettyJSON, err := json.MarshalIndent(jsonBody, "", "  "); err == nil {
			bodyStr = string(prettyJSON)
		} else {
			bodyStr = string(bodyBytes)
		}
	} else {
		bodyStr = string(bodyBytes)
	}

	r.logger.Info("Executing request",
		zap.String("method", method),
		zap.String("url", url),
		zap.String("body", bodyStr))
}

// doRequest performs an HTTP request with authentication and logging.
func (r *RemoteSchemaRegistry) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	fullURL, err := url.JoinPath(r.baseURL, path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build URL for path %s", path)
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
		// Log request body before sending
		if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
			r.logRequestBody(method, fullURL, body)
		}
	}
	r.logger.Info("Executing request",
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

func NewRemoteSchemaRegistry(baseURL string, logger *zap.Logger, opts ...RemoteOptions) (*RemoteSchemaRegistry, error) {
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

	// Ensure baseURL ends with a slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.timeout,
	}

	return &RemoteSchemaRegistry{
		config:     config,
		httpClient: httpClient,
		baseURL:    baseURL,
		cache:      make(map[ocpp.Version]map[string]*cachedSchema),
		compiler:   jsonschema.NewCompiler(),
		logger:     logger,
	}, nil
}

// buildSubjectName constructs a subject name from OCPP version and action.
// Format: ocpp-{version}-{action}
// Example: ocpp-1.6-BootNotificationRequest
func buildSubjectName(ocppVersion ocpp.Version, action string) string {
	versionStr := strings.ReplaceAll(ocppVersion.String(), ".", "-")
	return fmt.Sprintf("ocpp-%s-%s", versionStr, action)
}

// getLatestVersion fetches the latest version number for a subject from the remote registry.
func (r *RemoteSchemaRegistry) getLatestVersion(ctx context.Context, subject string) (int, error) {
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

// fetchSchemaFromRemote fetches a schema from the remote registry for a given subject and version.
func (r *RemoteSchemaRegistry) fetchSchemaFromRemote(ctx context.Context, subject string, version int) (json.RawMessage, error) {
	versionStr := fmt.Sprintf("%d", version)
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

	var schemaResponse struct {
		Schema string `json:"schema"`
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Try to parse as structured response first
		if err := json.Unmarshal(bodyBytes, &schemaResponse); err == nil && schemaResponse.Schema != "" {
			return json.RawMessage(schemaResponse.Schema), nil
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

func (r *RemoteSchemaRegistry) RegisterSchema(ocppVersion ocpp.Version, action string, rawSchema json.RawMessage) error {
	logger := r.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Registering schema to remote registry")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		return errors.Errorf("invalid OCPP version: %s", ocppVersion)
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, RequestSuffix) || strings.HasSuffix(action, ResponseSuffix)) {
		return errors.Errorf("action must end with 'Request' or 'Response': %s", action)
	}

	subject := buildSubjectName(ocppVersion, action)

	ctx, cancel := context.WithTimeout(context.Background(), r.config.timeout)
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

	// Validate the normalized JSON is still valid by compiling it again
	_, err = r.compiler.Compile(normalizedBytes)
	if err != nil {
		logger.Error("Normalized schema validation failed",
			zap.Error(err),
			zap.String("normalizedSchema", string(normalizedBytes)),
			zap.String("originalSchema", string(rawSchema)))
		return errors.Wrapf(err, "normalized schema is invalid JSON Schema for subject %s", subject)
	}

	// The schema must be sent as a JSON string
	// Convert normalized bytes to string - this is raw JSON without any escaping
	// json.Marshal will properly escape this string when serializing the request body
	schemaStr := string(normalizedBytes)

	schemaPreview := schemaStr
	if len(schemaPreview) > 100 {
		schemaPreview = schemaPreview[:100] + "..."
	}
	logger.Debug("Schema string prepared for registration",
		zap.String("subject", subject),
		zap.Int("schemaLength", len(schemaStr)),
		zap.String("schemaPreview", schemaPreview))

	// Create the request payload
	schemaType := "JSONSCHEMA"
	payload := map[string]interface{}{
		"schema":     schemaStr,
		"schemaType": schemaType,
	}

	// Serialize the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to serialize request payload for subject %s", subject)
	}

	logger.Debug("Normalized schema for registration",
		zap.String("subject", subject),
		zap.Int("schemaLength", len(schemaStr)),
		zap.Int("payloadLength", len(payloadBytes)))

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
	r.mu.Lock()
	if _, exists := r.cache[ocppVersion]; exists {
		delete(r.cache[ocppVersion], action)
	}
	r.mu.Unlock()

	logger.Debug("Successfully registered schema to remote registry")
	return nil
}

func (r *RemoteSchemaRegistry) GetSchema(ocppVersion ocpp.Version, action string) (*jsonschema.Schema, bool) {
	logger := r.logger.With(zap.String("ocppVersion", ocppVersion.String()), zap.String("action", action))
	logger.Debug("Getting schema")

	// Validate the OCPP version
	if !ocpp.IsValidProtocolVersion(ocppVersion) {
		logger.Warn("Invalid OCPP version")
		return nil, false
	}

	// Must be a valid action name ending with "Request" or "Response"
	if !(strings.HasSuffix(action, RequestSuffix) || strings.HasSuffix(action, ResponseSuffix)) {
		logger.Warn("Invalid action name")
		return nil, false
	}

	// Check cache first
	r.mu.RLock()
	if schemas, exists := r.cache[ocppVersion]; exists {
		if cached, exists := schemas[action]; exists {
			// Check if cache is still valid
			if time.Since(cached.cachedAt) < r.config.cacheRefresh {
				logger.Debug("Returning schema from cache")
				r.mu.RUnlock()
				return cached.schema, true
			}
			logger.Debug("Cache expired, fetching from remote")
		}
	}
	r.mu.RUnlock()

	// Cache miss or expired - fetch from remote
	subject := buildSubjectName(ocppVersion, action)
	ctx, cancel := context.WithTimeout(context.Background(), r.config.timeout)
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
	r.mu.Lock()
	if _, exists := r.cache[ocppVersion]; !exists {
		r.cache[ocppVersion] = make(map[string]*cachedSchema)
	}
	r.cache[ocppVersion][action] = &cachedSchema{
		schema:   schema,
		cachedAt: time.Now(),
	}
	r.mu.Unlock()

	logger.Debug("Successfully fetched and cached schema from remote")
	return schema, true
}

func (r *RemoteSchemaRegistry) Type() string {
	return "remote"
}
