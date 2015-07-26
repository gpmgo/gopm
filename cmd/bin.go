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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdBin = cli.Command{
	Name:  "bin",
	Usage: "download and link dependencies and build binary",
	Description: `Command bin downloads and links dependencies according to gopmfile,
and build executable binary to work directory

gopm bin <import path>@[<tag|commit|branch>:<value>]
gopm bin <package name>@[<tag|commit|branch>:<value>]

Can only specify one each time, and only works for projects that 
contain main package`,
	Action: runBin,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.StringFlag{"dir, d", "./", "build binary to given directory", ""},
		cli.BoolFlag{"update, u", "update package(s) and dependencies if any", ""},
		cli.BoolFlag{"remote, r", "build with packages in gopm local repository only", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func runBin(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 1 {
		errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 1"))
		return
	}

	// Check if given directory exists if specified.
	if ctx.IsSet("dir") && !base.IsDir(ctx.String("dir")) {
		errors.SetError(fmt.Errorf("Indicated path does not exist or not a directory"))
		return
	}

	// Backup exsited .vendor.
	if base.IsExist(setting.VENDOR) {
		os.Rename(setting.VENDOR, setting.VENDOR+".bak")
		defer func() {
			os.Rename(setting.VENDOR+".bak", setting.VENDOR)
		}()
	}

	// Parse package version.
	info := ctx.Args().First()
	pkgPath := info
	n := doc.NewNode(pkgPath, doc.BRANCH, "", true)
	if i := strings.Index(info, "@"); i > -1 {
		pkgPath = info[:i]
		var err error
		tp, val, err := validPkgInfo(info[i+1:])
		if err != nil {
			errors.SetError(err)
			return
		}
		n = doc.NewNode(pkgPath, tp, val, !ctx.Bool("download"))
	}

	// Check package name.
	if !strings.Contains(pkgPath, "/") {
		tmpPath, err := setting.GetPkgFullPath(pkgPath)
		if err != nil {
			errors.SetError(err)
			return
		}
		if tmpPath != pkgPath {
			n = doc.NewNode(tmpPath, n.Type, n.Value, n.IsGetDeps)
		}
	}

	if err := downloadPackages(".", ctx, []*doc.Node{n}); err != nil {
		errors.SetError(err)
		return
	}

	// Check if previous steps were successful.
	if !n.IsExist() {
		errors.SetError(fmt.Errorf("Download steps weren't successful"))
		return
	}

	tmpVendor := path.Join("vendor", path.Base(n.RootPath))
	os.RemoveAll(tmpVendor)
	os.RemoveAll(setting.VENDOR)
	defer func() {
		os.RemoveAll(tmpVendor)
		os.RemoveAll(setting.VENDOR)
	}()

	// FIXME: should use .gopm/temp path.
	if err := autoLink(n.InstallPath, tmpVendor); err != nil {
		errors.SetError(fmt.Errorf("Fail to link slef: %v", err))
		return
	}

	os.Chdir(tmpVendor)
	oldWorkDir := setting.WorkDir
	setting.WorkDir = path.Join(setting.WorkDir, tmpVendor)
	if !setting.Debug {
		defer func() {
			os.Chdir(oldWorkDir)
			os.RemoveAll("vendor")
			os.RemoveAll(setting.VENDOR)
		}()
	}

	// if err := buildBinary(ctx); err != nil {
	// 	errors.SetError(err)
	// 	return
	// }

	if err := linkVendors(ctx, n.ImportPath); err != nil {
		errors.SetError(err)
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
	cmdArgs = append(cmdArgs, n.ImportPath)
	if err := execCmd(setting.DefaultVendor, setting.WorkDir, cmdArgs...); err != nil {
		errors.SetError(fmt.Errorf("fail to run program: %v", err))
		return
	}

	gf, _, err := parseGopmfile(setting.GOPMFILE)
	if err != nil {
		errors.SetError(err)
		return
	}

	// Because build command moved binary to root path.
	binName := path.Base(n.ImportPath)
	binPath := path.Join(setting.DefaultVendor, "bin", path.Base(n.ImportPath))
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	// Move binary to given directory.
	movePath := oldWorkDir
	if ctx.IsSet("dir") {
		movePath = ctx.String("dir")
	} else if base.IsGoTool(n.ImportPath) {
		movePath = path.Join(runtime.GOROOT(), "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH)
		if !base.IsExist(binPath) {
			log.Info("Command executed successfully!")
			fmt.Println("Binary has been built into: " + movePath)
			return
		}
	}

	if !base.IsFile(binPath) {
		errors.SetError(fmt.Errorf("Previous steps weren't successful or the project does not contain main package"))
		return
	}

	if base.IsExist(path.Join(movePath, binName)) {
		if err := os.Remove(path.Join(movePath, binName)); err != nil {
			log.Warn("Cannot remove binary in work directory: %v", err)
		}
	}

	if err := os.Rename(binPath, movePath+"/"+binName); err != nil {
		errors.SetError(fmt.Errorf("Fail to move binary: %v", err))
		return
	}
	os.Chmod(movePath+"/"+binName, os.ModePerm)

	includes := strings.Split(gf.MustValue("res", "include"), "|")
	if len(includes) > 0 {
		log.Info("Copying resources to %s", movePath)
		for _, include := range includes {
			if base.IsDir(include) {
				os.RemoveAll(path.Join(movePath, include))
				if err := base.CopyDir(include, filepath.Join(movePath, include)); err != nil {
					errors.AppendError(errors.NewErrCopyResource(include))
				}
			}
		}
	}

	log.Info("Command executed successfully!")
	fmt.Println("Binary has been built into: " + movePath)
}
