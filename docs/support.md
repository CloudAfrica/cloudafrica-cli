# Support

`ca support tickets list` shows the account's tickets and `ca support tickets get` one ticket with its conversation. `ca support tickets create` raises a ticket, `ca support tickets reply` adds a reply and `ca support tickets resolve` resolves it.

A ticket is named by its reference number. Every flag is in the [reference](reference/ca_support_tickets.md).

For example, raising a ticket:

```sh
ca support tickets create --subject "Backup failed" --description "The nightly backup of web-01 failed"
```
