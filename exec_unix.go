//go:build unix

package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

type CmdExt exec.Cmd

func (c *CmdExt) Exec() error {
	if c.Process != nil {
		return errors.New("exec: already started")
	}

	lookPathErr := c.lookPathErr()
	if c.Path == "" && c.Err == nil && *lookPathErr == nil {
		c.Err = errors.New("exec: no command")
	}
	if c.Err != nil || *lookPathErr != nil {
		if *lookPathErr != nil {
			return *lookPathErr
		}
		return c.Err
	}

	ctx := c.ctx()
	if c.Cancel != nil && *ctx == nil {
		return errors.New("exec: command with a non-nil Cancel was not created with exec.CommandContext")
	}
	if *ctx != nil {
		select {
		case <-(*ctx).Done():
			return (*ctx).Err()
		default:
		}
	}

	attr, err := c.execAttr()
	if err != nil {
		return err
	}
	sys := attr.sys()
	if sys.Setctty && sys.Foreground {
		return errors.New("exec: both Setctty and Foreground set in SysProcAttr")
	}
	if sys.Setctty && sys.Ctty >= len(attr.Files) {
		return errors.New("exec: Setctty set but Ctty not valid in child")
	}

	return c.doExec(attr)
}

func (c *CmdExt) argv() []string {
	argv := func() []string {
		if len(c.Args) > 0 {
			return c.Args
		} else {
			return []string{c.Path}
		}
	}()
	if (runtime.GOOS == "freebsd" || runtime.GOOS == "dragonfly") && len(argv) > 0 && len(argv[0]) > len(c.Path) {
		argv[0] = c.Path
	}
	return argv
}

func (c *CmdExt) ctx() *context.Context {
	return (*context.Context)(unsafe.Pointer(reflect.ValueOf((*exec.Cmd)(c)).Elem().FieldByName("ctx").Addr().Pointer()))
}

func (c *CmdExt) lookPathErr() *error {
	return (*error)(unsafe.Pointer(reflect.ValueOf((*exec.Cmd)(c)).Elem().FieldByName("lookPathErr").Addr().Pointer()))
}

func (c *CmdExt) execStdio() (stdin *os.File, stdout *os.File, stderr *os.File, err error) {
	if c.Stdin == nil {
		c.Stdin = os.Stdin
	}
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	if c.Stderr == nil {
		c.Stderr = os.Stderr
	}
	if f, ok := c.Stdin.(*os.File); ok {
		stdin = f
	} else {
		err = errors.New("exec: Stdin is not an *os.File")
		return
	}
	if f, ok := c.Stdout.(*os.File); ok {
		stdout = f
	} else {
		err = errors.New("exec: Stdout is not an *os.File")
		return
	}
	if f, ok := c.Stderr.(*os.File); ok {
		stderr = f
	} else {
		err = errors.New("exec: Stderr is not an *os.File")
		return
	}
	return
}

func (c *CmdExt) execAttr() (*procAttr, error) {
	files := make([]*os.File, 0, 3+len(c.ExtraFiles))
	stdin, stdout, stderr, err := c.execStdio()
	if err != nil {
		return nil, err
	}
	files = append(files, stdin, stdout, stderr)
	files = append(files, c.ExtraFiles...)

	env := (*exec.Cmd)(c).Environ() // Swallows error

	return (*procAttr)(&os.ProcAttr{
		Dir:   c.Dir,
		Files: files,
		Env:   env,
		Sys:   c.SysProcAttr,
	}), nil
}

type procAttr os.ProcAttr

var zeroSysProcAttr unix.SysProcAttr

func (p *procAttr) sys() *unix.SysProcAttr {
	if p.Sys != nil {
		return p.Sys
	} else {
		return &zeroSysProcAttr
	}
}
