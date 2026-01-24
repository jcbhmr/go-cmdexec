//go:build unix

package exec

import (
	"errors"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

var zeroProcAttr syscall.ProcAttr
var zeroSysProcAttr unix.SysProcAttr

func execProcess2(argv0 string, argv []string, attr *syscall.ProcAttr) error {
	if attr == nil {
		attr = &zeroProcAttr
	}
	sys := attr.Sys
	if sys == nil {
		sys = &zeroSysProcAttr
	}

	if (runtime.GOOS == "freebsd" || runtime.GOOS == "dragonfly") && len(argv) > 0 && len(argv[0]) > len(argv0) {
		argv[0] = argv0
	}

	if sys.Setctty && sys.Foreground {
		return errors.New("both Setctty and Foreground set in SysProcAttr")
	}
	if sys.Setctty && sys.Ctty >= len(attr.Files) {
		return errors.New("Setctty set but Ctty not valid in child")
	}

	// Platform-specific
	return execProcess3(argv0, argv, attr, sys)
}
