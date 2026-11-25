## ca buckets usage

A bucket's storage usage and traffic over a window

### Synopsis

Not the billing basis — egress is billed differently. See the field docs.

```
ca buckets usage BUCKET [flags]
```

### Examples

```
  ca buckets usage logs-7f3a --hours 168
```

### Options

```
  -h, --help        help for usage
      --hours int   The traffic window, 1-720 hours. Defaults to 24.
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets](ca_buckets.md)	 - Manage object storage buckets

