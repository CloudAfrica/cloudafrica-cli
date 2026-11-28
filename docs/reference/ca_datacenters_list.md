## ca datacenters list

List the datacenters a placement group can be created in

### Synopsis

Lists the datacenters, each with its slug. `ca placement-groups create --datacenter-id` takes only a slug from this list: a display name, or a datacenter the account cannot place into, is refused.

```
ca datacenters list [flags]
```

### Examples

```
  ca datacenters list
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca datacenters](ca_datacenters.md)	 - List datacenters and check placement feasibility

