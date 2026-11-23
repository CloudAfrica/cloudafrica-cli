# Compute

Virtual machines and what attaches to them, their backups, placement groups, and the catalogues a VM
is created from. Every command is in the [reference](reference/ca.md).

Some of these commands, deletes among them, are protected actions: run `ca auth elevate` before them. A
protected command run without it exits 5 and prints the remedy. See [Protected actions](auth.md#protected-actions),
or [Scripts and CI](auth.md#scripts-and-ci) when you sign in with a token.

## Before you create a VM

```sh
ca datacenter list
ca datacenter feasibility --datacenter jnb --vcpu 4 --ram-gib 8 --boot-disk-gib 40
ca os-image list
ca ssh-key create --name laptop --public-key "$(cat ~/.ssh/id_ed25519.pub)"
ca ssh-key list|update|delete
```

A VM joins a VPC; see [Networking](networking.md).

## Virtual machines

```sh
ca vm list
ca vm get web-01
ca vm create --name web-01 --datacenter <dc-slug> --image <image-id> --vpc main \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --accept-terms
ca vm start|shutdown|reboot|force-stop web-01 [--no-wait] [--timeout 10m]
ca vm resize web-01 --vcpu 4 --ram-gib 8
ca vm apply-pending-restart web-01
ca vm rename web-01 --name db-01
ca vm delete web-01   # the VM must be stopped
```

`vm create` takes flags only, not `--data`. vCPU and RAM are even numbers; `--public-ip` attaches a public
IP and `--extra-disk-gib` adds a data disk (repeatable, up to 8).

## Disks, network and access

```sh
ca vm disk list --vm web-01
ca vm add-disk web-01 ...
ca vm nic list|attach|detach --vm web-01
ca vm public-ip list|attach|detach --vm web-01
ca vm private-ip list|create --vm web-01
ca vm ssh-key list|attach|detach --vm web-01
ca vm vip add|remove lb1 --vm web-01
ca vm firewall-rule list|create|update|delete --vm web-01
```

## Backups

The `vm backup` commands name a backup by its start time, RFC 3339 or epoch seconds. `vm restore` takes
epoch seconds only.

```sh
ca vm backup list|create --vm web-01
ca vm backup delete <started-at> --vm web-01
ca vm backup set-protection <started-at> --vm web-01 --protected
ca vm backup restore-site list --vm web-01 --backup <started-at>
ca vm backup restore-to-new-vm <started-at> --vm web-01 ...
ca vm restore web-01 --started-at <epoch-seconds>
ca vm backup-policy get|set|delete|preview-rotation --vm web-01
ca backup-policy list     # every VM's policy
ca backup-window list     # the backup window catalogue
ca account backup-retention
```

## Placement groups

```sh
ca placement-group list|get|create|delete
ca placement-group member add|remove web-01 --placement-group spread
```
