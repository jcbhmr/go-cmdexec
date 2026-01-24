//go:build openbsd && !mips64

package exec

import "golang.org/x/sys/unix"

func init() {
	openbsdlibcDup3 = unix.Dup3
}
