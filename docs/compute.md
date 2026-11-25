# Compute

Virtual machines and what attaches to them, their backups, placement groups, and the catalogues a VM
is created from. Every command is in the [reference](reference/ca.md).

Some of these commands, deletes among them, are protected actions: run `ca auth elevate` before them. A
protected command run without it exits 5 and prints the remedy. See [Protected actions](auth.md#protected-actions),
or [Scripts and CI](auth.md#scripts-and-ci) when you sign in with a token.

## Before you create a VM

```sh
ca datacenters list
ca datacenters feasibility --datacenter jnb --vcpu 4 --ram-gib 8 --boot-disk-gib 40
ca os-images list
ca ssh-keys create --name laptop --public-key "$(cat ~/.ssh/id_ed25519.pub)"
ca ssh-keys list|update|delete
```

A VM joins a VPC; see [Networking](networking.md).

## Virtual machines

```sh
ca vms list
ca vms get web-01
ca vms create --name web-01 --datacenter <dc-slug> --image <image-id> --vpc main \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --accept-terms
ca vms start|shutdown|reboot|force-stop web-01 [--no-wait] [--timeout 10m]
ca vms resize web-01 --vcpu 4 --ram-gib 8
ca vms apply-pending-restart web-01
ca vms rename web-01 --name db-01
ca vms delete web-01   # the VM must be stopped
```

`vms create` takes flags only, not `--data`. vCPU and RAM are even numbers; `--public-ip` attaches a public
IP and `--extra-disk-gib` adds a data disk (repeatable, up to 8).

## Disks, network and access

```sh
ca vms disks list --vm web-01
ca vms add-disk web-01 ...
ca vms nics list|attach|detach --vm web-01
ca vms public-ips list|attach|detach --vm web-01
ca vms private-ips list|create --vm web-01
ca vms ssh-keys list|attach|detach --vm web-01
ca vms vips add|remove lb1 --vm web-01
ca vms firewall-rules list|create|update|delete --vm web-01
```

## Backups

The `vms backups` commands name a backup by its start time, RFC 3339 or epoch seconds. `vms restore` takes
epoch seconds only.

```sh
ca vms backups list|create --vm web-01
ca vms backups delete <started-at> --vm web-01
ca vms backups set-protection <started-at> --vm web-01 --protected
ca vms backups restore-sites list --vm web-01 --backup <started-at>
ca vms backups restore-to-new-vm <started-at> --vm web-01 ...
ca vms restore web-01 --started-at <epoch-seconds>
ca vms backup-policy get|set|delete|preview-rotation --vm web-01
ca backup-policies list     # every VM's policy
ca backup-windows list     # the backup window catalogue
ca account backup-retention
```

## Placement groups

```sh
ca placement-groups list|get|create|delete
ca placement-groups members add|remove web-01 --placement-group spread
```
