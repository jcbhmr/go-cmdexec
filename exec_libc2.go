//go:build darwin || (openbsd && !mips64)

package exec

import (
	"os"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

var forked sync.Mutex

func execProcessUnix(argv0 string, argv []string, attr *syscall.ProcAttr, sys *unix.SysProcAttr) (err error) {
	fd := make([]int, len(attr.Files))
	nextfd := len(attr.Files)
	for i, ufd := range attr.Files {
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	forked.Lock()
	defer forked.Unlock()

	if sys.Ptrace {
		_, err = ptrace(unix.PTRACE_TRACEME, 0, 0, 0)
		if err != nil {
			return err
		}
	}

	if sys.Setsid {
		_, err = unix.Setsid()
		if err != nil {
			return err
		}
	}

	if sys.Setpgid || sys.Foreground {
		err = unix.Setpgid(0, sys.Pgid)
		if err != nil {
			return err
		}
	}

	if sys.Foreground {
		pgrp := sys.Pgid
		if pgrp == 0 {
			pgrp = os.Getpid()
		}
		err = unix.IoctlSetPointerInt(int(attr.Files[0]), unix.TIOCSPGRP, pgrp)
		if err != nil {
			return err
		}
	}

	if sys.Chroot != "" {
		err = unix.Chroot(sys.Chroot)
		if err != nil {
			return err
		}
	}

	if cred := sys.Credential; cred != nil {
		ngroups := len(cred.Groups)
		var groups []int
		if ngroups > 0 {
			groups = make([]int, ngroups)
			for i, g := range cred.Groups {
				groups[i] = int(g)
			}
		}
		if !cred.NoSetGroups {
			err = unix.Setgroups(groups)
			if err != nil {
				return err
			}
		}
		err = unix.Setgid(int(cred.Gid))
		if err != nil {
			return err
		}
		err = unix.Setuid(int(cred.Uid))
		if err != nil {
			return err
		}
	}

	if attr.Dir != "" {
		err = os.Chdir(attr.Dir)
		if err != nil {
			return err
		}
	}

	for i, f := range fd {
		if f >= 0 && f < i {
			if runtime.GOOS == "openbsd" {
				err = openbsdlibcDup3(f, nextfd, unix.O_CLOEXEC)
			} else {
				err = unix.Dup2(f, nextfd)
				if err != nil {
					return err
				}
				unix.CloseOnExec(nextfd)
			}
			if err != nil {
				return err
			}
			fd[i] = nextfd
			nextfd++
		}
	}

	for i, f := range fd {
		if f == -1 {
			_ = unix.Close(i)
			continue
		}
		if f == i {
			_, err = unix.FcntlInt(uintptr(f), unix.F_SETFD, 0)
			if err != nil {
				return err
			}
			continue
		}
		err = unix.Dup2(f, i)
		if err != nil {
			return err
		}
	}

	for i := len(fd); i < 3; i++ {
		_ = unix.Close(i)
	}

	if sys.Noctty {
		err = unix.IoctlSetInt(0, unix.TIOCNOTTY, 0)
		if err != nil {
			return err
		}
	}

	if sys.Setctty {
		err = unix.IoctlSetInt(sys.Ctty, unix.TIOCSCTTY, 0)
		if err != nil {
			return err
		}
	}

	return unix.Exec(argv0, argv, attr.Env)
}

// Set in exec_openbsdlibc.go
var openbsdlibcDup3 func(oldfd int, newfd int, flags int) error

func ptrace(op int, pid int, addr uintptr, data uintptr) (int, error) {
	r1, _, err := syscall.Syscall6(syscall.SYS_PTRACE, uintptr(op), uintptr(pid), addr, data, 0, 0)
	if err != 0 {
		return 0, err
	}
	return int(r1), nil
}
