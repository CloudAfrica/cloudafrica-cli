## ca vpcs internal-dns get

Get a VPC's internal DNS

### Synopsis

Whether internal DNS is on for the VPC, whether it is serving, and the names its VMs and virtual IPs resolve. 503 when the platform cannot be asked right now.

```
ca vpcs internal-dns get [flags]
```

### Options

```
  -h, --help         help for get
      --vpc string   the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs internal-dns](ca_vpcs_internal-dns.md)	 - Read and change a VPC's internal DNS

