# Compute

Virtual machines and what attaches to them, their backups, placement groups, and the catalogues a VM
is created from. Every command is in the [reference](reference/ca.md).

Some commands, `vms delete` among them, are [protected actions](auth.md#protected-actions).

## Before you create a VM

`ca datacenters list` shows the datacenters, and `ca datacenters feasibility` checks whether one can take a VM of a given size. `ca os-images list` shows the images a VM can be created from.

`ca ssh-keys create` registers a public key for VMs to use. `ca ssh-keys list` shows your keys, `ca ssh-keys update` renames one and `ca ssh-keys delete` removes one.

For example, registering the key of this machine:

```sh
ca ssh-keys create --name laptop --public-key "$(cat ~/.ssh/id_ed25519.pub)"
```

A VM joins a VPC; see [Networking](networking.md).

## Virtual machines

`ca vms list` shows your VMs and `ca vms get` one of them. `ca vms create` builds a VM and waits until it is built.

`ca vms start`, `ca vms shutdown`, `ca vms reboot` and `ca vms force-stop` change its power state. Each waits for the operation; `--no-wait` returns at once and `--timeout 10m` sets how long to wait. `ca vms resize` changes its vCPU and RAM, `ca vms apply-pending-restart` restarts it to apply a pending configuration change, and `ca vms rename` renames it. `ca vms delete` deletes it, and the VM must be stopped.

`vms create` takes flags only, not `--data`. vCPU and RAM are even numbers; `--public-ip` attaches a public
IP and `--extra-disk-gib` adds a data disk (repeatable, up to 8).

For example, creating a VM:

```sh
ca vms create --name web-01 --datacenter <dc-slug> --image <image-id> --vpc main \
  --vcpu 2 --ram-gib 4 --boot-disk-gib 40 --ssh-key laptop --accept-terms
```

## Disks, network and access

These commands take `--vm` to name the VM.

- `ca vms disks list` shows its disks and `ca vms disks create` attaches a data disk.
- `ca vms nics list` shows its network interfaces; `ca vms nics attach` and `ca vms nics detach` add and remove one.
- `ca vms public-ips list` shows its public IPs; `ca vms public-ips attach` and `ca vms public-ips detach` add and remove one.
- `ca vms private-ips list` shows its private IPs and `ca vms private-ips create` adds one.
- `ca vms ssh-keys list` shows the SSH keys it can use; `ca vms ssh-keys attach` and `ca vms ssh-keys detach` add and remove one.
- `ca vms vips add` makes it a member of a virtual IP and `ca vms vips remove` takes it out.
- `ca vms firewall-rules list` shows its firewall rules; `ca vms firewall-rules create`, `ca vms firewall-rules update` and `ca vms firewall-rules delete` change them.

For example, the disks of one VM:

```sh
ca vms disks list --vm web-01
```

## Backups

The `vms backups` commands name a backup by its start time, RFC 3339 or epoch seconds. `vms restore` takes
epoch seconds only.

- `ca vms backups list` shows a VM's backups and `ca vms backups create` takes one now.
- `ca vms backups delete` permanently deletes one backup, and `ca vms backups set-protection` protects or releases one.
- `ca vms backups restore-sites list` shows the datacenters a backup can be restored to as a new VM, and `ca vms backups restore-to-new-vm` does the restore.
- `ca vms restore` restores the VM itself from one of its backups.
- `ca vms backup-policy get` shows a VM's backup policy; `ca vms backup-policy set` and `ca vms backup-policy delete` change it, and `ca vms backup-policy preview-rotation` shows what a retention set would delete.
- `ca backup-policies list` shows the backup policy of every live VM, `ca backup-windows list` the backup window catalogue, and `ca account backup-retention` how long a deleted VM's newest backup is kept.

For example, taking a backup now:

```sh
ca vms backups create --vm web-01
```

## Placement groups

`ca placement-groups list` shows the placement groups and `ca placement-groups get` one of them. `ca placement-groups create` makes one and `ca placement-groups delete` removes it. `ca placement-groups members add` puts a VM in a group and `ca placement-groups members remove` takes it out.

For example, adding a VM to the `spread` group:

```sh
ca placement-groups members add web-01 --placement-group spread
```
