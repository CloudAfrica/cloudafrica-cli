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

## Reserved public IPs

```sh
ca vpcs reserved-ips list|create|delete --vpc web
```

`ca vms create --reserved-ip` builds a VM onto a reserved IP.

## Virtual IPs

```sh
ca vpcs vips create|delete --vpc web
ca vpcs vips public-ip attach|detach --vpc web --vip lb1
ca vpcs vips firewall-rules list|create|delete|set-description --vpc web --vip lb1
ca vms vips add|remove lb1 --vm web-01  # membership
ca vips list                            # every VIP in the account
```

## Every address

```sh
ca ips list
```
