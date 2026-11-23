# Account, team and billing

The account, its projects and members, billing, and the activity log. Every command is in the
[reference](reference/ca.md).

## Account

```sh
ca account get|limits|backup-retention
ca activity list --types vm,vpc
ca activity list --subject-type vm --subject-id <id>
```

`--types` and `--subject-type`/`--subject-id` do not mix.

## Projects

```sh
ca project list|get|create|update|delete
ca project resource list|move --project prod
```

## Members

```sh
ca member list
ca member create --email jo@example.com --role operator
ca member delete jo@example.com
ca member role set jo@example.com --role admin
ca member resend-invitation|cancel-invitation jo@example.com
```

A member is named by email. `create`, `delete` and `role set` need the `protected:team` scope:
`ca auth login --scope protected:team`, then `ca auth elevate` (see [Signing in](auth.md)).

## Billing

```sh
ca billing balance
ca billing accrual
ca billing details get|set
ca transaction list     # invoices and credit notes
```

`billing details set` needs `protected:billing`: `ca auth login --scope protected:billing`, then
`ca auth elevate`.
