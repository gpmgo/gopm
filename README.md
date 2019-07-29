🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨

In favor of [Go Modules Proxy](https://github.com/golang/go/wiki/Modules#are-there-always-on-module-repositories-and-enterprise-proxies) since Go 1.11, this project has been **archived** and website (gopm.io) will be **taken down** as of **12/31/2019**.

🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨

Go Package Manager
=========================

Gopm (Go Package Manager) is a Go package manage and build tool for Go.

Please see **[Documentation](https://github.com/gpmgo/docs)** before you start.

## Requirements

- Go development environment: >= **go1.2**

## Installation

### Install from source code

    go get -u github.com/gpmgo/gopm

The executable will be produced under `$GOPATH/bin` in your file system; for global use purpose, we recommend you to add this path into your `PATH` environment variable.

## Features

- No requirement for installing any version control system tool like `git` or `hg` in order to download packages.
- Download, install or build your packages with specific revisions.
- When building programs with `gopm build` or `gopm install`, everything just happens in its own GOPATH and does not bother anything you've done (unless you told it to).
- Can put your Go projects anywhere you want (through `.gopmfile`).

## Commands

```
NAME:
   Gopm - Go Package Manager

USAGE:
   Gopm [global options] command [command options] [arguments...]

COMMANDS:
   list		list all dependencies of current project
   gen		generate a gopmfile for current Go project
   get		fetch remote package(s) and dependencies
   bin		download and link dependencies and build binary
   config	configure gopm settings
   run		link dependencies and go run
   test		link dependencies and go test
   build	link dependencies and go build
   install	link dependencies and go install
   clean	clean all temporary files
   update	check and update gopm resources including itself
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --noterm, -n		disable color output
   --strict, -s		strict mode
   --debug, -d		debug mode
   --help, -h		show help
   --version, -v	print the version
```

## License

This project is under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for the full license text.
