## ca projects resources move

Move a resource into a project

### Synopsis

Moves a resource into the project from whichever project holds it now; repeating it changes nothing. Prints the project. It works within one account only, and resources that belong to another, such as public IPs and firewall rules, cannot be moved. Needs an account admin's sign-in or personal token, or an account service token with service_admin.

```
ca projects resources move RESOURCE_ID [flags]
```

### Examples

```
  ca projects resources move "$(ca vms get web-01 -q)" --project prod
```

### Options

```
  -h, --help             help for move
      --project string   the project id or name (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca projects resources](ca_projects_resources.md)	 - Manage the resources in a project

