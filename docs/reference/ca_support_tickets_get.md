## ca support tickets get

Get a support ticket and its conversation

### Synopsis

Shows the ticket, its description and its visible conversation, oldest first. It covers tickets raised through the portal or the API for this account and, with a personal token, that person's own requests. Internal notes are never shown, and HTML is sanitised.

```
ca support tickets get SUPPORT_TICKET [flags]
```

### Examples

```
  ca support tickets get 1234
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca support tickets](ca_support_tickets.md)	 - Manage support tickets

