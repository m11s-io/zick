package cli

import "fmt"

// SilentError signals a non-zero exit code without printing an additional
// error message. Use it when RunE has already printed human-readable output
// and only needs to set the exit code (e.g. "N packages below age gate").
//
// main() distinguishes SilentError from real errors: SilentError → os.Exit(Code),
// real error → print message + os.Exit(1).
type SilentError struct {
	Code int
}

func (e *SilentError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}
