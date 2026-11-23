## ca vm restore

Restore a virtual machine from one of its backups

### Synopsis

Requires protected:backup. Overwrites everything written since the named backup. `startedAt` is the whole-second value the backup list returns; a timestamp that matches no backup is 404, and one that matches more than one is 409. Asynchronous: returns 202 with the operation to poll at Location.

```
ca vm restore VM [flags]
```

### Options

```
  -d, --data string        the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help               help for restore
      --no-wait            print the operation and return without waiting for it
      --started-at int     The whole-second startedAt of the backup, from GET …/backups. (required)
      --timeout duration   how long to wait for the operation (default 10m0s)
  -y, --yes                do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca vm](ca_vm.md)	 - Manage virtual machines

