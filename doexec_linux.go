package exec

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func (c *CmdExt) doExec(attr *procAttr) (err error) {
	sys := attr.sys()

	if sys.UidMappings != nil {
		return errors.New("exec: UidMappings not implemented")
	}

	if sys.GidMappings != nil {
		return errors.New("exec: GidMappings not implemented")
	}

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

	if sys.Cloneflags != 0 {
		return errors.New("exec: Cloneflags not implemented")
	}
	if sys.PidFD != nil {
		return errors.New("exec: PidFD not implemented")
	}
	if sys.UseCgroupFD {
		return errors.New("exec: UseCgroupFD not implemented")
	}

	if len(sys.AmbientCaps) > 0 {
		return errors.New("exec: AmbientCaps not implemented")
	}

	if sys.Setsid {
		_, err = unix.Setsid()
		if err != nil {
			return err
		}
	}

	if sys.Setpgid || sys.Foreground {
		return errors.New("exec: Setpgid not implemented")
	}

	if sys.Foreground {
		return errors.New("exec: Foreground not implemented")
	}

	if sys.Unshareflags != 0 {
		err = unix.Unshare(0)
		if err != nil {
			return err
		}

		// TODO
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

	if sys.Pdeathsig != 0 {
		return errors.New("exec: Pdeathsig not implemented")
	}

	for i := range fd {
		if fd[i] >= 0 && fd[i] < i {
			err = unix.Dup3(fd[i], i, unix.O_CLOEXEC)
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
		err = unix.Dup3(fd[i], i, 0)
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
		err = unix.IoctlSetInt(sys.Ctty, unix.TIOCSCTTY, 1)
		if err != nil {
			return err
		}
	}

	if sys.Ptrace {
		return errors.New("exec: Ptrace not implemented")
	}

	return unix.Exec(c.Path, c.argv(), attr.Env)
}
