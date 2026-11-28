## ca vms backup-policy set

Set a virtual machine's backup policy

### Synopsis

Replaces the whole policy: a retention tier you leave out is dropped, not kept. Give --plan-id, or --cadence-kind with a retention, not both. A VM that another scheduler already backs up is refused.

Some fields of the request body have no flag; set them with --data.

```
ca vms backup-policy set [flags]
```

### Examples

```
  # A weekly backup on Sundays, overnight
  ca vms backup-policy set --vm web-01 --cadence-kind weekly --cadence-day 7 --window-preference overnight
```

### Options

```
      --cadence-day int            The day component: an ISO day-of-week (1-7) for weekly, 1-28 or -1 (last day of the month) for monthly, absent for daily.
      --cadence-kind string        daily, weekly or monthly.
  -d, --data string                the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                       help for set
      --paused                     Start the policy paused.
      --plan-id string             catalogue plan to copy; leave it out for a custom set
      --vm string                  the vm id or name (required)
      --window-preference string   overnight, morning, afternoon or evening.
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vms backup-policy](ca_vms_backup-policy.md)	 - Manage a VM's backup policy

