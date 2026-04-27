# ChargeFlow

A CLI tool for analyzing your raw OCPP JSON messages. Useful for debugging and compatibility checks
with various Charge Point Management Systems or Charge Point implementations.

## Features

- [x] Validate Raw OCPP JSON messages against multiple OCPP schemas
- [x] Generate human-readable reports
- [x] Support for remote schema registries using Kafka-compatible Schemas Registry APIs
- [x] Bring your own OCPP schemas for vendor-specific extensions

## Compatibility matrix

|          OCPP specification | Supported |   
|----------------------------:|:---------:|
|                    OCPP 1.6 |     ✅     |
| OCPP 1.6 Security Extension |     ✅     |
|                  OCPP 2.0.1 |     ✅     |
|                    OCPP 2.1 |     ✅     |

### Roadmap

- [ ] Support for signed messages
- [ ] Compatibility checks

## Installation

You can install ChargeFlow by downloading the binary.

```bash

```

## Usage

You can use ChargeFlow to validate OCPP messages by running the following command:

```bash
chargeflow validate '[2, "123456", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]'
```

For more options, you can run:

```bash
Usage:
  chargeflow [flags]
  chargeflow [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  schema      Manage schemas on a remote schema registry
  validate    Validate the OCPP message(s) against the registered OCPP schemas

Flags:
  -d, --debug            Enable debug mode
  -h, --help             help for chargeflow
  -v, --version string   OCPP version to use (1.6, 2.0.1 or 2.1) (default "1.6")
```

ChargeFlow will automatically determine whether it's a request or response message. All you need to provide is a OCPP
version!

> [!NOTE]
> If you want to validate a response message, you need to specify the response type using the `--response-type`
> flag.

Additionally, you can specify a custom path to vendor-specific OCPP schemas using the `--schemas` flag.

> [!TIP]
> You can also validate multiple OCPP messages from a file using the `-f` flag.
> The file should be a newline-separated list of JSON strings.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## Contributing

We welcome contributions to this project! Please read our [contributing guidelines](CONTRIBUTING.md) for more
information on how to get started.
