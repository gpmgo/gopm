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
func setup(ctx *cli.Context) (err error) {
	setting.Debug = ctx.GlobalBool("debug")
	log.PureMode = ctx.GlobalBool("noterm")
	log.Verbose = ctx.Bool("verbose")

	setting.HomeDir, err = com.HomeDir()
	if err != nil {
		if setting.LibraryMode {
			return fmt.Errorf("Fail to get home directory: %v", err)
		}
		log.Error("setup", "")
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

	if !setting.LibraryMode || len(setting.WorkDir) == 0 {
		setting.WorkDir, err = os.Getwd()
		if err != nil {
			if setting.LibraryMode {
				return fmt.Errorf("Fail to get work directory: %v", err)
			}
			log.Error("setup", "Fail to get work directory:")
			log.Fatal("", "\t"+err.Error())
		}
		setting.WorkDir = strings.Replace(setting.WorkDir, "\\", "/", -1)
	}

	if !ctx.Bool("remote") {
		if ctx.Bool("local") {
			gf, _, _, err := genGopmfile(ctx)
			if err != nil {
				return err
			}
			setting.InstallGopath = gf.MustValue("project", "local_gopath")
			if ctx.Command.Name != "gen" {
				if com.IsDir(setting.InstallGopath) {
					log.Log("Indicated local GOPATH: %s", setting.InstallGopath)
					setting.InstallGopath += "/src"
				} else {
					if setting.LibraryMode {
						return fmt.Errorf("Local GOPATH does not exist or is not a directory: %s",
							setting.InstallGopath)
					}
					log.Error("", "Invalid local GOPATH path")
					log.Error("", "Local GOPATH does not exist or is not a directory:")
					log.Error("", "\t"+setting.InstallGopath)
					log.Help("Try 'go help gopath' to get more information")
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
					if setting.LibraryMode {
						return fmt.Errorf("Local GOPATH does not exist or is not a directory: %s",
							setting.InstallGopath)
					}
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
	if err = setting.LoadLocalNodes(); err != nil {
		return err
	}

	setting.PkgNamesFile = path.Join(setting.HomeDir, ".gopm/data/pkgname.list")
	if err = setting.LoadPkgNameList(); err != nil {
		return err
	}

	setting.ConfigFile = path.Join(setting.HomeDir, ".gopm/data/gopm.ini")
	if err = setting.LoadConfig(); err != nil {
		return err
	}

	if com.IsDir(setting.GOPMFILE) {
		if setting.LibraryMode {
			return fmt.Errorf("gopmfile should be file but found directory")
		}
		log.Error("setup", "Invalid gopmfile:")
		log.Fatal("", "\tit should be file but found directory")
	}

	return doc.SetProxy(setting.HttpProxy)
}

// loadGopmfile loads and returns given gopmfile.
func loadGopmfile(fileName string) (*goconfig.ConfigFile, error) {
	gf, err := goconfig.LoadConfigFile(fileName)
	if err != nil {
		if setting.LibraryMode {
			return nil, fmt.Errorf("Fail to load gopmfile: %v", err)
		}
		log.Error("", "Fail to load gopmfile:")
		log.Fatal("", "\t"+err.Error())
	}
	return gf, nil
}

// saveGopmfile saves gopmfile to given path.
func saveGopmfile(gf *goconfig.ConfigFile, fileName string) error {
	if err := goconfig.SaveConfigFile(gf, fileName); err != nil {
		if setting.LibraryMode {
			return fmt.Errorf("Fail to save gopmfile: %v", err)
		}
		log.Error("", "Fail to save gopmfile:")
		log.Fatal("", "\t"+err.Error())
	}
	return nil
}

// validPkgInfo checks if the information of the package is valid.
func validPkgInfo(info string) (doc.RevisionType, string, error) {
	infos := strings.Split(info, ":")
	tp := doc.RevisionType(infos[0])
	val := infos[1]

	l := len(infos)
	switch {
	case l == 2:
		switch tp {
		case doc.BRANCH, doc.COMMIT, doc.TAG:
		default:
			if setting.LibraryMode {
				return "", "", fmt.Errorf("Invalid node type: %v", tp)
			}
			log.Error("", "Invalid node type:")
			log.Error("", fmt.Sprintf("\t%v", tp))
			log.Help("Try 'gopm help get' to get more information")
		}
		return tp, val, nil
	}

	if setting.LibraryMode {
		return "", "", fmt.Errorf("Cannot parse dependency version: %v", info)
	}
	log.Error("", "Cannot parse dependency version:")
	log.Error("", "\t"+info)
	log.Help("Try 'gopm help get' to get more information")
	return "", "", nil
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

	imports, err := doc.GetImports(target, doc.GetRootPath(target), dirPath, isTest)
	if err != nil {
		return nil, err
	}
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
		return fmt.Errorf("Fail to get gopmfile dependencies: %v", err)
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
				pkgPath := pkg.RootPath + pkg.ValSuffix()
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
						node.IsGetDeps = false
						if err = downloadPackages(target, ctx, []*doc.Node{node}); err != nil {
							return err
						}
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

			curPath = path.Join(doc.VENDOR, "src", pkg.RootPath)
			log.Log("Linking %s", pkg.RootPath+pkg.ValSuffix())
			if err := autoLink(newPath, curPath); err != nil {
				if setting.LibraryMode {
					return fmt.Errorf("Fail to make link dependency: %v", err)
				}
				log.Error("", "Fail to make link dependency:")
				log.Fatal("", "\t"+err.Error())
			}

			if err = getDepPkgs(gf, ctx, pkg.ImportPath, curPath, depPkgs, false); err != nil {
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

func genNewGopath(ctx *cli.Context, isTest bool) (string, string, string, error) {
	log.Trace("Work directory: %s", setting.WorkDir)

	gf, err := loadGopmfile(setting.GOPMFILE)
	if err != nil {
		return "", "", "", err
	}

	// Check dependencies.
	target := doc.ParseTarget(gf.MustValue("target", "path"))
	if target == "." {
		_, target = filepath.Split(setting.WorkDir)
	}

	// Clean old files.
	newGopath := path.Join(setting.WorkDir, doc.VENDOR)
	newGopathSrc := path.Join(newGopath, "src")
	os.RemoveAll(newGopathSrc)
	os.MkdirAll(newGopathSrc, os.ModePerm)

	// Link self.
	targetRoot := doc.GetRootPath(target)
	log.Log("Linking %s", targetRoot)
	if setting.Debug {
		fmt.Println(target)
		fmt.Println(path.Join(strings.TrimSuffix(setting.WorkDir, target), targetRoot))
		fmt.Println(path.Join(newGopathSrc, targetRoot))
	}
	if err := autoLink(path.Join(strings.TrimSuffix(setting.WorkDir, target), targetRoot),
		path.Join(newGopathSrc, targetRoot)); err != nil &&
		!strings.Contains(err.Error(), "file exists") {
		if setting.LibraryMode {
			return "", "", "", fmt.Errorf("Fail to make link self: %v", err)
		}
		log.Error("", "Fail to make link self:")
		log.Fatal("", "\t"+err.Error())
	}

	// Check and loads dependency pakcages.
	depPkgs := make(map[string]*doc.Pkg)
	if err := getDepPkgs(gf, ctx, target, setting.WorkDir, depPkgs, isTest); err != nil {
		if setting.LibraryMode {
			return "", "", "", fmt.Errorf("Fail to get dependency pakcages: %v", err)
		}
		log.Error("", "Fail to get dependency pakcages:")
		log.Fatal("", "\t"+err.Error())
	}

	newCurPath := path.Join(newGopathSrc, target)
	return target, newGopath, newCurPath, nil
}

func execCmd(gopath, curPath string, args ...string) error {
	oldGopath := os.Getenv("GOPATH")
	log.Log("Setting GOPATH to %s", gopath)

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
