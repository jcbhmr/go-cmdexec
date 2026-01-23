//go:build unix || plan9

package os

import (
	"os"

	syscall2 "github.com/jcbhmr/go-exec/syscall"
)

// ExecProcess is similar to [os.StartProcess]. Instead of starting
// the process, it replaces the current process with the new one using [unix.Exec] or [syscall.Exec].
//
// ExecProcess always returns a non-nil error.
func ExecProcess(name string, argv []string, attr *os.ProcAttr) error {
	// Platform-specific
	return syscall2.ExecProcess(name, argv, (*procAttrExt)(attr).lower())
}
