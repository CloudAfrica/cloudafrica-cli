# PR review rounds

One row per panel round. `rule` = the loop rule in force; origin = where the finding sat.

| PR | round | reviewedSha | outcome | findings (should-fix / optional) | mandateYield (raised/confirmed) | rule |
|---|---|---|---|---|---|---|
| #27 docs: one example per section | 1 | fe48165 | gating | 8 / 1, +1 pre-existing, 1 false positive | silent-failure 0/0, code-quality 2/2, comments 4/4, tests 2/1, senior-generalist 5/4 | ship-pr (any gating round) |
| #27 | 2 (fix-gate) | 8f082f9 | advisory | 1 / 1, +1 pre-existing | reconcile-critic 3/2 | ship-pr |
| CA-5701 CLI docs: help in ca's words | 1 | 4bf2901 | gating | 10 / 3 | silent-failure 1/1, code-quality 2/2, comments 5/5, tests 4/4, senior-generalist 5/5 | legacy |
| CA-5701 | 2 (fix-gate) | c35c7ca | advisory | 1 / 2 | reconcile-critic 3/3 | legacy |

criteriaReceipts: every round-1 reviewer `review:general=28 of 28`, `review:code=18 of 18` plus its own tag; round 2 `review:reconcile-critic=6 of 6`. CA-5701: the same in both rounds.

## Author misses (what the self-check had to do)

- A docs rewrite silently removed coverage from a guard test: flags moved from full `ca ...` invocations into bare prose spans, so validated (command, flag) pairs fell 65 -> 50. Count the guarded pairs before and after.
- Trimming examples dropped information the new prose did not restate (the `-q` secret-key caveat, `--enabled=false`, the per-shell completion commands). Diff the removed lines.
- Numeric and universal claims written in the rewrite were false ("Three flags", "each with its own example", "every VM"), and round 2 caught the replacement ("one example per section") as false again: the sweep grepped the old wording, never counted the new claim. Test the replacement claim, not the old text.
- The rule was applied to some `example: |` blocks, not all. Sweep the class, not the reported instances.
- The panel was first dispatched on an uncommitted tree and returned a false clean (`hasChanges:false`). Commit before dispatching in a cross-repo ship.
