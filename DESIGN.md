# Findings schema

This document is the source of truth for the `reviewd` output contract:
what a `Finding` means field-by-field, and what the surrounding `Report`
guarantees. Detectors, the CLI, the eval scorer, and the GitHub Action
are all just consumers of this — none of them get to redefine it locally.

## Finding

```go
type Finding struct {
	File        string
	Line        int
	Detector    string
	RuleID      string
	Severity    Severity
	Confidence  float64
	Explanation string
	Fingerprint string
}
```

**`File`** is repo-relative, forward-slash, always — e.g. `internal/foo.go`,
never `/tmp/reviewd-worktree-abc123/internal/foo.go` and never
`./internal/foo.go`. Detectors that read from a disposable worktree (the
LLM-backed ones) will naturally produce absolute paths rooted at the
worktree; that gets stripped once, centrally, before a `Report` is ever
encoded. A detector is never responsible for knowing its own path is
"wrong" — the funnel fixes it, so every detector can just report what it
saw.

**`Line`** is the line number in the *new* file, matching
`diff.Line.NewLine`. This is deliberate, not incidental: a `Finding`
describes something about the code as it will exist after the change,
so it should point at a line a reviewer can actually open and see. It is
never `diff.Line.OldLine` and never a byte offset. A `Finding` about
something that was deleted (which no current detector produces, but a
future one might) has no meaningful `Line` in this scheme — that detector
would need its own convention, documented where it's introduced.

**`Detector`** is the stable identifier from `Detector.Name()` —
`"secrets"`, `"secrets-llm"`, etc. It's an identity, not a description;
never a sentence, never something that changes between runs.

**`RuleID`** identifies *which* rule inside a detector fired — e.g. a
regex pattern name (`"aws-access-key-id"`) or a model-assigned rule
label. Optional in the sense that a detector with only one rule can
leave it empty, but if a detector has more than one way to fire, this is
what lets someone suppress one of them without suppressing all of them.

**`Severity`** is one of the fixed set defined in `severity.go`
(`critical`, `high`, `medium`, `low`), parsed through `ParseSeverity` at
the boundary where free text (model output) becomes a `Finding`. There
is no fifth value, and nothing downstream should have a `default:` case
that has to guess what an unrecognized severity means — invalid severity
is rejected before it becomes a `Finding` at all, not tolerated after.

**`Confidence`** is a detector's own estimate, `0.0`–`1.0`, of how
likely the finding is real (vs. a false positive). It is **never**
filtered when a `Report` is encoded to JSON — see "Confidence is not a
filter" below.

**`Explanation`** is one or two sentences, human-readable, aimed at
someone deciding whether to act on the finding. Not a template dump of
the rule definition; specific to what was seen at this file/line.

**`Fingerprint`** is a hash of `Detector + RuleID + File + Line`,
computed once, centrally, by `Report.Normalize`. It is *not* a hash of
the whole `Finding` — `Explanation` and `Confidence` can both change
between two runs over the same diff (a regex's explanation text gets
reworded, an LLM's confidence estimate isn't perfectly stable) without
that counting as "a different finding." The fingerprint's job is to answer
one question: is this the same defect, at the same place, reported by
the same rule, as one seen before? That's what a future suppression list
or "still open" tracker keys off of — it has to survive cosmetic changes
to the finding's text, or every suppression breaks the next time
`Explanation`'s wording is tweaked.

## Confidence is not a filter

`WriteJSON` never drops a finding for having low confidence — the JSON
report is the complete, unfiltered record of everything every detector
saw, always. Only `WriteText` (the human-facing view) takes a
`threshold` and drops findings below it.

This asymmetry is structural, not a default someone forgot to wire
through everywhere: the eval harness needs every finding, including the
ones a human would never want to see, to measure a detector's precision
at different thresholds. If low-confidence findings were dropped before
they ever hit disk, that measurement would be impossible after the fact
— you'd have to know in advance which threshold you'd eventually want
and re-run every detector if you guessed wrong. `Meta.Threshold` records
what threshold *was configured* for the run, precisely so a JSON report
can be inspected later and someone can tell "was this just not shown at
the time, or did the detector never see it."

## Report

```go
type Report struct {
	SchemaVersion int
	Meta          Meta
	Detectors     []DetectorStatus
	Findings      []Finding
}
```

**`SchemaVersion`** bumps only on a breaking change to this document —
a field renamed or removed, a meaning changed. Adding an optional field
is not a breaking change and does not bump it.

**`Meta`** describes the run itself: which `reviewd` binary
(`ReviewdVersion`), which repo state (`RepoSHA`, `Ref`), which model,
what threshold was configured (not applied — see above), and when. This
is what makes a `Report` interpretable on its own, without needing to
know what command produced it.

**`Detectors`** is one `DetectorStatus` per detector that was scheduled
to run, regardless of outcome — `ran`, `errored`, or `skipped`. A
detector erroring never removes its findings from consideration for
other detectors, and never removes its entry from this list; it shows up
here with `Status: "errored"` and whatever partial context is useful
(`Error`, `DurationMS`). A `Report` with fewer findings than expected
should always be explainable by reading `Detectors`, not by guessing.

**`Findings`** is deterministically ordered — `Report.Normalize` sorts
it (by file, then line, then detector, as a stable tiebreak) so that two
runs over an identical diff, with identical detector output, produce
byte-identical JSON. This is what makes a golden-file test of the
encoder possible at all, and it's also just good behavior for anything
downstream that diffs two reports against each other.

## Ordering guarantee

Nothing in `internal/findings` is allowed to reorder or filter findings
except `Report.Normalize` (ordering, path rewrite, fingerprinting) and
`WriteText` (threshold filtering, and only there). Detectors themselves
emit findings in whatever order is natural for how they scan; they are
not responsible for sorting or deduplicating their own output.
