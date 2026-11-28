## ca placement-groups create

Create a placement group

### Synopsis

Creates an empty placement group in the datacenter --datacenter-id names. spread keeps its VMs on different hosts; gather keeps them together.

```
ca placement-groups create [flags]
```

### Examples

```
  ca placement-groups create --name spread --strategy spread --datacenter-id <dc-slug>
```

### Options

```
  -d, --data string            the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --datacenter-id string   the datacenter's slug, from ca datacenters list (required)
  -h, --help                   help for create
      --name string            Display name. (required)
      --strategy string        spread (anti-affinity) or gather (affinity). (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca placement-groups](ca_placement-groups.md)	 - Manage placement groups

