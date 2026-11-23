## ca support ticket create

Raise a support ticket

### Synopsis

Raises a ticket for the account and answers with the ticket summary; the conversation is read by getting the ticket. An account or service token must name the member it acts for (requesterEmail on create, authorEmail on reply); a personal token must not. Requires an idempotency key: a retry under the same key returns the first call's ticket rather than raising a second.

```
ca support ticket create [flags]
```

### Examples

```
  ca support ticket create --subject "Backup failed" --description "The nightly backup of web-01 failed"
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

* [ca support ticket](ca_support_ticket.md)	 - Manage support tickets

