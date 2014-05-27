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
	"os"

	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdInstall = cli.Command{
	Name:  "install",
	Usage: "link dependencies and go install",
	Description: `Command install links dependencies according to gopmfile,
and execute 'go install'

gopm install

If no argument is supplied, then gopmfile must be present`,
	Action: runInstall,
	Flags: []cli.Flag{
		cli.BoolFlag{"package, p", "only install non-main packages"},
		cli.BoolFlag{"remote, r", "build with pakcages in gopm local repository only"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runInstall(ctx *cli.Context) {
	setup(ctx)

	var target, srcPath string
	switch len(ctx.Args()) {
	case 0:
		_, target, _ = genGopmfile()
		srcPath = setting.WorkDir
	default:
		log.Error("install", "Too many arguments:")
		log.Error("", "\tno argument needed")
		log.Help("Try 'gopm help install' to get more information")
	}

	os.RemoveAll(doc.VENDOR)

	_, newGopath, newCurPath := genNewGopath(ctx, false)

	log.Trace("Installing...")

	var installRepos []string
	if ctx.Bool("package") {
		installRepos = doc.GetImports(target, doc.GetRootPath(target), srcPath, false)
	} else {
		installRepos = []string{target}
	}

	for _, repo := range installRepos {
		cmdArgs := []string{"go", "install"}

		if ctx.Bool("verbose") {
			cmdArgs = append(cmdArgs, "-v")
		}
		cmdArgs = append(cmdArgs, repo)
		if err := execCmd(newGopath, newCurPath, cmdArgs...); err != nil {
			log.Error("install", "Fail to install program:")
			log.Fatal("", "\t"+err.Error())
		}
	}

	log.Success("SUCC", "install", "Command executed successfully!")
}
