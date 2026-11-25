## ca auth

Sign in, sign out and list profiles

```
ca auth [flags]
```

### Options

```
  -h, --help   help for auth
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca](ca.md)	 - The CloudAfrica command-line interface
* [ca auth elevate](ca_auth_elevate.md)	 - Allow protected actions (such as vms delete) for this sign-in or API token
* [ca auth list](ca_auth_list.md)	 - List profiles
* [ca auth login](ca_auth_login.md)	 - Sign in through the browser
* [ca auth logout](ca_auth_logout.md)	 - Revoke the selected profile's session and delete the profile
* [ca auth status](ca_auth_status.md)	 - Show the selected profile's sign-in

