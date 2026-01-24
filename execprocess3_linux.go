package exec

import (
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

func execProcess3(argv0 string, argv []string, attr *syscall.ProcAttr, sys *unix.SysProcAttr) (err error) {
	var uidmap []byte
	if sys.UidMappings != nil {
		uidmap = formatIDMappings(sys.UidMappings)
	}

	var setgroups []byte
	var gidmap []byte
	if sys.GidMappings != nil {
		if sys.GidMappingsEnableSetgroups {
			setgroups = []byte("allow\x00")
		} else {
			setgroups = []byte("deny\x00")
		}
		gidmap = formatIDMappings(sys.GidMappings)
	}

	fd := make([]int, len(attr.Files))
	nextfd := len(attr.Files)
	for i, ufd := range attr.Files {
		if nextfd < int(ufd) {
			nextfd = int(ufd)
		}
		fd[i] = int(ufd)
	}
	nextfd++

	if len(sys.AmbientCaps) > 0 {
		err = unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0)
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
		err = unix.IoctlSetPointerInt(sys.Ctty, unix.TIOCSPGRP, pgrp)
		if err != nil {
			return err
		}
	}

	if sys.Unshareflags != 0 {
		err = unix.Unshare(int(sys.Unshareflags))
		if err != nil {
			return err
		}

		if sys.Unshareflags&unix.CLONE_NEWUSER != 0 && sys.GidMappings != nil {
			err = os.WriteFile("/proc/self/setgroups", setgroups, 0o600)
			if err != nil {
				return err
			}

			err = os.WriteFile("/proc/self/gid_map", gidmap, 0o600)
			if err != nil {
				return err
			}
		}

		if sys.Unshareflags&unix.CLONE_NEWUSER != 0 && sys.UidMappings != nil {
			err = os.WriteFile("/proc/self/uid_map", uidmap, 0o600)
			if err != nil {
				return err
			}
		}

		if sys.Unshareflags&unix.CLONE_NEWNS == unix.CLONE_NEWNS {
			err = unix.Mount("none", "/", "", unix.MS_REC|unix.MS_PRIVATE, "")
			if err != nil {
				return err
			}
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
		if !(sys.GidMappings != nil && sys.GidMappingsEnableSetgroups && ngroups == 0) && !cred.NoSetGroups {
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

	if len(sys.AmbientCaps) != 0 {
		var capHeader unix.CapUserHeader
		var capData [2]unix.CapUserData
		capHeader.Version = unix.LINUX_CAPABILITY_VERSION_3
		err = unix.Capget(&capHeader, &capData[0])
		if err != nil {
			return err
		}
		for _, c := range sys.AmbientCaps {
			capData[c >> 5].Permitted |= 1 << uint(c&31)
			capData[c >> 5].Inheritable |= 1 << uint(c&31)
		}
		err = unix.Capset(&capHeader, &capData[0])
		if err != nil {
			return err
		}
		for _, c := range sys.AmbientCaps {
			err = unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_RAISE, c, 0, 0)
			if err != nil {
				return err
			}
		}
	}

	if attr.Dir != "" {
		err = os.Chdir(attr.Dir)
		if err != nil {
			return err
		}
	}

	if sys.Pdeathsig != 0 {
		err = unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(sys.Pdeathsig), 0, 0, 0)
		if err != nil {
			return err
		}
	}

	for i, f := range fd {
		if f >= 0 && f < i {
			err = unix.Dup3(f, nextfd, unix.O_CLOEXEC)
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
		err = unix.Dup3(f, i, 0)
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
		err = unix.IoctlSetPointerInt(sys.Ctty, unix.TIOCSCTTY, 1)
		if err != nil {
			return err
		}
	}

	if sys.Ptrace {
		_, err = ptrace(unix.PTRACE_TRACEME, 0, 0, 0)
		if err != nil {
			return err
		}
	}

	return unix.Exec(argv0, argv, attr.Env)
}

func ptrace(op int, pid int, addr uintptr, data uintptr) (int, error) {
	r1, _, err := syscall.Syscall6(syscall.SYS_PTRACE, uintptr(op), uintptr(pid), addr, data, 0, 0)
	if err != 0 {
		return 0, err
	}
	return int(r1), nil
}

func formatIDMappings(idMap []syscall.SysProcIDMap) []byte {
	var data []byte
	for _, im := range idMap {
		data = append(data, strconv.Itoa(im.ContainerID)+" "+strconv.Itoa(im.HostID)+" "+strconv.Itoa(im.Size)+"\n"...)
	}
	return data
}
