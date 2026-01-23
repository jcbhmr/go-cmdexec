//go:build unix || plan9

package syscall

import "syscall"

// ExecProcess2 is similar to [syscall.StartProcess]. Instead of starting
// the process, it replaces the current process with the new one using
// [unix.Exec] or [syscall.Exec].
//
// ExecProcess2 always returns a non-nil error.
func ExecProcess2(argv0 string, argv []string, attr *syscall.ProcAttr) error {
	// Platform-specific
	return execProcess2(argv0, argv, attr)
}
