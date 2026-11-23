## ca placement-group delete

Delete a placement group

### Synopsis

Releases the group; its members stay where they are, since a group only ever constrained future placement.

```
ca placement-group delete PLACEMENT_GROUP [flags]
```

### Options

```
  -h, --help   help for delete
  -y, --yes    do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca placement-group](ca_placement-group.md)	 - Manage placement groups

