# Networking

VPCs and what belongs to them: reserved public IPs, virtual IPs (VIPs) and their firewall rules. A VM's
own interfaces, IPs and firewall rules are under [Compute](compute.md). Every command is in the
[reference](reference/ca.md).

Some commands, `vpcs delete` among them, are [protected actions](auth.md#protected-actions).

## VPCs

`ca vpcs list` shows your VPCs and `ca vpcs get` one of them. `ca vpcs create` makes a VPC, `ca vpcs update` renames it and `ca vpcs delete` removes it. `ca vpcs firewall-rules list` shows the firewall rules that apply to it.

For example, creating a VPC:

```sh
ca vpcs create --name web
```

## Internal DNS

Internal DNS is not yet generally available: these commands work only on an account that has it enabled, and
otherwise the API answers 403 `feature-not-enabled`.

`ca vpcs internal-dns get` shows the state, the resolver addresses and the draining time. Add `-o json` to see the records, clashes and skipped VMs too. `ca vpcs internal-dns set` turns it on with `--enabled=true` and off with `--enabled=false`.

For example, turning it on:

```sh
ca vpcs internal-dns set --vpc web --enabled=true
```

`get` prints the state (`off`, `setting_up`, `active`, `not_serving` or `winding_down`) and the
resolver addresses guests are given; the names the VPC answers are in the `-o json` output.
`set` takes `--enabled=true` or `--enabled=false` (a bare `--enabled` is true), and repeating
either changes nothing. When names clash, or VMs or virtual IPs are left out of the zone, `get` and
`set` say so on stderr. Turning it off is a drain, so `winding_down` is a normal answer, not a failure: run `get`
again to see the state settle.

## Reserved public IPs

`ca vpcs reserved-ips list` shows the reserved public IPs, `ca vpcs reserved-ips create` reserves one and `ca vpcs reserved-ips delete` releases one. Each takes `--vpc`.

For example, listing a VPC's reserved IPs:

```sh
ca vpcs reserved-ips list --vpc web
```

`ca vms create --reserved-ip` builds a VM onto a reserved IP.

## Virtual IPs

- `ca vips list` shows every VIP in the account.
- `ca vips create` reserves a VIP in a VPC and `ca vips delete` releases it. Both take `--vpc`.
- `ca vips public-ip attach` gives a VIP a public IP and `ca vips public-ip detach` takes it off. Both take `--vpc` and `--vip`.
- `ca vips firewall-rules list` shows a VIP's firewall rules; `ca vips firewall-rules create`, `ca vips firewall-rules update` and `ca vips firewall-rules delete` change them. All take `--vpc` and `--vip`.
- `ca vms vips add` makes a VM a member of a VIP and `ca vms vips remove` takes it out.

For example, the firewall rules of one VIP:

```sh
ca vips firewall-rules list --vpc web --vip lb1
```

## Every address

`ca ips list` shows every IP address the account holds.

```sh
ca ips list
```
