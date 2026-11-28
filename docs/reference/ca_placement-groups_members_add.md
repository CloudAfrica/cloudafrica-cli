## ca placement-groups members add

Add a VM to a placement group

### Synopsis

Puts the VM in the group. That constrains where it is placed next; a running VM does not move. Adding a VM already in this group changes nothing. A VM in another group is refused, and so is a group at its member limit. Prints the group.

```
ca placement-groups members add VM [flags]
```

### Options

```
  -h, --help                     help for add
      --placement-group string   the placement-group id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca placement-groups members](ca_placement-groups_members.md)	 - Manage a placement group's members

