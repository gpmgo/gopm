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

package cmd

import (
	"fmt"
	"path"

	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/errors"
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
		cli.StringFlag{"tags", "", "apply build tags", ""},
		// cli.BoolFlag{"package, p", "only install non-main packages", ""},
		cli.BoolFlag{"remote, r", "build with packages in gopm local repository only", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func runInstall(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if err := linkVendors(ctx, ""); err != nil {
		errors.SetError(err)
		return
	}

	// Get target name.
	gfPath := path.Join(setting.WorkDir, setting.GOPMFILE)
	_, target, err := parseGopmfile(gfPath)
	if err != nil {
		errors.SetError(fmt.Errorf("fail to parse gopmfile: %v", err))
		return
	}

	log.Info("Installing...")

	cmdArgs := []string{"go", "install"}
	if ctx.Bool("verbose") {
		cmdArgs = append(cmdArgs, "-v")
	}
	if len(ctx.String("tags")) > 0 {
		cmdArgs = append(cmdArgs, "-tags")
		cmdArgs = append(cmdArgs, ctx.String("tags"))
	}
	cmdArgs = append(cmdArgs, target)
	if err := execCmd(setting.DefaultVendor, setting.WorkDir, cmdArgs...); err != nil {
		errors.SetError(fmt.Errorf("fail to run program: %v", err))
		return
	}

	log.Info("Command executed successfully!")
}
