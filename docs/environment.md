# Environment variables

## `ca`

| Variable | Effect |
|---|---|
| `CLOUDAFRICA_TOKEN` | A user API token. Overrides any profile; see [Scripts and CI](auth.md#scripts-and-ci). |
| `CLOUDAFRICA_ELEVATION` | The pairing secret `ca auth elevate` prints, sent with every request so a token may use protected actions. |
| `CLOUDAFRICA_PROFILE` | The profile to use when `--profile` is not given (default `default`). |
| `CLOUDAFRICA_ENDPOINT` | Another API endpoint; https, or http to a loopback address. A profile works only against the endpoint it signed in to. |

## The install scripts

| Variable | Effect |
|---|---|
| `CA_VERSION` | The release tag to install, such as `v0.7.0`; default the latest. Rerun with it to go back to an earlier release. |
| `CA_INSTALL_DIR` | Where to install. `install.sh` defaults to `/usr/local/bin` when writable, else `~/.local/bin`; `install.ps1` to `%LOCALAPPDATA%\Programs\cloudafrica`, which it adds to your user `PATH`. |
