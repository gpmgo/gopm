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
	"sort"

	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"
)

var CmdList = cli.Command{
	Name:  "list",
	Usage: "list all dependencies of current project",
	Description: `Command list lsit all dependencies of current Go project

gopm list

Make sure you run this command in the root path of a go project.`,
	Action: runList,
	Flags: []cli.Flag{
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runList(ctx *cli.Context) {
	setup(ctx)
	_, _, imports := genGopmfile()
	list := make([]string, 0, len(imports))
	for _, name := range imports {
		if !com.IsSliceContainsStr(list, name) {
			list = append(list, name)
		}
	}
	sort.Strings(list)

	fmt.Println("Dependency list:")
	for _, name := range list {
		fmt.Println("->", name)
	}
}
