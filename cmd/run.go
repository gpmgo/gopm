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
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdRun = cli.Command{
	Name:  "run",
	Usage: "link dependencies and go run",
	Description: `Command run links dependencies according to gopmfile,
and execute 'go run'

gopm run <go run commands>
gopm run -l will recursively find .gopmfile with value localPath 
and run the cmd in the .gopmfile, Windows hasn't supported yet,
you need to run the command right at the local_gopath dir.`,
	Action: runRun,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"local, l", "run command with local gopath context", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
	},
}

func valSuffix(val string) string {
	if len(val) > 0 {
		return "." + val
	}
	return ""
}

// validPkgInfo checks if the information of the package is valid.
func validPkgInfo(info string) (doc.RevisionType, string, error) {
	if len(info) == 0 {
		return doc.BRANCH, "", nil
	}

	infos := strings.Split(info, ":")

	if len(infos) == 2 {
		tp := doc.RevisionType(infos[0])
		val := infos[1]
		switch tp {
		case doc.BRANCH, doc.COMMIT, doc.TAG:
		default:
			return "", "", fmt.Errorf("invalid node type: %v", tp)
		}
		return tp, val, nil
	}
	return "", "", fmt.Errorf("cannot parse dependency version: %v", info)
}

func linkVendors(ctx *cli.Context, optTarget string) error {
	gfPath := path.Join(setting.WorkDir, setting.GOPMFILE)
	gf, target, err := parseGopmfile(gfPath)
	if err != nil {
		return fmt.Errorf("fail to parse gopmfile: %v", err)
	}
	if len(optTarget) > 0 {
		target = optTarget
	}
	rootPath := doc.GetRootPath(target)

	// TODO: local support.

	// Make link of self.
	log.Debug("Linking %s...", rootPath)
	from := setting.WorkDir
	to := path.Join(setting.DefaultVendorSrc, rootPath)
	if setting.Debug {
		log.Debug("Linking from %s to %s", from, to)
	}
	if err := autoLink(from, to); err != nil {
		return fmt.Errorf("fail to link self: %v", err)
	}

	// Check and loads dependency packages.
	log.Debug("Loading dependencies...")
	imports, err := doc.ListImports(target, rootPath, setting.DefaultVendor, setting.WorkDir, ctx.String("tags"), ctx.Bool("test"))
	if err != nil {
		return fmt.Errorf("fail to list imports: %v", err)
	}

	stack := make([]*doc.Pkg, len(imports))
	for i, name := range imports {
		name := doc.GetRootPath(name)
		tp, val, err := validPkgInfo(gf.MustValue("deps", name))
		if err != nil {
			return fmt.Errorf("fail to validate package(%s): %v", name, err)
		}

		stack[i] = doc.NewPkg(name, tp, val)
	}

	// FIXME: at least link once, need a set
	lastIdx := len(stack) - 1
	for lastIdx >= 0 {
		pkg := stack[lastIdx]
		linkPath := path.Join(setting.DefaultVendorSrc, pkg.RootPath)
		if setting.Debug {
			log.Debug("Import path: %s", pkg.ImportPath)
			log.Debug("Linking path: %s", linkPath)
		}
		if base.IsExist(linkPath) {
			stack = stack[:lastIdx]
			lastIdx = len(stack) - 1
			continue
		}

		if pkg.IsEmptyVal() && setting.HasGOPATHSetting &&
			base.IsExist(path.Join(setting.InstallGopath, pkg.RootPath)) {
			stack = stack[:lastIdx]
			lastIdx = len(stack) - 1
			continue
		}

		venderPath := path.Join(setting.InstallRepoPath, pkg.RootPath+pkg.ValSuffix())
		if !base.IsExist(venderPath) {
			return fmt.Errorf("package not installed: %s", pkg.RootPath+pkg.VerSuffix())
		}

		log.Debug("Linking %s...", pkg.RootPath+pkg.ValSuffix())
		if err := autoLink(venderPath, linkPath); err != nil {
			return fmt.Errorf("fail to link dependency(%s): %v", pkg.RootPath, err)
		}
		stack = stack[:lastIdx]

		gf, target, err := parseGopmfile(path.Join(linkPath, setting.GOPMFILE))
		if err != nil {
			return fmt.Errorf("fail to parse gopmfile(%s): %v", linkPath, err)
		}
		// parseGopmfile only returns right target when parse work directory.
		target = pkg.RootPath
		rootPath := target
		imports, err := doc.ListImports(target, rootPath, setting.DefaultVendor, linkPath, ctx.String("tags"), ctx.Bool("test"))
		if err != nil {
			errors.SetError(err)
		}
		for _, name := range imports {
			if name == "C" {
				continue
			}

			name := doc.GetRootPath(name)
			tp, val, err := validPkgInfo(gf.MustValue("deps", name))
			if err != nil {
				return fmt.Errorf("fail to validate package(%s): %v", name, err)
			}

			stack = append(stack, doc.NewPkg(name, tp, val))
		}
		lastIdx = len(stack) - 1
	}
	return nil
}

func runRun(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if err := linkVendors(ctx, ""); err != nil {
		errors.SetError(err)
		return
	}

	log.Info("Running...")

	cmdArgs := []string{"go", "run"}
	if len(ctx.String("tags")) > 0 {
		cmdArgs = append(cmdArgs, "-tags")
		cmdArgs = append(cmdArgs, ctx.String("tags"))
	}
	cmdArgs = append(cmdArgs, ctx.Args()...)
	if err := execCmd(setting.DefaultVendor, setting.WorkDir, cmdArgs...); err != nil {
		errors.SetError(fmt.Errorf("fail to run program: %v", err))
		return
	}

	log.Info("Command executed successfully!")
}
