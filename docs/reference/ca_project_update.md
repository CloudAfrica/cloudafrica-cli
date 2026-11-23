## ca project update

Rename a project

### Synopsis

Changes the display name only. Answers with the renamed project. 409 if the account already has a project with this name, ignoring case. Needs an account admin's personal token or an account-service token with service_admin; any other credential is refused with 403.

```
ca project update PROJECT [flags]
```

### Examples

```
  ca project update prod --name Production
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for update
      --name string   The project name, at most 120 characters. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca project](ca_project.md)	 - Manage projects

