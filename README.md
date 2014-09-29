Go Package Manager [![wercker status](https://app.wercker.com/status/899e79d6738e606dab98c915a269d531/s/ "wercker status")](https://app.wercker.com/project/bykey/899e79d6738e606dab98c915a269d531)
=========================

Gopm(Go Package Manager) is a Go package manage and build tool for Go.

**News** Give a shoot for [Gopm.io](http://gopm.io) for versioning caching and delivering Go package service.

**News** Want online cross-compile and download service? Just try [GoBuild.io](http://gobuild.io) and it won't let you down! BTW, it's powered by Gopm.

Please see **[Documentation](https://github.com/gpmgo/docs)** before you ever start.

## Requirements

- Go development environment: >= **go1.2**

## Installation

We use [gobuild](http://build.gopm.io) to do online cross-platform compile work, you can see the full available binary list [here](http://gobuild.io/download/github.com/gpmgo/gopm).

### Install from source code

	go get -u github.com/gpmgo/gopm

The executable will be produced under `$GOPATH/bin` in your file system; for global use purpose, we recommand you to add this path into your `PATH` environment variable.

## Features

- No requirement for installing any version control system tool like `git`, `svn` or `hg` in order to download packages.
- Download, install or build your packages with specific revisions.
- When build program with `gopm build` or `gopm install`, everything just happen in its own GOPATH and do not bother anything you've done unless you told to.
- Put your Go projects on anywhere you want(through `.gopmfile`).

## Commands

```
NAME:
   Gopm - Go Package Manager

USAGE:
   Gopm [global options] command [command options] [arguments...]

VERSION:
   0.8.3.0929 Beta

COMMANDS:
   list		list all dependencies of current project
   gen		generate a gopmfile for current Go project
   get		fetch remote package(s) and dependencies
   bin		download and link dependencies and build binary
   config	configurate gopm settings
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

This project is under Apache v2 License. See the [LICENSE](LICENSE) file for the full license text.