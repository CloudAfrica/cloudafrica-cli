## ca support tickets create

Raise a support ticket

### Synopsis

Raises a ticket and prints its summary; `ca support tickets get` shows the conversation. An account or service token must name the member it acts for with --requester-email; a personal token must not.

```
ca support tickets create [flags]
```

### Examples

```
  ca support tickets create --subject "Backup failed" --description "The nightly backup of web-01 failed"
```

### Options

```
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
      --description string       What you need help with. (required)
  -h, --help                     help for create
      --requester-email string   Required for an account or service token, which raises the ticket on behalf of the member with this email; refused for a personal token.
      --resource-id string       Optional id of a resource in this account the ticket is about.
      --subject string           The ticket's subject line. (required)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca support tickets](ca_support_tickets.md)	 - Manage support tickets

