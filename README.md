## Choria Binary Updater

This is a system that can update running software in place from a update server.

The idea is that you might embed this project into your system and use it to do on-demand updates of your binary.

This project does not handle restarts of your running process, in general though a simple *os.Exec()* is enough for that.

## Features

 * Support a basic HTTP(S) server for hosting the repository
 * Verifies checksums of data inside bz2 archives
 * Creates a backup of the binary being replaced
 * Support rollback by returning the backup to the original target
 * Support notifying the caller if the rollback failed
 * Utility to add a file into a repository

## Planned features

 * Crypto signature validation
 * Cloud object store support

## HTTP Repository

Hosting a repository is simple and can be done on any HTTP(S) server.

The structure of the repository can be seen here:

```
.
└── 0.7.0
    ├── darwin
    │   └── amd64
    │       ├── choria.bz2
    │       └── release.json
    └── linux
        └── amd64
            ├── choria.bz2
            └── release.json
```

Here is a single release - `0.7.0` - with 64 bit binaries for Linux and OS X.

The `release.json` looks like this:

```json
{
  "binary": "choria.bz2",
  "hash": "e3c7bfaa626e8c38ed66a674b5265df677d4938fbd8d835341b3734d1ce522c2"
}
```

*binary* is simply the path to the file to download and the *hash* is a *sha256* hash of the binary before it was bz2 compressed. 

Files can be added to this structure using the `update-repo` utility that can be downloaded from our releases page:

```
$ update-repo choria --arch amd64 --os linux --version 1.2.3
```

Valid os and arch combinations can be found using `go tool dist list`.

## Usage

A basic binary that updates itself and re-runs itself can be seen below:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"syscall"

	updater "github.com/choria-io/go-updater"
)

func main() {
	logger := log.New(os.Stdout, "go-updater ", 0)

	opts := []updater.Option{
		updater.Logger(logger),
		updater.SourceRepo("https://repo.example.net/updater"),
		updater.Version("0.7.0"),
		updater.TargetFile(os.Args[0]),
	}

	err := updater.Apply(opts...)
	if err != nil {
		if err := updater.RollbackError(err); err != nil {
			panic(fmt.Errorf("Update failed and rollback also failed, system in broken state: %s", err))
		}

		panic(fmt.Errorf("Update failed: %s", err))
	}

	fmt.Println("Updated self, restarting process.....")

	err = syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		panic(fmt.Errorf("Could not restart server: %s", err))
	}
}
```

