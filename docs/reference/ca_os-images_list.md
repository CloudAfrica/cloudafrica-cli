## ca os-images list

List the images a virtual machine can be created from

### Synopsis

Every image you can create from. `id` is the `osImageId` that POST /v1/vms takes. `requiresSshKey` says whether the create must carry at least one SSH key (true) or refuses keys (false, for Windows images).

```
ca os-images list [flags]
```

### Options

```
      --cursor string   with --limit, the page to fetch (the previous page's nextCursor)
  -h, --help            help for list
      --limit int       fetch one page of at most this many rows (the route's cap applies) and print it as returned
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca os-images](ca_os-images.md)	 - List OS images

