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
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/goconfig"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

// setup initializes and checks common environment variables.
func setup(ctx *cli.Context) (err error) {
	setting.Debug = ctx.GlobalBool("debug")
	log.NonColor = ctx.GlobalBool("noterm")
	log.Verbose = ctx.Bool("verbose")

	log.Info("App Version: %s", ctx.App.Version)

	setting.HomeDir, err = base.HomeDir()
	if err != nil {
		return fmt.Errorf("Fail to get home directory: %v", err)
	}
	setting.HomeDir = strings.Replace(setting.HomeDir, "\\", "/", -1)

	setting.InstallRepoPath = path.Join(setting.HomeDir, ".gopm/repos")
	if runtime.GOOS == "windows" {
		setting.IsWindows = true
	}
	os.MkdirAll(setting.InstallRepoPath, os.ModePerm)
	log.Info("Local repository path: %s", setting.InstallRepoPath)

	if !setting.LibraryMode || len(setting.WorkDir) == 0 {
		setting.WorkDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("Fail to get work directory: %v", err)
		}
		setting.WorkDir = strings.Replace(setting.WorkDir, "\\", "/", -1)
	}
	setting.DefaultGopmfile = path.Join(setting.WorkDir, setting.GOPMFILE)
	setting.DefaultVendor = path.Join(setting.WorkDir, setting.VENDOR)
	setting.DefaultVendorSrc = path.Join(setting.DefaultVendor, "src")

	if !ctx.Bool("remote") {
		if ctx.Bool("local") {
			// gf, _, _, err := genGopmfile(ctx)
			// if err != nil {
			// 	return err
			// }
			// setting.InstallGopath = gf.MustValue("project", "local_gopath")
			// if ctx.Command.Name != "gen" {
			// 	if com.IsDir(setting.InstallGopath) {
			// 		log.Log("Indicated local GOPATH: %s", setting.InstallGopath)
			// 		setting.InstallGopath += "/src"
			// 	} else {
			// 		if setting.LibraryMode {
			// 			return fmt.Errorf("Local GOPATH does not exist or is not a directory: %s",
			// 				setting.InstallGopath)
			// 		}
			// 		log.Error("", "Invalid local GOPATH path")
			// 		log.Error("", "Local GOPATH does not exist or is not a directory:")
			// 		log.Error("", "\t"+setting.InstallGopath)
			// 		log.Help("Try 'go help gopath' to get more information")
			// 	}
			// }

		} else {
			// Get GOPATH.
			setting.InstallGopath = base.GetGOPATHs()[0]
			if base.IsDir(setting.InstallGopath) {
				log.Info("Indicated GOPATH: %s", setting.InstallGopath)
				setting.InstallGopath += "/src"
				setting.HasGOPATHSetting = true
			} else {
				if ctx.Bool("gopath") {
					return fmt.Errorf("Local GOPATH does not exist or is not a directory: %s",
						setting.InstallGopath)
				} else {
					// It's OK that no GOPATH setting
					// when user does not specify to use.
					log.Warn("No GOPATH setting available")
				}
			}
		}
	}

	setting.ConfigFile = path.Join(setting.HomeDir, ".gopm/data/gopm.ini")
	if err = setting.LoadConfig(); err != nil {
		return err
	}

	setting.PkgNameListFile = path.Join(setting.HomeDir, ".gopm/data/pkgname.list")
	if err = setting.LoadPkgNameList(); err != nil {
		return err
	}

	setting.LocalNodesFile = path.Join(setting.HomeDir, ".gopm/data/localnodes.list")
	if err = setting.LoadLocalNodes(); err != nil {
		return err
	}
	return nil
}

func parseGopmfile(fileName string) (*goconfig.ConfigFile, string, error) {
	gf, err := setting.LoadGopmfile(fileName)
	if err != nil {
		return nil, "", err
	}
	target := doc.ParseTarget(gf.MustValue("target", "path"))
	if target == "." {
		_, target = filepath.Split(setting.WorkDir)
	}
	return gf, target, nil
}

func autoLink(oldPath, newPath string) error {
	os.MkdirAll(path.Dir(newPath), os.ModePerm)
	return makeLink(oldPath, newPath)
}

func execCmd(gopath, curPath string, args ...string) error {
	oldGopath := os.Getenv("GOPATH")
	log.Info("Setting GOPATH to %s", gopath)

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}

	if err := os.Setenv("GOPATH", gopath+sep+oldGopath); err != nil {
		if setting.LibraryMode {
			return fmt.Errorf("Fail to setting GOPATH: %v", err)
		}
		log.Error("", "Fail to setting GOPATH:")
		log.Fatal("", "\t"+err.Error())
	}
	if setting.HasGOPATHSetting {
		defer func() {
			log.Info("Setting GOPATH back to %s", oldGopath)
			os.Setenv("GOPATH", oldGopath)
		}()
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = curPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info("===== application outputs start =====\n")

	err := cmd.Run()

	log.Info("====== application outputs end ======")
	return err
}

// isSubpackage returns true if given package belongs to current project.
func isSubpackage(rootPath, target string) bool {
	return strings.HasSuffix(setting.WorkDir, rootPath) ||
		strings.HasPrefix(rootPath, target)
}
