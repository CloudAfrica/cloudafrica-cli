## ca placement-groups members add

Add a VM to a placement group

### Synopsis

Constrains where the VM is placed next; it does not move a running VM. Synchronous: returns 200 with the group as it stands. Idempotent, which is why it is a PUT: re-adding a VM that is already in THIS group answers 200 with the group unchanged, so a client whose first response was lost can safely resend. Refuses with 409 when the VM belongs to a DIFFERENT placement group, or when this group is at its member-count cap.

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

