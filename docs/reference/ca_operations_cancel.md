## ca operations cancel

Request cancellation of an operation

### Synopsis

Asks for the operation to be cancelled. It reaches its final state later.

Waits for the operation to finish; --no-wait prints it and returns.

```
ca operations cancel OPERATION_ID [flags]
```

### Examples

```
  ca operations cancel <operation-id>
```

### Options

```
  -h, --help               help for cancel
      --no-wait            print the operation and return without waiting for it
      --timeout duration   how long to wait for the operation (default 10m0s)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca operations](ca_operations.md)	 - Inspect and cancel asynchronous operations

