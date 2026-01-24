# Ergonomic `Exec` for Go

🪄 \*nix-specific `.Exec()` method for `os/exec.Cmd` instances

<table align=center><td>

```go
//go:build unix || plan9
cmd := exec.Command("go", "version")
err := (*jcbhmrexec.CmdExt)(cmd).Exec()
log.Fatal(err)
```

</table>

## Installation

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=Go&logoColor=FFFFFF)

```sh
go get github.com/jcbhmr/go-exec
```

This module requires `unix || plan9` to work. It does not work on Windows.

## Usage

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=Go&logoColor=FFFFFF)
![macOS](https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=macOS&logoColor=FFFFFF)
![Linux](https://img.shields.io/badge/Linux-222222?style=for-the-badge&logo=Linux&logoColor=FCC624)

The primary feature of this module is the `CmdExt.Exec` method which replaces the current process with the command specified in the `os/exec.Cmd` instance. You are encouraged to cast `*exec.Cmd` to `*jcbhmrexec.CmdExt` at the time of use to access the `Exec` method.

```go
//go:build unix || plan9

import (
    "os/exec"
    "log"

    jcbhmrexec "github.com/jcbhmr/go-exec"
)

func main() {
    cmd := exec.Command("go", "version")
    err := (*jcbhmrexec.CmdExt)(cmd).Exec()
    log.Fatal(err)
}
```

This module also offers `ExecProcess` (`os.StartProcess` equivalent) and `ExecProcess2` (`syscall.StartProcess` equivalent) functions for lower-level process execution.

## Development

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=Go&logoColor=FFFFFF)

- `CmdExt.Exec` is implemented in terms of `ExecProcess`.
- `ExecProcess` is implemented in terms of `ExecProcess2`.
- `ExecProcess2` has platform-specific `execProcess2` implementations for `unix` and `plan9`.
- `execProcess2` on `unix` is implemented in terms of `execProcess3`.
- `execProcess2` on `plan9` doesn't need to delegate further.
- `execProcess3` has platform-specific implementations for various Unix platforms.
- `execProcess2` on `unix` is based on [`syscall.forkExec`](https://github.com/golang/go/blob/go1.25.6/src/syscall/exec_unix.go#L143).
- `execProcess2` on `plan9` is based on [`syscall.forkExec`](https://github.com/golang/go/blob/go1.25.6/src/syscall/exec_plan9.go#L362).
- `execProcess3` implementations are based on platform-specific `syscall.forkAndExecInChild` implementations.
- We aren't in a post-`fork` child environment, so we can perform allocations.
- We aren't implementing a part of `syscall`, so using `os.WriteFile` or similar doesn't create a circular dependency.
