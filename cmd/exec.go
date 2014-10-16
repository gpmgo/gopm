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
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/errors"
)

var CmdExec = cli.Command{
	Name:        "exec",
	Usage:       "build and execute command",
	Description: `Command exec builds and executes command according to gopmfile`,
	Action:      runExec,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func runExec(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	// TODO: download and build command

	// TODO: exec command
}
