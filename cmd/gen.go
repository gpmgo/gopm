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
	"os"
	"path"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdGen = cli.Command{
	Name:  "gen",
	Usage: "generate a gopmfile for current Go project",
	Description: `Command gen gets dependencies and generates a gopmfile

gopm gen

Make sure you run this command in the root path of a go project.`,
	Action: runGen,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"local, l", "generate local GOPATH directories", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func runGen(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	gfPath := path.Join(setting.WorkDir, setting.GOPMFILE)
	if !setting.HasGOPATHSetting && !base.IsFile(gfPath) {
		log.Warn("Dependency list may contain package itself without GOPATH setting and gopmfile.")
	}
	gf, target, err := parseGopmfile(gfPath)
	if err != nil {
		errors.SetError(err)
		return
	}

	list, err := getDepList(ctx, target, setting.WorkDir, setting.DefaultVendor)
	if err != nil {
		errors.SetError(err)
		return
	}
	for _, name := range list {
		// Check if user has specified the version.
		if val := gf.MustValue("deps", name); len(val) == 0 {
			gf.SetValue("deps", name, "")
		}
	}

	// Check resources.
	if _, err = gf.GetValue("res", "include"); err != nil {
		resList := make([]string, 0, len(setting.CommonRes))
		for _, res := range setting.CommonRes {
			if base.IsExist(res) {
				resList = append(resList, res)
			}
		}
		if len(resList) > 0 {
			gf.SetValue("res", "include", strings.Join(resList, "|"))
		}
	}

	if err = setting.SaveGopmfile(gf, gfPath); err != nil {
		errors.SetError(err)
		return
	}

	if ctx.Bool("local") {
		localGopath := gf.MustValue("project", "local_gopath")
		if len(localGopath) == 0 {
			localGopath = "./vendor"
			gf.SetValue("project", "local_gopath", localGopath)
			if err = setting.SaveGopmfile(gf, gfPath); err != nil {
				errors.SetError(err)
				return
			}
		}

		for _, name := range []string{"src", "pkg", "bin"} {
			os.MkdirAll(path.Join(localGopath, name), os.ModePerm)
		}
	}

	log.Info("Generate gopmfile successfully!")
}
