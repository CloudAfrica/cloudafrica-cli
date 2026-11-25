## ca projects resources move

Move a resource into a project

### Synopsis

Files the resource in this project, moving it out of whichever project it is in now; repeating the call is harmless. Answers with the destination project. Within one account only, and not every kind of resource can be moved: those authorized through a parent, such as public IPs and firewall rules, are refused. Needs an account admin's personal token or an account-service token with service_admin; any other credential is refused with 403.

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

