# Signing in and protected actions

## Signing in

```sh
ca auth login     # opens the browser, or prints a code on a machine without one
ca auth login --device     # always use a code
ca auth login --scope write:billing   # also request a scope login does not ask for by default
ca auth login --profile globex --account <account-id>   # sign in to a specific account
ca auth status
ca auth list
ca auth logout
```

Tokens are stored per profile in the user config directory (`~/.config/cloudafrica/<profile>.json`
on Linux), mode 0600. Pick a profile with `--profile`, else `CLOUDAFRICA_PROFILE`, else `default`.

### Signing in from another machine

With `--device`, `ca` prints a portal URL and a code. Open the URL in a browser on any machine, enter
the code, and approve; `ca` waits until you do, for up to the 10 minutes the code lasts. `ca` opens no
browser itself.

`ca` picks this flow without the flag in an SSH session (`SSH_CONNECTION` or `SSH_TTY` set), and on
Linux and BSD when neither `DISPLAY` nor `WAYLAND_DISPLAY` is set. `--no-device` keeps the browser on
this machine instead. Against a server without the device flow, `--device` fails, and the automatic
choice falls back to the browser with a note.

Only continue if you started the sign-in yourself, on a machine you control: approving someone else's
code signs them in as you. If the server cancels the sign-in because the code was used twice, run
`ca auth login` again.

The consent page never asks which account: `ca` names it. `--account` sets it for a first sign-in, and a
later `ca auth login` on the same profile and endpoint stays in the account that profile is signed in to.
With no account named, the gateway picks one or refuses (see the portal's Connected apps); name the
account with `--account` to be sure. If the session comes back for another account, `ca` saves nothing
and exits 4. One profile holds one account, so use `--profile` per account.

Scopes added with `--scope` are kept, and requested again by every later `ca auth login` on that
profile. A command that needs a scope the sign-in lacks says which, and how to add it.

## Protected actions

Signing in grants scopes but never allows protected actions, such as `vms delete`. `ca auth elevate` opens a portal page where you step up and confirm the code it
prints; the sign-in may then use protected actions for the account's elevation window. Connected apps →
"Allow protected actions" in the portal does the same. A sign-in made before `ca auth login` requested
a `protected:` scope must sign in again first.

```sh
ca auth elevate
```

## Scripts and CI

A script uses `CLOUDAFRICA_TOKEN`, a user API token, which overrides any profile. For protected actions
the token must hold the protected scope, and a person pairs it before the run:

```sh
export CLOUDAFRICA_TOKEN=cak_...
CLOUDAFRICA_ELEVATION=$(ca auth elevate) || exit
export CLOUDAFRICA_ELEVATION
./teardown.sh
```

Keep the assignment on its own line. `eval "$(ca auth elevate)"` and
`export CLOUDAFRICA_ELEVATION=$(ca auth elevate)` both return 0 when `ca auth elevate` fails, so the
`|| exit` would never fire. In PowerShell, check the exit code yourself:

```powershell
$env:CLOUDAFRICA_ELEVATION = ca auth elevate
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
```

In CI, store the printed value as a masked variable alongside the token. Running `ca auth elevate`
without capturing it prints the secret to the terminal, where it stays in the scrollback.

`ca auth elevate` opens a portal page where the person steps up and confirms the code; it then prints
only the pairing secret on stdout, which `ca` sends with every request while `CLOUDAFRICA_ELEVATION`
is set. The pairing lasts for the account's elevation window and only while that person's portal
session does. Run `ca auth elevate` again with the variable still set to pair the same value. The
secret is useless without the token, but treat it as one: do not log it or commit it.

## Another endpoint

`CLOUDAFRICA_ENDPOINT` selects another API endpoint, which must be https (plain http only to a loopback
address); a profile can only be used against the endpoint it signed in to.
