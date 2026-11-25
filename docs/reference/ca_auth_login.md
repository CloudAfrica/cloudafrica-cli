## ca auth login

Sign in through the browser

### Synopsis

Sign in through the browser and store the session in a profile. One profile holds one account: to use several accounts, sign in once per account with --profile and name the account with --account. Signing in again to a profile stays in the account it is signed in to. Signing in never allows protected actions such as vms delete; `ca auth elevate` does.

```
ca auth login [flags]
```

### Options

```
      --account string      sign in to this account id (default: the account this profile is already signed in to)
  -h, --help                help for login
      --scope stringArray   also request this scope, e.g. protected:billing (repeatable; kept for every later sign-in of this profile)
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca auth](ca_auth.md)	 - Sign in, sign out and list profiles

