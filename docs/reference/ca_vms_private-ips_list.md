## ca vms private-ips list

List the private IP addresses leased to a virtual machine

### Synopsis

One entry per lease, across all of the VM's network interfaces. The id is the leaseId that privateLeaseId on POST /v1/vms/{id}/public-ips takes. Private addresses cannot be released through the API. Cursor-paginated: follow nextCursor via `?cursor=`.

```
ca vms private-ips list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
      --vm string       the vm id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms private-ips](ca_vms_private-ips.md)	 - Manage a VM's private IP addresses

