# Remote schema registry

ChargeFlow can push schemas to and remove schemas from a Kafka-compatible schema registry such as
[Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/index.html)
or [Redpanda Schema Registry](https://docs.redpanda.com/current/manage/schema-reg/).

All `schema` subcommands share a set of common flags for the registry URL and authentication.

## Common flags

| Flag | Description | Default |
|------|-------------|---------|
| `--url` | Remote schema registry URL (required) | — |
| `--auth-type` | Authentication type: `basic`, `bearer`, `api-key`, or omit for none | — |
| `--username` | Username for basic authentication | — |
| `--password` | Password for basic authentication | — |
| `--bearer-token` | Bearer token | — |
| `--api-key` | API key value | — |
| `--api-key-header` | Header name for the API key | `X-API-Key` |
| `--custom-header` | Custom header name | — |
| `--custom-value` | Custom header value | — |
| `--timeout` | Request timeout | `5s` |

## Registering schemas

Use `schema register` to upload JSON schemas to the registry.

### Register a single schema file

`--action` is required when registering a single file.

```bash
chargeflow schema --url http://localhost:8081 \
  register --file BootNotificationRequest.json --action BootNotificationRequest
```

### Register all schemas from a directory

ChargeFlow reads every `.json` file in the directory and derives the action name from the file name.

```bash
chargeflow schema --url http://localhost:8081 --version 2.0.1 \
  register --dir ./schemas
```

### Register vendor-specific schemas

Supply `--vendor` and `--model` on the root command to scope the schemas to a specific charging
station make and model.

```bash
chargeflow --vendor Acme --model FastCharger \
  schema --url http://localhost:8081 \
  register --dir ./vendor-schemas
```

## Deleting schemas

Use `schema remove` to delete schemas from the registry. Exactly one of `--action`, `--file`, or
`--dir` must be specified.

### Delete by action name

```bash
chargeflow schema --url http://localhost:8081 \
  remove --action BootNotificationRequest
```

### Derive the action from a file name

ChargeFlow strips the `.json` suffix and uses the result as the action name.

```bash
chargeflow schema --url http://localhost:8081 \
  remove --file BootNotificationRequest.json
```

### Delete all schemas matching a directory

Removes every schema whose action name matches a `.json` file found in the directory.

```bash
chargeflow schema --url http://localhost:8081 \
  remove --dir ./schemas
```

### Delete vendor-specific schemas

```bash
chargeflow --vendor Acme --model FastCharger \
  schema --url http://localhost:8081 \
  remove --action DataTransfer
```

## Authentication examples

### Basic authentication

```bash
chargeflow schema --url http://registry:8081 \
  --auth-type basic --username admin --password secret \
  register --dir ./schemas
```

### Bearer token

```bash
chargeflow schema --url https://registry.example.com \
  --auth-type bearer --bearer-token eyJ... \
  register --file MySchema.json --action MyAction
```

### API key

The default header is `X-API-Key`. Override it with `--api-key-header` if your registry uses a
different header name.

```bash
chargeflow schema --url https://registry.example.com \
  --auth-type api-key --api-key abc123 \
  register --dir ./schemas
```

### Custom header

```bash
chargeflow schema --url https://registry.example.com \
  --custom-header X-Registry-Auth --custom-value my-secret \
  register --dir ./schemas
```