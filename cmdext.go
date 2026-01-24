//go:build unix || plan9

package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
)

// CmdExt is a trait-like newtype of [os/exec.Cmd] with additional methods.
//
// You can use this trait-like newtype by casting [*os/exec.Cmd]
// (or equivalent newtype) to *CmdExt:
//
//	var cmd *exec.Cmd = exec.Command("mycommand")
//	(*CmdExt)(cmd) /* .Exec() or whatever */
//
// You are encouraged to do the (*CmdExt)(cmd) conversion inline where needed instead
// of assigning the expression to a new variable.
type CmdExt exec.Cmd

// Exec is similar to [os/exec.Cmd.Start]. Instead of spawning a new process, it replaces
// the current process with the new one using [syscall.Exec].
//
// Exec always returns a non-nil error.
func (c *CmdExt) Exec() error {
	var stdin, stdout, stderr *os.File
	var ok bool
	if c.Stdin != nil {
		stdin, ok = c.Stdin.(*os.File)
		if !ok {
			return errors.New("exec: Stdin is not an *os.File")
		}
	} else {
		stdin = os.Stdin
	}
	if c.Stdout != nil {
		stdout, ok = c.Stdout.(*os.File)
		if !ok {
			return errors.New("exec: Stdout is not an *os.File")
		}
	} else {
		stdout = os.Stdout
	}
	if c.Stderr != nil {
		stderr, ok = c.Stderr.(*os.File)
		if !ok {
			return errors.New("exec: Stderr is not an *os.File")
		}
	} else {
		stderr = os.Stderr
	}
	c.Stdin, c.Stdout, c.Stderr = stdin, stdout, stderr

	path, argv, attr, err := c.lower(stdin, stdout, stderr)
	if err != nil {
		return err
	}
	return ExecProcess(path, argv, attr)
}

// lower lowers an [exec.Cmd] instance into the arguments required by [os.StartProcess] and [ExecProcess].
func (c *CmdExt) lower(stdin, stdout, stderr *os.File) (path string, argv []string, attr *os.ProcAttr, err error) {
	err = c.ensureIsBuilding()
	if err != nil {
		return "", nil, nil, err
	}

	path = c.Path
	argv = c.argv()
	attr, err = c.attr(stdin, stdout, stderr)
	if err != nil {
		return "", nil, nil, err
	}
	return path, argv, attr, nil
}

func (c *CmdExt) ctx() context.Context {
	return *(*context.Context)(reflect.ValueOf((*exec.Cmd)(c)).Elem().FieldByName("ctx").Addr().UnsafePointer())
}

func (c *CmdExt) lookPathErr() error {
	return *(*error)(reflect.ValueOf((*exec.Cmd)(c)).Elem().FieldByName("lookPathErr").Addr().UnsafePointer())
}

// ensureIsBuilding returns an error if the command is not in the "building" state.
//
// The states of an [exec.Cmd] are: "building", "started", and "done".
func (c *CmdExt) ensureIsBuilding() error {
	if c.Process != nil {
		return errors.New("exec: already started")
	}

	lookPathErr := c.lookPathErr()
	if c.Path == "" && c.Err == nil && lookPathErr == nil {
		c.Err = errors.New("exec: no command")
	}
	if c.Err != nil || lookPathErr != nil {
		if lookPathErr != nil {
			return lookPathErr
		}
		return c.Err
	}

	ctx := c.ctx()
	if c.Cancel != nil && ctx == nil {
		return errors.New("exec: command with a non-nil Cancel was not created with exec.CommandContext")
	}
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

func (c *CmdExt) argv() []string {
	if len(c.Args) > 0 {
		return c.Args
	} else {
		return []string{c.Path}
	}
}

// attr constructs the [os.ProcAttr] for the [os/exec.Cmd] that is ready to be passed to [os.StartProcess] or [ExecProcess].
//
// You must provide stdin, stdout, and stderr as [*os.File] instances because the Stdin,
// Stdout, and Stderr fields are of type [io.Reader] and [io.Writer] respectively; they don't have file descriptors.
// Callers are free to preprocess Stdin, Stdout, and Stderr as needed
// and provide the underlying [*os.File] instances to this method.
func (c *CmdExt) attr(stdin, stdout, stderr *os.File) (*os.ProcAttr, error) {
	files := make([]*os.File, 3, 3+len(c.ExtraFiles))
	files[0] = stdin
	files[1] = stdout
	files[2] = stderr
	files = append(files, c.ExtraFiles...)

	env := (*exec.Cmd)(c).Environ() // Swallows error

	return (&os.ProcAttr{
		Dir:   c.Dir,
		Files: files,
		Env:   env,
		Sys:   c.SysProcAttr,
	}), nil
}
