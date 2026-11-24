## ca api

Make an authenticated API call

### Synopsis

Send METHOD PATH (e.g. GET /v1/vms) to the API with the selected credential and print the response body. --data sends a JSON body: a literal, @file, or @- for stdin. Writes get a fresh Idempotency-Key, except a call that mints a URL or token.

```
ca api METHOD PATH [flags]
```

### Options

```
  -d, --data string   JSON request body: a literal, @file or @- (stdin)
  -h, --help          help for api
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca](ca.md)	 - The CloudAfrica command-line interface

