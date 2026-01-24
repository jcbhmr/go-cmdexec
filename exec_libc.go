//go:build aix || solaris

package exec

import (
	"errors"
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
		// err = unix.IoctlSetPointerInt(sys.Ctty, unix.TIOCSPGRP, pgrp)
		err = errors.New("cannot use unix.TIOCSPGRP (untyped int constant 18446744071562359926) as int value in argument to unix.IoctlSetPointerInt (overflows)")
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
			switch runtime.GOOS {
			case "illumos", "solaris":
				_, err = unix.FcntlInt(uintptr(f), solarisF_DUP2FD_CLOEXEC, nextfd)
			default:
				err = unix.Dup2(f, nextfd)
				if err != nil {
					return err
				}
				unix.CloseOnExec(nextfd)
			}
			if err != nil {
				return err
			}
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
		if solarisTIOCSCTTY == 0 {
			return unix.ENOSYS
		}
		err = unix.IoctlSetInt(sys.Ctty, solarisTIOCSCTTY, 0)
		if err != nil {
			return err
		}
	}

	return unix.Exec(argv0, argv, attr.Env)
}

// Set in exec_solaris.go
var solarisF_DUP2FD_CLOEXEC int
var solarisTIOCSCTTY int
