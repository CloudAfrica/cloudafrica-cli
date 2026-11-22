# Traps

Author misses the PR review caught. Check these before opening a PR.

- **A spec's list is a checklist.** When a spec enumerates what a doc or command must cover, tick
  each item against the diff; the README for token elevation shipped missing three of §8b's items.
- **The write of a command's deliverable checks its error.** If stdout carries the result a script
  captures (a secret, a Location), a failed `fmt.Fprint*` must return an error, or the command exits
  0 with nothing captured.
- **Say what `ca` does, not what the gateway does.** A comment or help line about server behaviour
  is a claim about another repo; state ca's own rule, or cite the gateway source.
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
