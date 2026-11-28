## ca projects delete

Delete a project

### Synopsis

Deletes an empty project. The default project and projects the platform manages cannot be deleted, and a project that holds resources is refused: move them out first. Only an account admin's own sign-in or personal token can delete; an account service token is refused.

```
ca projects delete PROJECT [flags]
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

* [ca projects](ca_projects.md)	 - Manage projects

