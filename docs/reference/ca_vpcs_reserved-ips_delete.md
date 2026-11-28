## ca vpcs reserved-ips delete

Release a reserved public IP

### Synopsis

Releases the reserved address back to the pool. An address a VM or VIP still holds is refused: detach it first. Needs protected:network.

```
ca vpcs reserved-ips delete PUBLIC_IP_ID [flags]
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

* [ca vpcs reserved-ips](ca_vpcs_reserved-ips.md)	 - Manage a VPC's reserved public IP addresses

