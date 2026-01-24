package exec

import (
	"encoding/binary"
	"os"
	"slices"
	"strconv"
	"syscall"

	"golang.org/x/sys/plan9"
)

var zeroProcAttr syscall.ProcAttr
var zeroSysProcAttr syscall.SysProcAttr

func execProcess2(argv0 string, argv []string, attr *syscall.ProcAttr) (err error) {
	if attr == nil {
		attr = &zeroProcAttr
	}
	sys := attr.Sys
	if sys == nil {
		sys = &zeroSysProcAttr
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

	dupdevfd, err := plan9.Open("#d", plan9.O_RDONLY)
	if err != nil {
		return err
	}

	statbuf := make([]byte, plan9.STATMAX)
	for {
		n, err := plan9.Pread(dupdevfd, statbuf, 0)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		for b := statbuf[:n]; len(b) > 0; {
			var s osString
			s, b = osStringList(b).Dirname()
			if s == nil {
				return plan9.ErrBadStat
			}
			if s[len(s)-1] == 'l' {
				continue
			}
			n, _ := strconv.Atoi(string(s.Bytes()))
			if n != dupdevfd && !slices.Contains(fd, n) {
				_ = plan9.Close(n)
			}
		}
	}
	_ = plan9.Close(dupdevfd)
	if attr.Dir != "" {
		err = os.Chdir(attr.Dir)
		if err != nil {
			return err
		}
	}

	for i, f := range fd {
		if f >= 0 && f < i {
			_, err = plan9.Dup(f, nextfd)
			if err != nil {
				return err
			}

		}
	}

	for i, f := range fd {
		if f == -1 {
			_ = plan9.Close(i)
			continue
		}
		if f == i {
			continue
		}
		_, err = plan9.Dup(f, i)
		if err != nil {
			return err
		}
	}

	for _, f := range fd {
		if f >= len(attr.Files) {
			_ = plan9.Close(f)
		}
	}

	return syscall.Exec(argv0, argv, attr.Env)
}

type osString []byte

func (o osString) Bytes() []byte {
	if len(o) < 2 {
		return nil
	}
	n, o := binary.LittleEndian.Uint16(o), o[2:]
	if int(n) > len(o) {
		return nil
	}
	return o[:n]
}

type osStringList []byte

// https://github.com/golang/go/blob/go1.25.6/src/syscall/exec_syscall.go#L36
const nameOffset = 39

func (o osStringList) Dirname() (name osString, rest osStringList) {
	if len(o) < 2 {
		return
	}
	size, o := binary.LittleEndian.Uint16(o), o[2:]
	if size < plan9.STATFIXLEN || int(size) > len(o) {
		return
	}
	name = osString(o[nameOffset:size])
	rest = o[size:]
	return
}
