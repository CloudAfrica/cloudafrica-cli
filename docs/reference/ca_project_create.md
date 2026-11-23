## ca project create

Create a project

### Synopsis

Creates an empty project in the token's account. Only an account admin's personal token can create a project: every account-service token is refused with 403. The slug is optional: 2 to 5 letters or digits, starting with a letter, upper-cased on receipt (prd is stored as PRD), DEF is reserved, derived from the name when omitted (Production gives PRD), and fixed once created. 409 if the account already has a project with this name (ignoring case) or with this slug. Requires an idempotency key: a retry under the same key returns the first call's project rather than creating a second.

```
ca project create [flags]
```

### Examples

```
  ca project create --name Production --slug PRD
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

* [ca project](ca_project.md)	 - Manage projects

