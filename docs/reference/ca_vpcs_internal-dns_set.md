## ca vpcs internal-dns set

Turn a VPC's internal DNS on or off

### Synopsis

Send {"enabled": true} to turn it on or {"enabled": false} to turn it off; repeating the same request changes nothing. Turning it on checks the VPC first and answers 409 with a sentence saying why when it cannot be served. Turning it off drains: the names keep resolving for guests whose DHCP lease still names the resolver, and state is winding_down until then. The answer is the state the change leads to, setting_up or winding_down when it could not be read back yet: read it again with GET.

```
ca vpcs internal-dns set [flags]
```

### Examples

```
  ca vpcs internal-dns set --vpc web --enabled=false
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --enabled       true turns internal DNS on, false turns it off. (required)
  -h, --help          help for set
      --vpc string    the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs internal-dns](ca_vpcs_internal-dns.md)	 - Read and change a VPC's internal DNS

