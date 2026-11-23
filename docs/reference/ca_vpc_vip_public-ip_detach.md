## ca vpc vip public-ip detach

Take a VIP's public IP off

### Synopsis

Requires protected:network. Returns the address to the pool. Synchronous: returns 200 with the VIP; a VIP with no public IP answers 200 unchanged. Refuses with 409 while firewall rules on the VIP remain (remove them with /v1/vpcs/{id}/vips/{vipId}/firewall-rules first), and for a VIP the platform manages.

```
ca vpc vip public-ip detach [flags]
```

### Options

```
  -h, --help         help for detach
      --vip string   the vip id or name (required)
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

* [ca vpc vip public-ip](ca_vpc_vip_public-ip.md)	 - Attach and detach a virtual IP's public address

