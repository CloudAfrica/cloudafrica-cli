## ca projects update

Rename a project

### Synopsis

Renames the project. A name another project has, ignoring case, is refused. Needs an account admin's sign-in or personal token, or an account service token with service_admin.

```
ca projects update PROJECT [flags]
```

### Examples

```
  ca projects update prod --name Production
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

* [ca projects](ca_projects.md)	 - Manage projects

