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
	"path"

	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdBuild = cli.Command{
	Name:  "build",
	Usage: "link dependencies and go build",
	Description: `Command build links dependencies according to gopmfile,
and execute 'go build'

gopm build <go build commands>`,
	Action: runBuild,
	Flags: []cli.Flag{
		cli.BoolFlag{"update, u", "update pakcage(s) and dependencies if any"},
		cli.BoolFlag{"remote, r", "build with pakcages in gopm local repository only"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func buildBinary(ctx *cli.Context, args ...string) {
	target, newGopath, newCurPath := genNewGopath(ctx, false)

	log.Trace("Building...")

	cmdArgs := []string{"go", "build"}
	cmdArgs = append(cmdArgs, args...)
	if err := execCmd(newGopath, newCurPath, cmdArgs...); err != nil {
		log.Error("build", "fail to build program:")
		log.Fatal("", "\t"+err.Error())
	}

	if setting.IsWindowsXP {
		fName := path.Base(target)
		binName := fName + ".exe"
		os.Remove(binName)
		exePath := path.Join(newCurPath, doc.VENDOR, "src", target, binName)
		if com.IsFile(exePath) {
			if err := os.Rename(exePath, path.Join(newCurPath, binName)); err != nil {
				log.Error("build", "fail to move binary:")
				log.Fatal("", "\t"+err.Error())
			}
		} else {
			log.Warn("No binary generated")
		}
	}
}

func runBuild(ctx *cli.Context) {
	setup(ctx)

	os.RemoveAll(doc.VENDOR)
	if !setting.Debug {
		defer os.RemoveAll(doc.VENDOR)
	}
	buildBinary(ctx, ctx.Args()...)

	log.Success("SUCC", "build", "Command executed successfully!")
}
