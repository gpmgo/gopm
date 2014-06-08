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

	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
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
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runTest(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	os.RemoveAll(doc.VENDOR)
	if !setting.Debug {
		defer os.RemoveAll(doc.VENDOR)
	}

	_, newGopath, newCurPath, err := genNewGopath(ctx, true)
	if err != nil {
		errors.SetError(err)
		return
	}

	log.Trace("Testing...")

	cmdArgs := []string{"go", "test"}
	cmdArgs = append(cmdArgs, ctx.Args()...)
	if err := execCmd(newGopath, newCurPath, cmdArgs...); err != nil {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Fail to run program: %v", err))
			return
		}
		log.Error("test", "Fail to run program:")
		log.Fatal("", "\t"+err.Error())
	}

	log.Success("SUCC", "test", "Command executed successfully!")
}
