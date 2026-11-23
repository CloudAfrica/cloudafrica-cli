## ca vpc reserved-ip delete

Release a reserved public IP

### Synopsis

Requires protected:network. Returns the address to the pool. Synchronous: returns 204. Refuses with 409 while a VM or a VIP holds the address; detach it first. A retry after a lost answer is a 404, the address no longer being the account's.

```
ca vpc reserved-ip delete PUBLIC_IP_ID [flags]
```

### Options

```
  -h, --help         help for delete
      --vpc string   the vpc id or name (required)
  -y, --yes          do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpc reserved-ip](ca_vpc_reserved-ip.md)	 - Manage a VPC's reserved public IP addresses

