# Traps

Author misses the PR review caught. Check these before opening a PR.

- **A spec's list is a checklist.** When a spec enumerates what a doc or command must cover, tick
  each item against the diff; the README for token elevation shipped missing three of §8b's items.
- **The write of a command's deliverable checks its error.** If stdout carries the result a script
  captures (a secret, a Location), a failed `fmt.Fprint*` must return an error, or the command exits
  0 with nothing captured. Pin it with an `h.failStdout` test per command (missed again on #8).
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
  CI fails on staticcheck quick-fixes that `go vet` does not report.
- **Every enum state gets a branch and a test.** A `== CURRENT` check with an `else` lumps the
  other four upgrade states together; switch over the generated consts and add a table row each.
- **`--wait` never exits 0 without waiting** unless the end state already holds. Each early return
  (no operation id, nothing to do) checks the flag.
- **A substring assertion must not match its negation.** `"ready"` is in `"not ready"`; assert the
  whole row, and include the negative case.
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
