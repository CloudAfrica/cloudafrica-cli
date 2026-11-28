# Signing in and protected actions

## Signing in

`ca auth login` signs in. It opens the browser, or prints a code on a machine without one. Three flags change it:

- `--device` always uses a code.
- `--scope` also requests a scope that login does not ask for by default, such as `write:billing`.
- `--profile` and `--account` sign in to a specific account, under its own profile.

`ca auth status` shows the selected profile's sign-in, `ca auth list` lists your profiles, and `ca auth logout` revokes the selected profile's session and deletes the profile.

For example, signing in to a specific account:

```sh
ca auth login --profile globex --account <account-id>
```

Tokens are stored per profile in the user config directory (`~/.config/cloudafrica/<profile>.json`
on Linux), mode 0600. Pick a profile with `--profile`, else `CLOUDAFRICA_PROFILE`, else `default`.

The consent page never asks which account: `ca` names it. `--account` sets it for a first sign-in, and a
later `ca auth login` on the same profile and endpoint stays in the account that profile is signed in to.
With no account named, the gateway picks one or refuses (see the portal's Connected apps); name the
account with `--account` to be sure. If the session comes back for another account, `ca` saves nothing
and exits 4. One profile holds one account, so use `--profile` per account.

Scopes added with `--scope` are kept, and requested again by every later `ca auth login` on that
profile. A command that needs a scope the sign-in lacks says which, and how to add it.

### Signing in from another machine

With `--device`, `ca` prints a portal URL and a code. Open the URL in a browser on any machine, enter
the code, and approve; `ca` waits until you do or the code expires. `ca` opens no browser itself.

`ca` picks this flow without the flag in an SSH session (`SSH_CONNECTION` or `SSH_TTY` set), and on
Linux and BSD when neither `DISPLAY` nor `WAYLAND_DISPLAY` is set. `--no-device` keeps the browser on
this machine instead. Against a server without the device flow, `--device` fails, and the automatic
choice falls back to the browser with a note.

Only continue if you started the sign-in yourself, on a machine you control: approving someone else's
code signs them in as you. If the server cancels the sign-in because the code was used twice, run
`ca auth login` again.

## Protected actions

Approving a sign-in that asks for a `protected:` scope, on the consent page, also allows protected
actions such as `vms delete` for the account's elevation window; signing out of the portal does not end
it. If a protected command is refused as not
paired, `ca` offers to sign in again; on a script it exits 5. Connected apps → "Allow protected actions"
in the portal does the same without signing in again. Choose Write or lower on the consent page to leave
protected actions off.

`ca auth elevate` on a sign-in checks whether it is already paired and, if not, signs in again so that
approving a `protected:` scope pairs it, then checks. With `CLOUDAFRICA_TOKEN` it pairs the token instead, as below.

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
is set. The pairing lasts for the account's elevation window. Run `ca auth elevate` again with the variable still set to pair the same value. The
secret is useless without the token, but treat it as one: do not log it or commit it.

## Another endpoint

`CLOUDAFRICA_ENDPOINT` selects another API endpoint, which must be https (plain http only to a loopback
address); a profile can only be used against the endpoint it signed in to.
