# Networking

VPCs and what belongs to them: reserved public IPs, virtual IPs (VIPs) and their firewall rules. A VM's
own interfaces, IPs and firewall rules are under [Compute](compute.md). Every command is in the
[reference](reference/ca.md).

## VPCs

```sh
ca vpc list
ca vpc get web
ca vpc create --name web
ca vpc update web ...
ca vpc delete web
ca vpc firewall-rule list --vpc web
```

## Reserved public IPs

```sh
ca vpc reserved-ip list|create|delete --vpc web
```

`ca vm create --reserved-ip` builds a VM onto a reserved IP.

## Virtual IPs

```sh
ca vpc vip create|delete --vpc web
ca vpc vip public-ip attach|detach --vpc web --vip lb1
ca vpc vip firewall-rule list|create|delete|set-description --vpc web --vip lb1
ca vm vip add|remove lb1 --vm web-01  # membership
ca vip list                            # every VIP in the account
```

## Every address

```sh
ca ip-address list
```
