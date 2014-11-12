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

package doc

import (
	"fmt"
	"go/build"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

// ParseTarget guesses import path of current package
// if target is empty.
func ParseTarget(target string) string {
	if len(target) > 0 {
		return target
	}

	for _, gopath := range base.GetGOPATHs() {
		if strings.HasPrefix(setting.WorkDir, gopath) {
			target = strings.TrimPrefix(setting.WorkDir, path.Join(gopath, "src")+"/")
			log.Info("Guess import path: %s", target)
			return target
		}
	}
	return "."
}

func joinPath(name string, num int) string {
	subdirs := strings.Split(name, "/")
	if len(subdirs) > num {
		return strings.Join(subdirs[:num], "/")
	}
	return name
}

var gopkgPathPattern = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// GetRootPath returns project root path.
func GetRootPath(name string) string {
	for prefix, num := range setting.RootPathPairs {
		if strings.HasPrefix(name, prefix) {
			return joinPath(name, num)
		}
	}

	if strings.HasPrefix(name, "gopkg.in") {
		m := gopkgPathPattern.FindStringSubmatch(strings.TrimPrefix(name, "gopkg.in"))
		if m == nil {
			return name
		}
		user := m[1]
		repo := m[2]
		return path.Join("gopkg.in", user, repo+"."+m[3])
	}
	return name
}

// IsGoRepoPath returns true if package is from standard library.
func IsGoRepoPath(name string) bool {
	return base.IsDir(path.Join(runtime.GOROOT(), "src/pkg", name)) ||
		base.IsDir(path.Join(runtime.GOROOT(), "src", name))
}

// ListImports checks and returns a list of imports of given import path and options.
func ListImports(importPath, rootPath, vendorPath, srcPath, tags string, isTest bool) ([]string, error) {
	oldGOPATH := os.Getenv("GOPATH")
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}

	ctxt := build.Default
	ctxt.BuildTags = strings.Split(tags, " ")
	ctxt.GOPATH = vendorPath + sep + oldGOPATH
	if setting.Debug {
		log.Debug("Import/root path: %s : %s", importPath, rootPath)
		log.Debug("Context GOPATH: %s", ctxt.GOPATH)
		log.Debug("Srouce path: %s", srcPath)
	}
	pkg, err := ctxt.Import(importPath, srcPath, build.AllowBinary)
	if err != nil {
		if _, ok := err.(*build.NoGoError); !ok {
			return nil, fmt.Errorf("fail to get imports(%s): %v", importPath, err)
		}
		log.Warn("Getting imports: %v", err)
	}

	rawImports := pkg.Imports
	numImports := len(rawImports)
	if isTest {
		rawImports = append(rawImports, pkg.TestImports...)
		numImports = len(rawImports)
	}
	imports := make([]string, 0, numImports)
	for _, name := range rawImports {
		if IsGoRepoPath(name) {
			continue
		} else if strings.HasPrefix(name, rootPath) {
			moreImports, err := ListImports(name, rootPath, vendorPath, srcPath, tags, isTest)
			if err != nil {
				return nil, err
			}
			imports = append(imports, moreImports...)
			continue
		}
		if setting.Debug {
			log.Debug("Found dependency: %s", name)
		}
		imports = append(imports, name)
	}
	return imports, nil
}

// GetVcsName checks whether dirPath has .git .hg .svn else return ""
func GetVcsName(dirPath string) string {
	switch {
	case base.IsExist(path.Join(dirPath, ".git")):
		return "git"
	case base.IsExist(path.Join(dirPath, ".hg")):
		return "hg"
	case base.IsExist(path.Join(dirPath, ".svn")):
		return "svn"
	}
	return ""
}
