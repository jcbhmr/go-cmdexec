//go:build unix

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Cmd exec.Cmd

// Exec replaces the current process with the command specified in c.
func (c *Cmd) Exec() error {
	if c.Process != nil {
		return fmt.Errorf("%s: already started", "exec")
	}
	if c.Path == "" {
		c.Err = fmt.Errorf("%s: no command", "exec")
	}
	if c.Err != nil {
		return c.Err
	}

	// Want to do the non-fork-related stuff. This function does not need to
	// be //go:noescape or avoid allocations; this is not a low-level fork+exec.
	// https://github.com/golang/go/blob/go1.25.6/src/syscall/exec_libc.go#L81

	var err error

	envv := (*exec.Cmd)(c).Environ()
	if dirAbs, err := filepath.Abs(c.Dir); err == nil {
		(*environ)(&envv).Set("PWD", dirAbs)
	}

	if c.SysProcAttr != nil {
		if c.SysProcAttr.Setsid {
			_, err = unix.Setsid()
			if err != nil {
				return err
			}
		}

		if c.SysProcAttr.Setpgid || c.SysProcAttr.Foreground {
			err = unix.Setpgid(0, c.SysProcAttr.Pgid)
			if err != nil {
				return err
			}
		}

		if c.SysProcAttr.Foreground {
			pgid := c.SysProcAttr.Pgid
			if pgid == 0 {
				pgid = os.Getpid()
			}
			err = unix.IoctlSetPointerInt(c.SysProcAttr.Ctty, unix.TIOCSPGRP, pgid)
			if err != nil {
				return err
			}
		}

		if c.SysProcAttr.Chroot != "" {
			err = unix.Chroot(c.SysProcAttr.Chroot)
			if err != nil {
				return err
			}
		}

		if c.SysProcAttr.Credential != nil {
			var groups []int
			if len(c.SysProcAttr.Credential.Groups) > 0 {
				groups = make([]int, len(c.SysProcAttr.Credential.Groups))
				for i, g := range c.SysProcAttr.Credential.Groups {
					groups[i] = int(g)
				}
			}
			if !c.SysProcAttr.Credential.NoSetGroups {
				err = unix.Setgroups(groups)
				if err != nil {
					return err
				}
			}
			err = unix.Setgid(int(c.SysProcAttr.Credential.Gid))
			if err != nil {
				return err
			}
			err = unix.Setuid(int(c.SysProcAttr.Credential.Uid))
			if err != nil {
				return err
			}
		}
	}

	if c.Dir != "" {
		err = os.Chdir(c.Dir)
		if err != nil {
			return err
		}
	}

	if f, ok := c.Stdin.(*os.File); ok && int(f.Fd()) != unix.Stdin {
		err = unix.Dup2(int(f.Fd()), unix.Stdin)
		if err != nil {
			return err
		}
	}
	if f, ok := c.Stdout.(*os.File); ok && int(f.Fd()) != unix.Stdout {
		err = unix.Dup2(int(f.Fd()), unix.Stdout)
		if err != nil {
			return err
		}
	}
	if f, ok := c.Stderr.(*os.File); ok && int(f.Fd()) != unix.Stderr {
		err = unix.Dup2(int(f.Fd()), unix.Stderr)
		if err != nil {
			return err
		}
	}

	if c.SysProcAttr != nil {
		if c.SysProcAttr.Noctty {
			err = unix.IoctlSetInt(unix.Stdin, unix.TIOCNOTTY, 0)
			if err != nil {
				return err
			}
		}

		if c.SysProcAttr.Setctty {
			if unix.TIOCSCTTY == 0 {
				return unix.ENOSYS
			}
			err = unix.IoctlSetInt(c.SysProcAttr.Ctty, unix.TIOCSCTTY, 0)
			if err != nil {
				return err
			}
		}
	}

	return unix.Exec(c.Path, c.argv(), envv)
}

func (c *Cmd) argv() []string {
	if len(c.Args) > 0 {
		return c.Args
	}
	return []string{c.Path}
}

type environ []string

func (e *environ) Set(key string, value string) {
	prefix := key + "="
	for i, kv := range *e {
		if kv[:len(prefix)] == prefix {
			(*e)[i] = prefix + value
			return
		}
	}
	// else
	*e = append(*e, prefix+value)
}
