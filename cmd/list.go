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
	"sort"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/goconfig"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdList = cli.Command{
	Name:  "list",
	Usage: "list all dependencies of current project",
	Description: `Command list lists all dependencies of current Go project

gopm list

Make sure you run this command in the root path of a go project.`,
	Action: runList,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"test, t", "show test imports", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func verSuffix(gf *goconfig.ConfigFile, name string) string {
	val := gf.MustValue("deps", name)
	if len(val) > 0 {
		val = " @ " + val
	}
	return val
}

// getDepList gets list of dependencies in root path format and nature order.
func getDepList(ctx *cli.Context, target, pkgPath, vendor string) ([]string, error) {
	vendorSrc := path.Join(vendor, "src")
	rootPath := doc.GetRootPath(target)
	// If work directory is not in GOPATH, then need to setup a vendor path.
	if !setting.HasGOPATHSetting || !strings.HasPrefix(pkgPath, setting.InstallGopath) {
		// Make link of self.
		log.Debug("Linking %s...", rootPath)
		from := pkgPath
		to := path.Join(vendorSrc, rootPath)
		if setting.Debug {
			log.Debug("Linking from %s to %s", from, to)
		}
		if err := autoLink(from, to); err != nil {
			return nil, err
		}
	}

	imports, err := doc.ListImports(target, rootPath, vendor, pkgPath, ctx.String("tags"), ctx.Bool("test"))
	if err != nil {
		return nil, err
	}

	list := make([]string, 0, len(imports))
	for _, name := range imports {
		name = doc.GetRootPath(name)
		if !base.IsSliceContainsStr(list, name) {
			list = append(list, name)
		}
	}
	sort.Strings(list)
	return list, nil
}

func runList(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if !setting.HasGOPATHSetting && !base.IsFile(setting.DefaultGopmfile) {
		log.Warn("Dependency list may contain package itself without GOPATH setting and gopmfile.")
	}
	gf, target, err := parseGopmfile(setting.DefaultGopmfile)
	if err != nil {
		errors.SetError(err)
		return
	}

	list, err := getDepList(ctx, target, setting.WorkDir, setting.DefaultVendor)
	if err != nil {
		errors.SetError(err)
		return
	}

	fmt.Printf("Dependency list (%d):\n", len(list))
	for _, name := range list {
		fmt.Printf("-> %s%s\n", name, verSuffix(gf, name))
	}
}
