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
