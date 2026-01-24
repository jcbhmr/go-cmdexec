//go:build unix || plan9

package exec

import (
	"os"
	"syscall"
)

// ExecProcess is similar to [os.StartProcess]. Instead of starting
// the process, it replaces the current process with the new one using [syscall.Exec].
//
// ExecProcess always returns a non-nil error.
func ExecProcess(name string, argv []string, attr *os.ProcAttr) error {
	// Platform-specific
	return ExecProcess2(name, argv, (*procAttrExt)(attr).lower())
}

// ExecProcess2 is similar to [syscall.StartProcess]. Instead of starting
// the process, it replaces the current process with the new one using
// [syscall.Exec].
//
// ExecProcess2 always returns a non-nil error.
func ExecProcess2(argv0 string, argv []string, attr *syscall.ProcAttr) error {
	// Platform-specific
	return execProcess2(argv0, argv, attr)
}

type procAttrExt os.ProcAttr

func (p *procAttrExt) lower() *syscall.ProcAttr {
	if p == nil {
		return nil
	}

	sysattr := &syscall.ProcAttr{
		Dir: p.Dir,
		Env: p.Env,
		Sys: p.Sys,
	}

	sysattr.Files = make([]uintptr, 0, len(p.Files))
	for _, f := range p.Files {
		sysattr.Files = append(sysattr.Files, f.Fd())
	}

	return sysattr
}
