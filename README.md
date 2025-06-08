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
chargeflow validate 1.6 [1234567, "1", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]
```

ChargeFlow will automatically determine whether it's a request or response message. All you need to provide is a OCPP
version!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## Contributing

We welcome contributions to this project! Please read our [contributing guidelines](CONTRIBUTING.md) for more
information on how to get started.