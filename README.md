# `Exec` for Go

🪄 Unix-specific `.Exec()` method for `exec.Cmd` instances

## Installation

```sh
go get github.com/jcbhmr/go-exec
```

This module requires `unix || plan9` to work. It does not work on Windows.

## Usage

```go
//go:build unix

import (
    "os/exec"
    jcbhmrexec "github.com/jcbhmr/go-exec"
)

func main() {
    cmd := exec.Command("go", "version")
    err := (*jcbhmrexec.CmdExt)(cmd).Exec()
    log.Fatal(err)
}
```

## Development

TODO
