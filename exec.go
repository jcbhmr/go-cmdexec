//go:build unix || plan9

package exec

import (
	"os"
	"runtime"
	"syscall"
)

// ExecProcess is similar to [os.StartProcess]. Instead of starting
// the process, it replaces the current process with the new one using [syscall.Exec].
//
// ExecProcess always returns a non-nil error.
func ExecProcess(name string, argv []string, attr *os.ProcAttr) error {
	sysattr := (*procAttrExt)(attr).lower()

	// Platform-specific
	err := execProcess(name, argv, sysattr)
	runtime.KeepAlive(attr.Files)
	return err
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
