## ca support ticket reply

Reply on a support ticket

### Synopsis

Posts a reply on the ticket and answers with the ticket summary. An account or service token must name the member it acts for (requesterEmail on create, authorEmail on reply); a personal token must not.

```
ca support ticket reply SUPPORT_TICKET [flags]
```

### Examples

```
  ca support ticket reply 1234 --body "That fixed it"
```

### Options

```
      --author-email string   Required for an account or service token, which posts the reply on behalf of the member with this email; refused for a personal token.
      --body string           The reply text. (required)
  -d, --data string           the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                  help for reply
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca support ticket](ca_support_ticket.md)	 - Manage support tickets

