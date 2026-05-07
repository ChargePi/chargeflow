# Custom and vendor-specific schemas

ChargeFlow ships with built-in schemas for all supported OCPP versions. You can extend or override
these with your own JSON schemas to handle vendor-specific extensions or non-standard field
constraints.

## Vendor- and model-specific validation

Pass `--vendor` (`-V`) and/or `--model` (`-m`) to select schemas scoped to a specific charging
station. When both flags are provided, ChargeFlow looks for schemas registered under that
vendor/model combination before falling back to the standard built-in schemas.

```bash
chargeflow --vendor Acme --model FastCharger validate \
  '[2, "1", "BootNotification", {"chargePointVendor": "Acme", "chargePointModel": "FastCharger"}]'
```

## Loading schemas from a local directory

Use `--schemas` (`-a`) on the `validate` command to point ChargeFlow at a folder of custom JSON
schema files. File names must match the OCPP action name they cover, e.g.
`BootNotificationRequest.json`. Custom schemas **replace** the built-in schemas for the actions
they cover.

```bash
chargeflow validate --schemas ./vendor-schemas \
  '[2, "1", "BootNotification", {"chargePointVendor": "Acme", "chargePointModel": "FastCharger"}]'
```

Combining vendor/model flags with a custom schema folder is the typical pattern for validating
messages from a specific charging station:

```bash
chargeflow --vendor Acme --model FastCharger \
  validate --schemas ./vendor-schemas -f messages.txt -o report.json
```

## Schema file naming convention

Schema files must be named after the OCPP action with a `.json` extension:

```
BootNotificationRequest.json
BootNotificationResponse.json
DataTransfer.json
...
```

ChargeFlow strips the `.json` suffix and uses the remaining string as the action name when
registering the schema internally.