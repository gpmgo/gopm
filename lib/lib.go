// +build go1.2

// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Package lib is a library version of Gopm(Go Package Manager).
package lib

import (
	"io"
	"runtime"

	"github.com/gpmgo/gopm/cmd"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

const APP_VER = "0.8.8.0307 Beta"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	setting.LibraryMode = true
}

func Run(args []string) *setting.Error {
	app := cli.NewApp()
	app.Name = "Gopm"
	app.Usage = "Go Package Manager"
	app.Version = APP_VER
	app.Commands = []cli.Command{
		cmd.CmdList,
		cmd.CmdGen,
		cmd.CmdGet,
		cmd.CmdBin,
		cmd.CmdConfig,
		cmd.CmdRun,
		cmd.CmdTest,
		cmd.CmdBuild,
		cmd.CmdInstall,
		cmd.CmdClean,
		cmd.CmdUpdate,
		// CmdSearch,
	}
	app.Flags = append(app.Flags, []cli.Flag{
		cli.BoolFlag{"noterm, n", "disable color output", ""},
		cli.BoolFlag{"strict, s", "strict mode", ""},
		cli.BoolFlag{"debug, d", "debug mode", ""},
	}...)
	app.Run(args)
	return setting.RuntimeError
}

func SetOutput(out io.Writer) {
	log.Output = out
}
