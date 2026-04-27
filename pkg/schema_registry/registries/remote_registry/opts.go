package remote_registry

import "time"

type Options func(*remoteRegistryConfig)

func WithCacheRefreshDuration(d time.Duration) Options {
	return func(c *remoteRegistryConfig) {
		c.cacheRefresh = d
	}
}

// WithCache replaces the default MemoryCache with a custom Cache implementation.
func WithCache(cache Cache) Options {
	return func(c *remoteRegistryConfig) {
		c.cache = cache
	}
}

func WithTimeout(d time.Duration) Options {
	return func(c *remoteRegistryConfig) {
		c.timeout = d
	}
}

// WithBasicAuth configures basic authentication with username and password.
func WithBasicAuth(username, password string) Options {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType: authTypeBasic,
			username: username,
			password: password,
		}
	}
}

// WithBearerToken configures bearer token authentication.
func WithBearerToken(token string) Options {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType:    authTypeBearer,
			bearerToken: token,
		}
	}
}

// WithAPIKey configures API key authentication with a custom header name.
// If headerName is empty, it defaults to "X-API-Key".
func WithAPIKey(apiKey, headerName string) Options {
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
func WithCustomHeader(headerName, headerValue string) Options {
	return func(c *remoteRegistryConfig) {
		c.auth = authConfig{
			authType:          authTypeCustomHeader,
			customHeaderName:  headerName,
			customHeaderValue: headerValue,
		}
	}
}
