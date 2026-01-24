//go:build unix || plan9

package exec_test

import (
	"log"
	"os"

	jcbhmrexec "github.com/jcbhmr/go-exec"
)

func ExampleExecProcess() {
	log.Fatal(jcbhmrexec.ExecProcess("go", os.Args, &os.ProcAttr{
		Env: os.Environ(),
	}))
}
