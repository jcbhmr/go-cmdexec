//go:build unix || plan9

package os

import (
	"os"
	"syscall"
)

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
