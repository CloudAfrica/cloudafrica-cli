## ca auth elevate

Allow protected actions (such as vms delete) for this sign-in or API token

### Synopsis

Allow protected actions for the selected profile's sign-in. A sign-in that is already paired is left alone; otherwise it signs in again with the profile's scopes, because approving a sign-in that holds a protected: scope on the consent page pairs it for the account's elevation window, and then checks that the sign-in is paired.

With CLOUDAFRICA_TOKEN set, it pairs that API token instead: it opens a page in the portal where you step up and confirm the code shown here, waits up to 5 minutes for the confirmation, and prints the token's secret, alone, on stdout; the pairing lasts for the account's elevation window. A script sets it with

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

