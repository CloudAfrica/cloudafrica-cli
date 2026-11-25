## ca vpcs create

Create a VPC

### Synopsis

Creates one VPC, owned by this token's account. Requires write:network. Must carry an Idempotency-Key header, which the backend dedupes on, so a retried create returns the first VPC rather than a second. Synchronous: returns 201 with the VPC. A no-capacity answer is 409.

```
ca vpcs create [flags]
```

### Examples

```
  ca vpcs create --name web
```

### Options

```
  -d, --data string          the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string   Optional description; absent leaves it unset.
  -h, --help                 help for create
      --name string          The VPC's display name. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs](ca_vpcs.md)	 - Manage VPCs

