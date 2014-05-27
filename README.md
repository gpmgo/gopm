Go Package Manager [![wercker status](https://app.wercker.com/status/899e79d6738e606dab98c915a269d531/s/ "wercker status")](https://app.wercker.com/project/bykey/899e79d6738e606dab98c915a269d531)
=========================

Gopm(Go Package Manager) is a Go package manage tool for searching, installing, updating and sharing packages in Go.

**News** Want online cross-compile and download service? Just try [GoBuild.io](http://gobuild.io) and it won't let you down!

Please see **[Documentation](https://github.com/gpmgo/docs)** before you ever start.

Code Convention: based on [Go Code Convention](https://github.com/Unknwon/go-code-convention).

## Commands

```
NAME:
   Gopm - Go Package Manager

USAGE:
   Gopm [global options] command [command options] [arguments...]

VERSION:
   0.6.5.0527 Beta

COMMANDS:
   bin		download and link dependencies and build binary
   gen		generate a gopmfile for current Go project
   get		fetch remote package(s) and dependencies
   list		list all dependencies of current project
   config	configurate gopm settings
   run		link dependencies and go run
   test		link dependencies and go test
   build	link dependencies and go build
   install	link dependencies and go install
   clean	clean all temporary files
   help, h	Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --noterm, -n		disable color output
   --strict, -s		strict mode
   --debug, -d		debug mode
   --version, -v	print the version
   --help, -h		show help
```

## License

This project is under Apache v2 License. See the [LICENSE](LICENSE) file for the full license text.