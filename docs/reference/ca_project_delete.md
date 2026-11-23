## ca project delete

Delete a project

### Synopsis

Only an empty project that is not the account's default and not managed by the platform can be deleted; anything else answers 409, so move its resources out first. Only an account admin's personal token can delete a project: every account-service token is refused with 403.

```
ca project delete PROJECT [flags]
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

* [ca project](ca_project.md)	 - Manage projects

