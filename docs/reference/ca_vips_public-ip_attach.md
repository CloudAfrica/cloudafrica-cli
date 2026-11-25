## ca vips public-ip attach

Give a VIP a public IP

### Synopsis

Requires protected:network. Must carry an Idempotency-Key header. Takes a free public address from the pool; the address is not a parameter. Synchronous: returns 200 with the VIP. Refuses with 409 when the VIP already has a public IP or no address is free, and with 422 limit-reached at the account's public-IP cap (the PUBLIC_IP key). If the answer to an attach is lost, a retry with the same Idempotency-Key replays the 200; under a new key it answers 409, so list GET /v1/vips and treat a VIP that now has a publicAddress as attached.

```
ca vips public-ip attach [flags]
```

### Options

```
  -h, --help         help for attach
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

* [ca vips public-ip](ca_vips_public-ip.md)	 - Attach and detach a virtual IP's public address

