## ca billing accrual

Get the account's month-to-date accrual

### Synopsis

A reading from the usage ledger, not an invoice: one total for the whole account across every billed resource type (VMs, backups, buckets, VPCs), never for one type. `total` is in rands. Optionally as-of a past instant; the window is always clamped to the meter's coverage frontier.

```
ca billing accrual [flags]
```

### Examples

```
  ca billing accrual
```

### Options

```
      --as-of string   ISO-8601 instant to read the accrual as of. Defaults to now.
  -h, --help           help for accrual
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca billing](ca_billing.md)	 - Read the balance and accrual, and manage the billing details

