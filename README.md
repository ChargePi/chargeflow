# ChargeFlow

A CLI tool for analyzing your raw OCPP JSON messages. Useful for debugging and compatibility checks
with various Charge Point Management Systems or Charge Point implementations.

## Features

- [x] Parse Raw OCPP JSON messages
- [x] Support for OCPP 1.6 and 2.0.1
- [x] Request and Response payload validation

### Roadmap

- [ ] Generate human-readable reports
- [ ] Validate messages from a file
- [ ] Support for OCPP 2.1
- [ ] Support for signed messages
- [ ] Compatibility checks
- [ ] Remote schema registry for vendors and models

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
chargeflow validate

Validate the OCPP message(s) against the registered OCPP schema(s).

Usage:
  chargeflow validate [flags]

Examples:
chargeflow --version 1.6 validate '[1, "123456", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]'

Flags:
  -h, --help             help for validate
  -a, --schemas string   Path to additional OCPP schemas folder

Global Flags:
  -d, --debug            Enable debug mode
  -v, --version string   OCPP version to use (1.6 or 2.0.1) (default "1.6")
```

ChargeFlow will automatically determine whether it's a request or response message. All you need to provide is a OCPP
version!

Additionally, you can specify a custom path to vendor-specific OCPP schemas using the `--schemas` flag.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## Contributing

We welcome contributions to this project! Please read our [contributing guidelines](CONTRIBUTING.md) for more
information on how to get started.