// Copyright 2014 Unknown
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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Unknwon/goconfig"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdRun = cli.Command{
	Name:  "run",
	Usage: "link dependencies and go run",
	Description: `Command run links dependencies according to gopmfile,
and execute 'go run'

gopm run <go run commands>
gopm run -l will recursively find .gopmfile with value localPath 
and run the cmd in the .gopmfile, Windows hasn't supported yet,
you need to run the command right at the local_gopath dir.`,
	Action: runRun,
	Flags: []cli.Flag{
		cli.BoolFlag{"local, l", "run command with local gopath context"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runRun(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	// TODO: So ugly, need to fix.
	if ctx.Bool("local") {
		var localGopath string
		var err error
		var wd string
		var gf *goconfig.ConfigFile
		wd, _ = os.Getwd()
		for wd != "/" {
			gf, _ = goconfig.LoadConfigFile(".gopmfile")
			if gf != nil {
				localGopath = gf.MustValue("project", "local_gopath")
			}
			if localGopath != "" {
				break
			}
			os.Chdir("..")
			wd, _ = os.Getwd()
		}
		if wd == "/" {
			log.Fatal("run", "no gopmfile in the directory or parent directory")
		}
		argss := gf.MustValue("run", "cmd")
		if localGopath == "" {
			log.Fatal("run", "No local GOPATH set")
		}
		args := strings.Split(argss, " ")
		argsLen := len(args)
		for i := 0; i < argsLen; i++ {
			strings.Trim(args[i], " ")
		}
		if len(args) < 2 {
			log.Fatal("run", "cmd arguments less than 2")
		}
		if err = execCmd(localGopath, localGopath, args...); err != nil {
			log.Error("run", "Fail to run program:")
			log.Fatal("", "\t"+err.Error())
		}
		return
	}

	os.RemoveAll(doc.VENDOR)
	if !setting.Debug {
		defer os.RemoveAll(doc.VENDOR)
	}
	// Run command with gopm repos context
	// need version control , auto link to GOPATH/src repos
	_, newGopath, newCurPath, err := genNewGopath(ctx, false)
	if err != nil {
		errors.SetError(err)
		return
	}

	log.Trace("Running...")

	cmdArgs := []string{"go", "run"}
	cmdArgs = append(cmdArgs, ctx.Args()...)
	if err := execCmd(newGopath, newCurPath, cmdArgs...); err != nil {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Fail to run program: %v", err))
			return
		}
		log.Error("run", "Fail to run program:")
		log.Fatal("", "\t"+err.Error())
	}

	log.Success("SUCC", "run", "Command executed successfully!")
}
