package events

const (
	// ExitCodeSuccess is the exit code for a successful run.
	ExitCodeSuccess = iota
	ExitCodeGenericFailure
	ExitCodeTimeoutFailure
)
