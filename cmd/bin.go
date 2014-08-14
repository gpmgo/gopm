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
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

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
		cli.StringFlag{"dir, d", "./", "build binary to given directory"},
		cli.BoolFlag{"update, u", "update pakcage(s) and dependencies if any"},
		cli.BoolFlag{"remote, r", "build with pakcages in gopm local repository only"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runBin(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 1 {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 1"))
			return
		}
		log.Error("bin", "Incorrect number of arguments for command")
		log.Error("", "\tshould have 1")
		log.Help("Try 'gopm help bin' to get more information")
	}

	// Check if given directory exists if specified.
	if ctx.IsSet("dir") && !com.IsDir(ctx.String("dir")) {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Indicated path does not exist or not a directory"))
			return
		}
		log.Error("bin", "Cannot start command:")
		log.Fatal("", "\tIndicated path does not exist or not a directory")
	}

	// Parse package version.
	info := ctx.Args().First()
	pkgPath := info
	n := doc.NewNode(pkgPath, doc.BRANCH, "", true)
	if i := strings.Index(info, "@"); i > -1 {
		pkgPath = info[:i]
		var err error
		n.Type, n.Value, err = validPkgInfo(info[i+1:])
		if err != nil {
			errors.SetError(err)
			return
		}
	}

	// Check package name.
	if !strings.Contains(pkgPath, "/") {
		tmpPath := setting.GetPkgFullPath(pkgPath)
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
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Download steps weren't successful"))
			return
		}
		log.Error("bin", "Cannot continue command:")
		log.Fatal("", "\tDownload steps weren't successful")
	}

	buildPath := path.Join(setting.InstallRepoPath, n.ImportPath)
	oldWorkDir := setting.WorkDir
	// Change to repository path.
	log.Log("Changing work directory to %s", buildPath)
	if err := os.Chdir(buildPath); err != nil {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Fail to change work directory: %v", err))
			return
		}
		log.Error("bin", "Fail to change work directory:")
		log.Fatal("", "\t"+err.Error())
	}
	setting.WorkDir = buildPath

	// TODO: should use .gopm/temp path.
	os.RemoveAll(path.Join(buildPath, doc.VENDOR))
	if !setting.Debug {
		defer os.RemoveAll(path.Join(buildPath, doc.VENDOR))
	}

	if err := buildBinary(ctx); err != nil {
		errors.SetError(err)
		return
	}

	gf, target, _, err := genGopmfile(ctx)
	if err != nil {
		errors.SetError(err)
		return
	}
	if target == "." {
		_, target = filepath.Split(setting.WorkDir)
	}

	// Because build command moved binary to root path.
	binName := path.Base(target)
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	if !com.IsFile(binName) {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Previous steps weren't successful or the project does not contain main package"))
			return
		}
		log.Error("bin", "Binary does not exist:")
		log.Error("", "\t"+binName)
		log.Fatal("", "\tPrevious steps weren't successful or the project does not contain main package")
	}

	// Move binary to given directory.
	movePath := oldWorkDir
	if ctx.IsSet("dir") {
		movePath = ctx.String("dir")
	} else if strings.HasPrefix(n.ImportPath, "code.google.com/p/go.tools/cmd/") {
		movePath = path.Join(runtime.GOROOT(), "pkg/tool", runtime.GOOS+"_"+runtime.GOARCH)
	}

	if com.IsExist(movePath + "/" + binName) {
		if err := os.Remove(movePath + "/" + binName); err != nil {
			log.Warn("Cannot remove binary in work directory:")
			log.Warn("\t %s", err)
		}
	}

	if err := os.Rename(binName, movePath+"/"+binName); err != nil {
		if setting.LibraryMode {
			errors.SetError(fmt.Errorf("Fail to move binary: %v", err))
			return
		}
		log.Error("bin", "Fail to move binary:")
		log.Fatal("", "\t"+err.Error())
	}
	os.Chmod(movePath+"/"+binName, os.ModePerm)

	includes := strings.Split(gf.MustValue("res", "include"), "|")
	if len(includes) > 0 {
		log.Log("Copying resources to %s", movePath)
		for _, include := range includes {
			if com.IsDir(include) {
				os.RemoveAll(path.Join(movePath, include))
				if err := com.CopyDir(include, filepath.Join(movePath, include)); err != nil {
					if setting.LibraryMode {
						errors.AppendError(errors.NewErrCopyResource(include))
					} else {
						log.Error("bin", "Fail to copy following resource:")
						log.Error("", "\t"+include)
					}
				}
			}
		}
	}

	log.Log("Changing work directory back to %s", oldWorkDir)
	os.Chdir(oldWorkDir)

	log.Success("SUCC", "bin", "Command executed successfully!")
	fmt.Println("Binary has been built into: " + movePath)
}
