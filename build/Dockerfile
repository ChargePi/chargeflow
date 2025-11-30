FROM golang:1.24.3 AS build
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

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o ./chargeflow .

FROM debian:buster-slim as app

RUN apt-get update

SHELL ["/bin/sh", "-l", "-c"]

ARG user=app
ARG group=docker
RUN addgroup $group
RUN adduser \
    --disabled-password \
    --gecos "" \
    --ingroup $group \
    $user
USER $user

WORKDIR /usr/local/bin

# Copy the binary and set execute permissions
COPY --chown=$user:$group --from=build /app/chargeflow /usr/local/bin/chargeflow
RUN chmod +x /usr/local/bin/chargeflow

CMD ["/usr/local/bin/chargeflow"]