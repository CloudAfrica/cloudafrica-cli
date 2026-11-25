## ca vpcs reserved-ips create

Reserve a public IP

### Synopsis

Requires write:network. Takes a free public address from the pool and holds it for the account until it is released; the address is not a parameter. Synchronous: returns 200 with the reservation. Refuses with 409 when no address is free, and with 422 limit-reached at the account's public-IP cap (the PUBLIC_IP key), which a reservation counts against whether or not it is attached. A retry with the same Idempotency-Key replays a 200 already returned; after a timeout or 5xx, or under a new key, it may reserve a second address, so list GET /v1/vpcs/{id}/reserved-ips first.

```
ca vpcs reserved-ips create [flags]
```

### Options

```
  -h, --help         help for create
      --vpc string   the vpc id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vpcs reserved-ips](ca_vpcs_reserved-ips.md)	 - Manage a VPC's reserved public IP addresses

