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
	"path/filepath"
	"sort"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
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
		cli.BoolFlag{"local, l", "generate local GOPATH directories"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runGen(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	gf, _, _, err := genGopmfile()
	if err != nil {
		errors.SetError(err)
		return
	}

	if ctx.Bool("local") {
		localGopath := gf.MustValue("project", "local_gopath")
		if len(localGopath) == 0 {
			localGopath = "./vendor"
			gf.SetValue("project", "local_gopath", localGopath)
			if err = saveGopmfile(gf, setting.GOPMFILE); err != nil {
				errors.SetError(err)
				return
			}
		}

		for _, name := range []string{"src", "pkg", "bin"} {
			os.MkdirAll(path.Join(localGopath, name), os.ModePerm)
		}
	}
	log.Success("SUCC", "gen", "Generate gopmfile successfully!")
}

// genGopmfile generates gopmfile and returns it,
// along with target and dependencies.
func genGopmfile() (*goconfig.ConfigFile, string, []string, error) {
	if !com.IsExist(setting.GOPMFILE) {
		os.Create(setting.GOPMFILE)
	}
	gf, err := loadGopmfile(setting.GOPMFILE)
	if err != nil {
		return nil, "", nil, err
	}

	// Check dependencies.
	target := doc.ParseTarget(gf.MustValue("target", "path"))
	rootPath := doc.GetRootPath(target)

	oldGopath := os.Getenv("GOPATH")
	if len(oldGopath) == 0 || !com.IsExist(path.Join(oldGopath, "src", rootPath)) {
		tmpPath := path.Join(setting.InstallRepoPath, rootPath)
		os.RemoveAll(tmpPath)
		os.MkdirAll(path.Dir(tmpPath), os.ModePerm)

		relPath, err := filepath.Rel(target, rootPath)
		if err != nil {
			log.Error("", "Fail to get relative path of target")
			log.Fatal("", "\t"+err.Error())
		}
		absPath, _ := filepath.Abs(relPath)
		if setting.IsWindows {
			com.CopyDir(absPath, tmpPath, func(filePath string) bool {
				return !strings.Contains(filePath, ".git")
			})
		} else {
			os.Symlink(absPath, tmpPath)
		}
	}

	imports, err := doc.GetImports(target, rootPath, setting.WorkDir, false)
	if err != nil {
		return nil, "", nil, err
	}
	sort.Strings(imports)
	for _, name := range imports {
		name = doc.GetRootPath(name)
		// Check if user has specified the version.
		if val := gf.MustValue("deps", name); len(val) == 0 {
			gf.SetValue("deps", name, "")
		}
	}

	// Check resources.
	if _, err := gf.GetValue("res", "include"); err != nil {
		resList := make([]string, 0, len(setting.CommonRes))
		for _, res := range setting.CommonRes {
			if com.IsExist(res) {
				resList = append(resList, res)
			}
		}
		gf.SetValue("res", "include", strings.Join(resList, "|"))
	}

	return gf, target, imports, saveGopmfile(gf, setting.GOPMFILE)
}
