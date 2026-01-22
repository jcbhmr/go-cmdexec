//go:build dragonfly || netbsd || (openbsd && mips64)

package exec

import (
	"errors"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// https://github.com/golang/go/blob/go1.25.6/src/syscall/exec_bsd.go#L54
func (c *CmdExt) doExec(attr *procAttr) (err error) {
	sys := attr.sys()

	fd := make([]int, len(attr.Files))
	nextfd := len(attr.Files)
	for i, f := range attr.Files {
		ufd := f.Fd()
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	if sys.Ptrace {
		return errors.New("exec: Ptrace not implemented")
	}

	if sys.Setsid {
		return errors.New("exec: Setsid not implemented")
	}

	if sys.Foreground {
		return errors.New("exec: Foreground not implemented")
	}

	if sys.Chroot != "" {
		err = unix.Chroot(sys.Chroot)
		if err != nil {
			return err
		}
	}

	if cred := sys.Credential; cred != nil {
		ngroups := len(cred.Groups)
		groups := ([]int)(nil)
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

	if c.Dir != "" {
		err = os.Chdir(c.Dir)
		if err != nil {
			return err
		}
	}

	for i := range fd {
		if fd[i] >= 0 && fd[i] < i {
			if runtime.GOOS == "netbsd" || (runtime.GOOS == "openbsd" && runtime.GOARCH == "mips64") {
				err = unix.Dup3(fd[i], nextfd, unix.O_CLOEXEC)
			} else if runtime.GOOS == "dragonfly" {
				_, err = unix.FcntlInt(uintptr(fd[i]), unix.F_DUP2FD_CLOEXEC, nextfd)
			} else {
				err = unix.Dup2(fd[i], nextfd)
				if err != nil {
					return err
				}
				_, err = unix.FcntlInt(uintptr(nextfd), unix.F_SETFD, unix.FD_CLOEXEC)
			}
			if err != nil {
				return err
			}
			fd[i] = nextfd
			nextfd++
		}
	}

	for i := range fd {
		if fd[i] == -1 {
			_ = unix.Close(i)
			continue
		}
		if fd[i] == i {
			_, err = unix.FcntlInt(uintptr(fd[i]), unix.F_SETFD, 0)
			if err != nil {
				return err
			}
			continue
		}
		err = unix.Dup2(fd[i], i)
		if err != nil {
			return err
		}
	}

	for i := len(fd); i < 3; i++ {
		_ = unix.Close(i)
	}

	if sys.Noctty {
		err = unix.IoctlSetInt(sys.Ctty, unix.TIOCSCTTY, 0)
		if err != nil {
			return err
		}
	}

	return unix.Exec(c.Path, c.argv(), attr.Env)
}
