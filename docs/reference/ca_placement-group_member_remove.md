## ca placement-group member remove

Remove a VM from a placement group

### Synopsis

The VM stays where it is; the constraint no longer applies to it. Synchronous: returns 200 with the group as it stands.

```
ca placement-group member remove VM [flags]
```

### Options

```
  -h, --help                     help for remove
      --placement-group string   the placement-group id or name (required)
  -y, --yes                      do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca placement-group member](ca_placement-group_member.md)	 - Manage a placement group's members

