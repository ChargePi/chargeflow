FROM golang:1.23.0 AS build
WORKDIR /app/src

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/root/.cache/go-build
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOOS=linux

# Leverage build cache by copying go.mod and go.sum first
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download
RUN go mod verify

# Copy the rest of the source code
COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o ../service ./cmd/service

# Debug stage using Delve
FROM build AS debug

COPY --from=build /app/src /reporting-service/src

WORKDIR /reporting-service/src

# Install Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest
# Build the application with debugging flags
RUN go build -gcflags="all=-N -l" -o ./service ./cmd/service

CMD ["/go/bin/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "./service"]

FROM debian:buster-slim as app

RUN apt-get update && \
    apt-get install -y wkhtmltopdf && \
    apt-get install -y curl

SHELL ["/bin/sh", "-l", "-c"]

# Add curl for health checks
ARG user=app
ARG group=docker
RUN addgroup $group
RUN adduser \
    --disabled-password \
    --gecos "" \
    --ingroup $group \
    $user
USER $user
WORKDIR /usr/local/service

# Copy the binary and set execute permissions
COPY --chown=$user:$group --from=build /app/service /usr/local/bin/service
RUN chmod +x /usr/local/bin/service


# Add a health check
HEALTHCHECK --interval=5s --timeout=3s --retries=3 \
    CMD curl --fail http://localhost:80/healthz || exit 1

CMD ["/usr/local/bin/service"]