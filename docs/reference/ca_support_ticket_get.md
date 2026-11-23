## ca support ticket get

Get a support ticket and its conversation

### Synopsis

The ticket, its original description and its visible conversation, oldest first. The tickets returned are those raised through the portal or the API for this account, plus, for a personal token, the token owner's own requests. Internal notes are never returned, and HTML bodies are sanitised.

```
ca support ticket get SUPPORT_TICKET [flags]
```

### Examples

```
  ca support ticket get 1234
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

* [ca support ticket](ca_support_ticket.md)	 - Manage support tickets

