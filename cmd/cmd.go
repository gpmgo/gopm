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
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

// setup initializes and checks common environment variables.
func setup(ctx *cli.Context) {
	setting.Debug = ctx.GlobalBool("debug")
	log.PureMode = ctx.GlobalBool("noterm")
	log.Verbose = ctx.Bool("verbose")

	var err error
	setting.HomeDir, err = com.HomeDir()
	if err != nil {
		log.Error("setup", "Fail to get home directory:")
		log.Fatal("", "\t"+err.Error())
	}
	setting.HomeDir = strings.Replace(setting.HomeDir, "\\", "/", -1)

	setting.InstallRepoPath = path.Join(setting.HomeDir, ".gopm/repos")
	if runtime.GOOS == "windows" {
		setting.IsWindows = true
		setting.InstallRepoPath = path.Join(setting.InstallRepoPath, "src")
	}
	os.MkdirAll(setting.InstallRepoPath, os.ModePerm)
	log.Log("Local repository path: %s", setting.InstallRepoPath)

	setting.WorkDir, err = os.Getwd()
	if err != nil {
		log.Error("setup", "Fail to get work directory:")
		log.Fatal("", "\t"+err.Error())
	}
	setting.WorkDir = strings.Replace(setting.WorkDir, "\\", "/", -1)

	if !ctx.Bool("remote") {
		if ctx.Bool("local") {
			gf, _, _ := genGopmfile()
			setting.InstallGopath = gf.MustValue("project", "local_gopath")
			if ctx.Command.Name != "gen" {
				if com.IsDir(setting.InstallGopath) {
					log.Log("Indicated local GOPATH: %s", setting.InstallGopath)
					setting.InstallGopath += "/src"
				} else {
					log.Error("", "Invalid local GOPATH path")
					log.Error("", "Local GOPATH does not exist or is not a directory:")
					log.Fatal("", "\t"+setting.InstallGopath)
				}
			}

		} else {
			// Get GOPATH.
			setting.InstallGopath = com.GetGOPATHs()[0]
			if com.IsDir(setting.InstallGopath) {
				log.Log("Indicated GOPATH: %s", setting.InstallGopath)
				setting.InstallGopath += "/src"
			} else {
				if ctx.Bool("gopath") {
					log.Error("", "Invalid GOPATH path")
					log.Error("", "GOPATH does not exist or is not a directory:")
					log.Error("", "\t"+setting.InstallGopath)
					log.Help("Try 'go help gopath' to get more information")
				} else {
					// It's OK that no GOPATH setting
					// when user does not specify to use.
					log.Warn("No GOPATH setting available")
				}
			}
		}
	}

	setting.GopmTempPath = path.Join(setting.HomeDir, ".gopm/temp")

	setting.LocalNodesFile = path.Join(setting.HomeDir, ".gopm/data/localnodes.list")
	setting.LoadLocalNodes()

	setting.PkgNamesFile = path.Join(setting.HomeDir, ".gopm/data/pkgname.list")
	setting.LoadPkgNameList()

	setting.ConfigFile = path.Join(setting.HomeDir, ".gopm/data/gopm.ini")
	setting.LoadConfig()

	if com.IsDir(setting.GOPMFILE) {
		log.Error("setup", "Invalid gopmfile:")
		log.Fatal("", "\tit should be file but found directory")
	}

	doc.SetProxy(setting.HttpProxy)
}

// loadGopmfile loads and returns given gopmfile.
func loadGopmfile(fileName string) *goconfig.ConfigFile {
	gf, err := goconfig.LoadConfigFile(fileName)
	if err != nil {
		log.Error("", "Fail to load gopmfile:")
		log.Fatal("", "\t"+err.Error())
	}
	return gf
}

// saveGopmfile saves gopmfile to given path.
func saveGopmfile(gf *goconfig.ConfigFile, fileName string) {
	if err := goconfig.SaveConfigFile(gf, fileName); err != nil {
		log.Error("", "Fail to save gopmfile:")
		log.Fatal("", "\t"+err.Error())
	}
}

// validPkgInfo checks if the information of the package is valid.
func validPkgInfo(info string) (doc.RevisionType, string) {
	infos := strings.Split(info, ":")
	tp := doc.RevisionType(infos[0])
	val := infos[1]

	l := len(infos)
	switch {
	case l == 2:
		switch tp {
		case doc.BRANCH, doc.COMMIT, doc.TAG:
		default:
			log.Error("", "Invalid node type:")
			log.Error("", fmt.Sprintf("\t%v", tp))
			log.Help("Try 'gopm help get' to get more information")
		}
		return tp, val
	}

	log.Error("", "Cannot parse dependency version:")
	log.Error("", "\t"+info)
	log.Help("Try 'gopm help get' to get more information")
	return "", ""
}

// isSubpackage returns true if given package belongs to current project.
func isSubpackage(rootPath, target string) bool {
	return strings.HasSuffix(setting.WorkDir, rootPath) ||
		strings.HasPrefix(rootPath, target)
}

func getGopmPkgs(
	gf *goconfig.ConfigFile,
	target, dirPath string,
	isTest bool) (pkgs map[string]*doc.Pkg, err error) {

	var deps map[string]string
	if deps, err = gf.GetSection("deps"); err != nil {
		deps = nil
	}

	imports := doc.GetImports(target, doc.GetRootPath(target), dirPath, isTest)
	pkgs = make(map[string]*doc.Pkg)
	for _, name := range imports {
		if name == "C" {
			continue
		}

		if !doc.IsGoRepoPath(name) {
			if deps != nil {
				if info, ok := deps[name]; ok {
					// Check version. there should chek
					// local first because d:\ contains :
					if com.IsDir(info) {
						pkgs[name] = doc.NewPkg(name, doc.LOCAL, info)
						continue
					} else if i := strings.Index(info, ":"); i > -1 {
						pkgs[name] = doc.NewPkg(name, doc.RevisionType(info[:i]), info[i+1:])
						continue
					}
				}
			}
			pkgs[name] = doc.NewDefaultPkg(name)
		}
	}
	return pkgs, nil
}

func getDepPkgs(
	gf *goconfig.ConfigFile,
	ctx *cli.Context,
	target, curPath string,
	depPkgs map[string]*doc.Pkg,
	isTest bool) error {

	if setting.Debug {
		log.Trace("Current Path: %s", curPath)
	}

	pkgs, err := getGopmPkgs(gf, target, curPath, isTest)
	if err != nil {
		return fmt.Errorf("fail to get gopmfile dependencies: %v", err)
	}

	if setting.Debug {
		for _, pkg := range pkgs {
			if _, ok := depPkgs[pkg.RootPath]; !ok {
				log.Trace("Found new dependency: %s", pkg.ImportPath)
			}
		}
	}

	for name, pkg := range pkgs {
		if _, ok := depPkgs[pkg.RootPath]; !ok {
			var newPath string
			if !build.IsLocalImport(name) && pkg.Type != doc.LOCAL {
				pkgPath := strings.Replace(
					pkg.ImportPath, pkg.RootPath, pkg.RootPath+pkg.ValSuffix(), 1)
				newPath = path.Join(setting.InstallRepoPath, pkgPath)
				if len(pkg.ValSuffix()) == 0 && !ctx.Bool("remote") &&
					com.IsDir(path.Join(setting.InstallGopath, pkgPath)) {
					newPath = path.Join(setting.InstallGopath, pkgPath)
				}
				if target != "" && strings.HasPrefix(pkg.ImportPath, target) {
					newPath = path.Join(curPath, strings.TrimPrefix(pkg.ImportPath, target))
				} else {
					if !com.IsExist(newPath) || ctx.Bool("update") {
						node := doc.NewNode(pkg.ImportPath, pkg.Type, pkg.Value, true)
						downloadPackages(target, ctx, []*doc.Node{node})
					}
				}
			} else {
				if pkg.Type == doc.LOCAL {
					newPath, err = filepath.Abs(pkg.Value)
				} else {
					newPath, err = filepath.Abs(name)
				}
				if err != nil {
					return err
				}
			}
			depPkgs[pkg.RootPath] = pkg
			if err = getDepPkgs(gf, ctx, pkg.ImportPath, newPath, depPkgs, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func autoLink(oldPath, newPath string) error {
	os.MkdirAll(path.Dir(newPath), os.ModePerm)
	return makeLink(oldPath, newPath)
}

func genNewGopath(ctx *cli.Context, isTest bool) (string, string, string) {
	log.Trace("Work directory: %s", setting.WorkDir)

	gf, target, _ := genGopmfile()
	if target == "." {
		_, target = filepath.Split(setting.WorkDir)
	}

	// Check and loads dependency pakcages.
	depPkgs := make(map[string]*doc.Pkg)
	if err := getDepPkgs(gf, ctx, target, setting.WorkDir, depPkgs, isTest); err != nil {
		log.Error("", "Fail to get dependency pakcages:")
		log.Fatal("", "\t"+err.Error())
	}

	// Clean old files.
	newGopath := path.Join(setting.WorkDir, doc.VENDOR)
	newGopathSrc := path.Join(newGopath, "src")
	os.RemoveAll(newGopathSrc)
	os.MkdirAll(newGopathSrc, os.ModePerm)

	for name, pkg := range depPkgs {
		var oldPath string
		if pkg.Type == doc.LOCAL {
			oldPath, _ = filepath.Abs(pkg.Value)
		} else {
			oldPath = path.Join(setting.InstallRepoPath, name) + pkg.ValSuffix()
		}

		newPath := path.Join(newGopathSrc, name)
		paths := strings.Split(name, "/")
		var isExist, isCurChild bool
		if name == target {
			continue
		}

		for i := 0; i < len(paths)-1; i++ {
			pName := strings.Join(paths[:len(paths)-1-i], "/")
			if _, ok := depPkgs[pName]; ok {
				isExist = true
				break
			}
			if target == pName {
				isCurChild = true
				break
			}
		}
		if isCurChild {
			continue
		}

		if !isExist && (!pkg.IsEmptyVal() || ctx.Bool("remote") ||
			!com.IsDir(path.Join(setting.InstallGopath, pkg.ImportPath))) {

			log.Log("Linking %s", name+pkg.ValSuffix())
			if err := autoLink(oldPath, newPath); err != nil {
				log.Error("", "Fail to make link dependency:")
				log.Fatal("", "\t"+err.Error())
			}
		}
	}

	targetRoot := doc.GetRootPath(target)
	newCurPath := path.Join(newGopathSrc, target)
	log.Log("Linking %s", targetRoot)
	if setting.Debug {
		fmt.Println(target)
		fmt.Println(path.Join(strings.TrimSuffix(setting.WorkDir, target), targetRoot))
		fmt.Println(path.Join(newGopathSrc, targetRoot))
	}
	if err := autoLink(path.Join(
		strings.TrimSuffix(setting.WorkDir, target), targetRoot),
		path.Join(newGopathSrc, targetRoot)); err != nil &&
		!strings.Contains(err.Error(), "file exists") {
		log.Error("", "Fail to make link self:")
		log.Fatal("", "\t"+err.Error())
	}
	return target, newGopath, newCurPath
}

func execCmd(gopath, curPath string, args ...string) error {
	oldGopath := os.Getenv("GOPATH")
	log.Log("Setting GOPATH to %s", gopath)

	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}

	if err := os.Setenv("GOPATH", gopath+sep+oldGopath); err != nil {
		log.Error("", "Fail to setting GOPATH:")
		log.Fatal("", "\t"+err.Error())
	}
	defer func() {
		log.Log("Setting GOPATH back to %s", oldGopath)
		os.Setenv("GOPATH", oldGopath)
	}()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = curPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Log("===== application outputs start =====\n")

	err := cmd.Run()

	log.Log("====== application outputs end ======")
	return err
}
