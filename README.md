# ChargeFlow

A CLI tool for analyzing your raw OCPP JSON messages. Useful for debugging and compatibility checks
with various Charge Point Management Systems or Charge Point implementations.

## Features

- [x] Parse Raw OCPP JSON messages
- [x] Support for OCPP 1.6, 2.0.1 and 2.1
- [x] Request and Response payload validation
- [x] Validate messages from a file

### Roadmap

- [ ] Generate human-readable reports
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
chargeflow --version 1.6 validate '[2, "123456", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]'
chargeflow validate -f ./message.txt

Flags:
  -f, --file string            Path to a file containing the OCPP message to validate. If this flag is set, the message will be read from the file instead of the command line argument.
  -h, --help                   help for validate
  -r, --response-type string   Response type to validate against (e.g. 'BootNotificationResponse'). Currently needed if you want to validate a single response message. 
  -a, --schemas string         Path to additional OCPP schemas folder

Global Flags:
  -d, --debug            Enable debug mode
  -v, --version string   OCPP version to use (1.6, 2.0.1 or 2.1) (default "1.6")
```

ChargeFlow will automatically determine whether it's a request or response message. All you need to provide is a OCPP
version!

> [!NOTE]
> If you want to validate a response message, you need to specify the response type using the `--response-type`
> flag.

Additionally, you can specify a custom path to vendor-specific OCPP schemas using the `--schemas` flag.

> [!TIP]
> You can now also validate multiple messages (both request and responses) from a file using the `-f` flag.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## Contributing

We welcome contributions to this project! Please read our [contributing guidelines](CONTRIBUTING.md) for more
information on how to get started.