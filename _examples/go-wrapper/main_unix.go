//go:build unix

package main

import (
	"log"
	"os"
	"os/exec"

	jcbhmrexec "github.com/jcbhmr/go-exec"
)

func main() {
	cmd := exec.Command("go", os.Args[1:]...)
	err := (*jcbhmrexec.CmdExt)(cmd).Exec()
	log.Fatal(err)
}
