## ca placement-groups create

Create a placement group

### Synopsis

Created empty and owned by this token's account — the body names no account. `datacenterId` must be a slug from GET /v1/placement-groups/datacenters. Must carry an Idempotency-Key header: a retry under the same key replays the first call's stored response rather than starting a second, 409 while that first call is still in flight, and 422 if the key is reused for a different request.

```
ca placement-groups create [flags]
```

### Options

```
  -d, --data string            the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --datacenter-id string   A datacenter slug from GET /v1/placement-groups/datacenters. (required)
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

