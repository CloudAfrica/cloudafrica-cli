## ca vpc update

Rename a VPC

### Synopsis

Changes the display name only; the id, range and attached VMs are untouched. Answered from the backend's own post-write read, so the new name renders without waiting on a projection that can lag a sync cycle.

```
ca vpc update VPC [flags]
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for update
      --name string   The new display name. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpc](ca_vpc.md)	 - Manage VPCs

