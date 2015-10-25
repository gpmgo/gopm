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

	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdTest = cli.Command{
	Name:  "test",
	Usage: "link dependencies and go test",
	Description: `Command test links dependencies according to gopmfile,
and execute 'go test'

gopm test <go test commands>`,
	Action: runTest,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func runTest(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if err := linkVendors(ctx, ""); err != nil {
		errors.SetError(err)
		return
	}

	log.Info("Testing...")

	cmdArgs := []string{"go", "test"}
	if len(ctx.String("tags")) > 0 {
		cmdArgs = append(cmdArgs, "-tags")
		cmdArgs = append(cmdArgs, ctx.String("tags"))
	}
	if ctx.IsSet("verbose") {
		cmdArgs = append(cmdArgs, "-v")
	}
	cmdArgs = append(cmdArgs, ctx.Args()...)
	if err := execCmd(setting.DefaultVendor, setting.WorkDir, cmdArgs...); err != nil {
		errors.SetError(fmt.Errorf("fail to run program: %v", err))
		return
	}

	log.Info("Command executed successfully!")
}
