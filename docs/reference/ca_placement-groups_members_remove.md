## ca placement-groups members remove

Remove a VM from a placement group

### Synopsis

Takes the VM out of the group. The VM stays where it is; the constraint no longer applies to it. Prints the group.

```
ca placement-groups members remove VM [flags]
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

* [ca placement-groups members](ca_placement-groups_members.md)	 - Manage a placement group's members

