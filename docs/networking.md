# Networking

VPCs and what belongs to them: reserved public IPs, virtual IPs (VIPs) and their firewall rules. A VM's
own interfaces, IPs and firewall rules are under [Compute](compute.md). Every command is in the
[reference](reference/ca.md).

Some of these commands, deletes among them, are protected actions: run `ca auth elevate` before them. A
protected command run without it exits 5 and prints the remedy. See [Protected actions](auth.md#protected-actions),
or [Scripts and CI](auth.md#scripts-and-ci) when you sign in with a token.

## VPCs

```sh
ca vpcs list
ca vpcs get web
ca vpcs create --name web
ca vpcs update web ...
ca vpcs delete web
ca vpcs firewall-rules list --vpc web
```

## Internal DNS

```sh
ca vpcs internal-dns get --vpc web                    # state, resolver addresses, draining time
ca vpcs internal-dns get --vpc web -o json            # records, clashes and skipped VMs too
ca vpcs internal-dns set --vpc web --enabled=true
ca vpcs internal-dns set --vpc web --enabled=false
```

`get` prints the state (`off`, `setting_up`, `active`, `not_serving` or `winding_down`) and the
resolver addresses guests are given; the names the VPC answers are in the `-o json` output.
`set` takes `--enabled=true` or `--enabled=false` (a bare `--enabled` is true), and repeating
either changes nothing. When names clash or VMs are left out of the zone, `get` and `set` say so on
stderr. Turning it off is a drain, so `winding_down` is a normal answer, not a failure: run `get`
again to see the state settle.

## Reserved public IPs

```sh
ca vpcs reserved-ips list|create|delete --vpc web
```

`ca vms create --reserved-ip` builds a VM onto a reserved IP.

## Virtual IPs

```sh
ca vips list                            # every VIP in the account
ca vips create|delete --vpc web
ca vips public-ip attach|detach --vpc web --vip lb1
ca vips firewall-rules list|create|delete|update --vpc web --vip lb1
ca vms vips add|remove lb1 --vm web-01  # membership
```

## Every address

```sh
ca ips list
```
