// Copyright 2013 gopm authors.
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
	"os"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"
	"github.com/codegangsta/cli"
	"path"

	"github.com/gpmgo/gopm/doc"
	"github.com/gpmgo/gopm/log"
)

var CmdRun = cli.Command{
	Name:  "run",
	Usage: "link dependencies and go run",
	Description: `Command run links dependencies according to gopmfile,
and execute 'go run'

gopm run <go run commands>
gopm run -l  will recursively find .gopmfile with value localPath and run the cmd in the .gopmfile,windows os is unspported, you need to run the command right at the localPath dir.
gopm run -l -r run go souce command 
gopm run -l -t run go test command
`,
	Action: runRun,
	Flags: []cli.Flag{
		cli.BoolFlag{"local,l", "run command with local gopath context"},
		cli.BoolFlag{"test,t", "test go souce files"},
		cli.BoolFlag{"run,r", "run go souce files"},
	},
}

func runRun(ctx *cli.Context) {
	setup(ctx)
	//support unix only
	if ctx.Bool("local") {
		var localPath string
		var localWd string
		var err error
		var wd string
		var gf *goconfig.ConfigFile
		wd, _ = os.Getwd()
		//recursively find project .gopmfile
		for wd != "/" {
			gf, _ = goconfig.LoadConfigFile(".gopmfile")
			if gf != nil {
				localPath = gf.MustValue("project", "localPath")
			}
			if localPath != "" {
				break
			}
			os.Chdir("..")
			wd, _ = os.Getwd()
		}
		if wd == "/" {
			log.Fatal("run", "no gopmfile in the directory or parent directory")
		}
		var argss string
		var args []string
		args = ctx.Args()
		var argsLen = len(args)
		if argsLen == 1 {
			args = append([]string{"go", "run"}, args...)
		}
		if ctx.Bool("run") {
			argss = gf.MustValue("cmd", "run")
			args = strings.Split(argss, " ")
		}
		if ctx.Bool("test") {
			argss = gf.MustValue("cmd", "test")
			args = strings.Split(argss, " ")
		}
		if localPath == "" {
			log.Fatal("run", "No localPath set")
		}
		localWd = gf.MustValue("project", "localWd")
		if localWd == "" {
			localWd = localPath
		}
		// if not abs path , then join it with localPath
		if !path.IsAbs(localWd) {
			localWd = path.Join(localPath, localWd)
		}
		argsLen = len(args)
		for i := 0; i < argsLen; i++ {
			strings.Trim(args[i], " ")
		}
		if argsLen < 2 {
			log.Fatal("run", "cmd arguments less than 2")
			log.Fatal("run", "running .gopmfile cmd at "+localWd)
		}
		err = execCmd(localPath, localWd, args...)
		if err != nil {
			log.Error("run", "Fail to run program:")
			log.Fatal("", "\t"+err.Error())
		}
		return
	}
	// Get GOPATH.
	installGopath = com.GetGOPATHs()[0]
	if com.IsDir(installGopath) {
		isHasGopath = true
		log.Log("Indicated GOPATH: %s", installGopath)
		installGopath += "/src"
	}
	// run command with gopm repos context
	// need version control , auto link to GOPATH/src repos
	genNewGoPath(ctx, false)
	defer os.RemoveAll(doc.VENDOR)

	log.Trace("Running...")

	cmdArgs := []string{"go", "run"}
	cmdArgs = append(cmdArgs, ctx.Args()...)
	err := execCmd(newGoPath, newCurPath, cmdArgs...)
	if err != nil {
		log.Error("run", "Fail to run program:")
		log.Fatal("", "\t"+err.Error())
	}

	log.Success("SUCC", "run", "Command executed successfully!")
}
