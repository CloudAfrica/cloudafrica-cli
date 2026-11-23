# Traps

Author misses the PR review caught. Check these before opening a PR.

- **A spec's list is a checklist.** When a spec enumerates what a doc or command must cover, tick
  each item against the diff; the README for token elevation shipped missing three of §8b's items.
- **The write of a command's deliverable checks its error.** If stdout carries the result a script
  captures (a secret, a Location), a failed `fmt.Fprint*` must return an error, or the command exits
  0 with nothing captured. Pin it with an `h.failStdout` test per command (missed again on #8).
  The response can lack the deliverable too: run each such command on a replay, an empty 200/201/204,
  `available:false` and with `-q`, and see what a script captures (`bucket-key create|reveal-secret`,
  #13: `-q` printed the key id, a replay and an empty 204 exited 0).
- **Say what `ca` does, not what the gateway does.** A comment or help line about server behaviour
  is a claim about another repo; state ca's own rule, or cite the gateway source.
- **A command's own precondition runs before anything that fails with other advice.** `ca k8s token`
  checked for CLOUDAFRICA_TOKEN inside `withSession`, so a shell with no token got "run `ca auth
  login`" from the session first. Run the command in an empty environment before trusting a guard.
- **When a remedy is wrong, enumerate every error path of the command.** The pre-check fixed the
  missing token; a 401 and an insufficient-scope refusal still printed the generic sign-in hint.
- **A rewrite asserts what it leaves alone and what it must not carry.** `rewriteKubeconfig` was
  tested only on the fields it set; nothing checked the CA data and contexts survived, and a static
  token beside `exec` survived too, against the README's "holds no credential" (Copilot, #8).
- **Every failure branch of a fetch gets a fake knob and a test.** Non-200, empty body and bad
  YAML on `ca k8s kubeconfig` had none, nor had the argv0-derived exe name.
- **Test every branch a flag-dependent path reaches.** A reused-secret path that is paired on the
  first response does not cover the browser/poll branch; set the poll count so both run.
- **Pin a length or range boundary with the off-by-one value.** A test value at the minimum passes
  when the minimum is lowered; add the value one below it as a refused case.
- **Run golangci-lint locally** (`go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run`);
  CI fails on staticcheck quick-fixes and forcetypeassert that `go vet` does not report (missed again
  on #11).
- **Every enum state gets a branch and a test.** A `== CURRENT` check with an `else` lumps the
  other four upgrade states together; switch over the generated consts and add a table row each.
  A state the API may add later ends a wait with an error, never a loop (#11).
- **`--wait` never exits 0 without waiting** unless the end state already holds. Each early return
  (no operation id, nothing to do) checks the flag.
- **A substring assertion must not match its negation.** `"ready"` is in `"not ready"`; assert the
  whole row, and include the negative case. A new column set pins its header line and every cell of
  a row filled in every column, so a swapped path fails; a prefix match leaves the tail unpinned
  (#12, then #13 again: `contains` on loose tokens passed with two columns swapped).
- **Read the SDK's description before naming a column.** `quotaBytes` is the priced usage, not a
  limit; the header QUOTA told a customer the wrong thing (#13). Open the field's description.
- **Trace every flag value in an example or help text through the executor.** Body fields are sent
  verbatim and only path params resolve names, so `--bucket-resource-id logs-7f3a` sent a name where
  the API wants an id, and the SDK's "id, not its name" help sat on a flag that does resolve names.
  Run `ca <cmd> --help` for each override; a backticked span in help becomes the pflag metavar (#13).
- **A remedy string is a claim about another command.** The reveal-secret hint said to delete the
  key; the API refuses to delete the master key, which is the one most likely revealed. Check the
  remedy against the API contract and test it on the case that triggers it (#13).
- **Copy no helper by hand.** k8s inlined vm's 202 handling and drifted; reuse or factor it.
- **A dependency bump checks what the dependency reports.** cloudafrica-go v0.2.0 kept `Version`
  at 0.1.0, so `ca` 0.2.0 printed the wrong SDK; `TestSDKVersionMatchesGoMod` now pins it, and run
  `ca version` on the release before calling it done.
- **Test install.ps1 on both Windows shells.** `pwsh` and Windows PowerShell 5.1 differ (the
  redirect target is `ResponseUri` on 5.1, `RequestMessage.RequestUri` on 7); the `install-windows`
  job runs `scripts/test-install.ps1` under each, and a fake must answer in the edition's shape.
- **Never `Select-Object -First 1` a native command whose file you then replace.** It stops the
  pipeline without waiting, and the process still holds the exe ("being used by another process").
- **Windows PowerShell wraps long warnings even into a file** — match message text whitespace-free.
- **Under `irm | iex` the installer inherits the caller's scope:** throw rather than exit (an exit
  closes the shell, pinned by the test's sentinel), and `Set-StrictMode -Off` inside the function.
- **Validate a downloaded binary before it replaces anything**; a broken one installed over ours
  then reads as a foreign program and blocks the next install.
- **Split an error by kind before choosing the remedy.** `MkdirTemp` failing as permission denied
  earns "run with sudo"; every other cause returns the wrapped error. One branch for both told a
  full disk to use sudo (`ca update`, #10).
- **A zero-value enum must not mean an outcome.** `State` starts at `iota + 1`, so an unset result
  matches no reported state.
- **A probe that fails has two answers:** it ran and exited non-zero (not ours, leave it quietly),
  or it never started or timed out (unknown, warn). Folding both into one answer warns about a
  foreign program or overwrites an unknown one.
- **A child process gets no credentials it does not need.** Strip `CLOUDAFRICA_*` from anything
  `ca` runs that is not `ca` doing the user's request, such as a version probe of a sibling binary
  (Copilot, #10).
- **Name the command the way the user invoked it.** A notice or remedy uses `a.exe()`, so a user
  running `cloudafrica` is not told to run `ca`, which may be another program (Copilot, #10).
- **A rename keeps the source's mtime;** age a renamed file by a timestamp in its name, not by its
  ModTime. Any cleanup that may run while another process is mid-swap leaves young files alone
  (Copilot, #10).
- **Every input form of a body flag gets a refused case.** `--data` took `null`, an array, and a
  valid object followed by trailing content; each needs a test (#11).
- **A decode error is never discarded.** A 2xx whose body is not JSON, a list page missing its rows
  key or with a non-string cursor, is an error, not an empty result (#11).
- **Every overlay section gets a generator refusal test.** A stale noun, an unknown converter or an
  unused column set passed silently until each had one (#11).
- **A fallback covers every failure of the thing it falls back from.** The name lookup fell back to
  the id only for one named scope; any list failure but 401 or a cancelled context must (#11).
- **An "at least one of" check counts every way a field can be given,** `--data` included (#11).
- **A PR body's rule is a claim about the code.** "An unlisted operation fails generation" was
  false; check it against the code path before writing it (Copilot, #11).
- **Read page-size and mode rules off the spec's own text, per route.** A global page size sent
  `limit=200` to `/v1/activity`, which takes 1 to 50; the same route's two modes cannot mix, and
  `--limit` and `--cursor` bypassed both checks until probed one flag away (#14).
- **A fixture fills every field a column reads.** A blank time or address column keeps a wrong path
  green; assert full rows, and revert a path to see the test go red (#14).
- **A multi-flag mode needs all its flags, non-blank.** `--subject-type` alone reached the API (Copilot, #14).
- **A summary view must not hide the thing the command is for.** `support ticket get` showed a
  message count and no message text (Copilot, #14).
- **Run each deliverable command with `-q` on the response it really gets.** `member create -q`
  printed nothing because the response keys on `userId` (#14).
- **A doc claim about a flag comes from the CLI, not the API's description.** `vm restore` was documented
  as taking RFC 3339, `--cursor` without `--limit`, and every resource flag as taking a name; none held
  (#15). Read the code path or run it.
- **A `$(ca …)` capture of a confirm command needs `--yes`**, since a capture is not a terminal (#15).
- **`git add -N .` stages deletions**, so `git diff --exit-code` after it misses a removed file; check
  `git status --porcelain` instead (#15).
