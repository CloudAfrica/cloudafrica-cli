## ca auth login

Sign in through the browser

### Synopsis

Sign in through the browser and store the session in a profile. With --device, or by default over SSH and on Linux or BSD without a display, ca prints a URL and a code to use in a browser on any machine; --no-device keeps the browser on this one. One profile holds one account: to use several accounts, sign in once per account with --profile and name the account with --account. Signing in again to a profile stays in the account it is signed in to. Approving a sign-in that holds a protected: scope also allows protected actions such as vms delete, for the account's elevation window; choose a Write level or lower on the consent page to leave them off.

```
ca auth login [flags]
```

### Examples

```
  ca auth login
  ca auth login --profile globex --account <account-id>
```

### Options

```
      --account string      sign in to this account id (default: the account this profile is already signed in to)
      --device              sign in with a code, using a browser on any machine
  -h, --help                help for login
      --no-device           sign in through a browser on this machine
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

