## ca auth elevate

Allow protected actions (such as vm delete) for this sign-in or API token

### Synopsis

Pair the selected profile's sign-in for protected actions. Opens a page in the portal where you step up and confirm the code shown here, then waits up to 5 minutes for the confirmation. The pairing lasts for the account's elevation window.

With CLOUDAFRICA_TOKEN set, it pairs that API token instead and prints its secret, alone, on stdout; that pairing also ends with the confirming person's portal session. A script sets it with

  CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit; export CLOUDAFRICA_ELEVATION

An existing CLOUDAFRICA_ELEVATION is paired again rather than replaced.

```
ca auth elevate [flags]
```

### Options

```
  -h, --help   help for elevate
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca auth](ca_auth.md)	 - Sign in, sign out and list profiles

