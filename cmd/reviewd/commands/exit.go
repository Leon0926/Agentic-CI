package commands

import "errors"

// Exit codes for `reviewd run`. 0/1 are outcomes about the code under
// review; 2 is reserved for reviewd itself failing to do its job.
const (
	ExitClean         = 0 // ran clean, nothing at or above threshold
	ExitFindingsFound = 1 // ran clean, but findings at or above threshold exist
	ExitInfraFailure  = 2 // reviewd itself failed: bad diff, git error, sandbox failure, a detector erroring, ...
)

// exitError pairs an error with the process exit code it should
// produce, so main can tell "found findings" apart from "reviewd broke"
// without parsing error strings.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// ExitCode extracts the exit code a RunE error should produce. nil is
// ExitClean; any error not wrapping an *exitError is treated as
// ExitInfraFailure — the safe default for a failure this command didn't
// specifically classify.
func ExitCode(err error) int {
	if err == nil {
		return ExitClean
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	return ExitInfraFailure
}
