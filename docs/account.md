# Account, team and billing

The account, its projects and members, billing, and the activity log. Every command is in the
[reference](reference/ca.md).

## Account

`ca account get` shows the account you are signed in to, `ca account limits` its limits, and `ca account backup-retention` how long a deleted VM's newest backup is kept.

`ca activity list` shows the activity log. `--types` narrows it to kinds of resource, such as `vm,vpc`; `--subject-type` with `--subject-id` narrows it to one resource. `--types` and the subject pair do not mix.

For example, the activity on one VM:

```sh
ca activity list --subject-type vm --subject-id <id>
```

## Projects

`ca projects list` shows the projects and `ca projects get` one of them. `ca projects create` makes a project, `ca projects update` renames it and `ca projects delete` removes it.

`ca projects resources list` shows what a project holds, and `ca projects resources move` moves a resource into a project, named with `--project`.

For example, moving a VM into the `prod` project:

```sh
ca projects resources move "$(ca vms get web-01 -q)" --project prod
```

## Members

`ca members list` shows the team. `ca members create` invites someone by email, with `--role`, and `ca members delete` removes a member. `ca members role set` changes a member's role. `ca members resend-invitation` and `ca members cancel-invitation` act on a pending invitation.

A member is named by email. `create`, `delete` and `role set` need the `protected:team` scope:
`ca auth login --scope protected:team` (see [Signing in](auth.md)).

For example, inviting an operator:

```sh
ca members create --email jo@example.com --role operator
```

## Billing

`ca billing balance` shows the outstanding balance and `ca billing accrual` the month-to-date accrual. `ca billing details get` shows the billing details and `ca billing details set` changes them. `ca transactions list` shows invoices and credit notes.

`billing details set` needs `protected:billing`: `ca auth login --scope protected:billing`.

For example, changing the billing email:

```sh
ca billing details set --billing-email accounts@example.com
```
