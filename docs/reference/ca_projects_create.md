## ca projects create

Create a project

### Synopsis

Creates an empty project. Only an account admin's own sign-in or personal token can; an account service token is refused. --slug is 2 to 5 letters or digits starting with a letter, upper-cased (prd becomes PRD); DEF is reserved, it is derived from the name when you leave it out (Production gives PRD), and it cannot be changed. A name (ignoring case) or slug another project has is refused.

```
ca projects create [flags]
```

### Examples

```
  ca projects create --name Production --slug PRD
```

### Options

```
  -d, --data string   the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help          help for create
      --name string   The project name, at most 120 characters. (required)
      --slug string   Optional short handle: 2 to 5 letters or digits, starting with a letter, upper-cased on receipt; DEF is reserved. Derived from the name when omitted (Production gives PRD). Cannot be changed later.
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca projects](ca_projects.md)	 - Manage projects

