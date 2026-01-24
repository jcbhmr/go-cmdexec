package exec

import "golang.org/x/sys/unix"

func init() {
	solarisF_DUP2FD_CLOEXEC = unix.F_DUP2FD_CLOEXEC
	solarisTIOCSCTTY = unix.TIOCSCTTY
}
