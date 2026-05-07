# Validating messages from a file

Use the `-f` flag to read OCPP messages from a file instead of passing them as a command-line
argument. Each line must be a single, complete OCPP JSON string; blank lines are ignored.

```bash
chargeflow validate -f messages.txt
```

Example `messages.txt`:

```
[2, "1", "BootNotification", {"chargePointVendor": "TestVendor", "chargePointModel": "TestModel"}]
[2, "2", "Heartbeat", {}]
[3, "1", {"status": "Accepted", "currentTime": "2024-01-01T00:00:00Z", "interval": 300}]
```

> [!NOTE]
> Response messages (type `3`) require the `--response-type` flag so ChargeFlow knows which schema
> to validate against, e.g. `--response-type BootNotificationResponse`.

## Saving the report to a file

Use `-o` to write the validation report to a file instead of stdout. Supported extensions are
`.json`, `.csv`, and `.txt`.

```bash
chargeflow validate -f messages.txt -o report.json
chargeflow validate -f messages.txt -o report.csv
chargeflow validate -f messages.txt -o report.txt
```

## Specifying the OCPP version

The default version is `1.6`. Use `--version` (`-v`) to change it.

```bash
chargeflow --version 2.0.1 validate -f messages.txt -o report.json
```