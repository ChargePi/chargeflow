.PHONY:gen format lint gen-schema-registry-client

gen: gen-mocks gen-schema-registry-client

gen-mocks:
	mockery

gen-schema-registry-client:
	@echo "Fetching OpenAPI schema from Redpanda..."
	@mkdir -p gen/schema-registry
	@curl -sSL https://docs.redpanda.com/api/doc/schema-registry.yaml -o gen/schema-registry/openapi2.yaml
	@echo "Converting OpenAPI 2.0 to OpenAPI 3.0..."
	@curl -sSL -X POST "https://converter.swagger.io/api/convert" \
		-H "Content-Type: application/yaml" \
		-H "Accept: application/yaml" \
		--data-binary @gen/schema-registry/openapi2.yaml \
		-o gen/schema-registry/openapi.yaml
	@echo "Generating Go HTTP client..."
	@cd gen/schema-registry && go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config oapi-codegen.yaml openapi.yaml
	@echo "Formatting generated code..."
	@go fmt ./gen/schema-registry/client.go
	@echo "Cleaning up schema files..."
	@rm -f gen/schema-registry/openapi2.yaml gen/schema-registry/openapi.yaml
	@echo "Schema registry client generated successfully in gen/schema-registry/client.go"

lint:
	golangci-lint run

format:
	golangci-lint fmt