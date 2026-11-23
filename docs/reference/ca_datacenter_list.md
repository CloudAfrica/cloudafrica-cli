## ca datacenter list

List the datacenters a placement group can be created in

### Synopsis

The slug on each entry is the only value create accepts as `datacenterId`. Create refuses anything else — a display name, or a site this account cannot place into — because a group in a site the placement engine cannot match is silently inert. Read the slug from here rather than composing one.

```
ca datacenter list [flags]
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

* [ca datacenter](ca_datacenter.md)	 - List datacenters and check placement feasibility

