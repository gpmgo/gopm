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
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/goconfig"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdGet = cli.Command{
	Name:  "get",
	Usage: "fetch remote package(s) and dependencies",
	Description: `Command get fetches a package or packages,
and any package that it or they depend(s) on.
If the package has a gopmfile, the fetch process will be driven by that.

gopm get
gopm get <import path>@[<tag|commit|branch>:<value>]
gopm get <package name>@[<tag|commit|branch>:<value>]

Can specify one or more: gopm get cli@tag:v1.2.0 github.com/Unknwon/macaron

If no version specified and package exists in GOPATH,
it will be skipped, unless user enabled '--remote, -r' option
then all the packages go into gopm local repository.`,
	Action: runGet,
	Flags: []cli.Flag{
		cli.StringFlag{"tags", "", "apply build tags", ""},
		cli.BoolFlag{"download, d", "download given package only", ""},
		cli.BoolFlag{"update, u", "update package(s) and dependencies if any", ""},
		cli.BoolFlag{"local, l", "download all packages to local GOPATH", ""},
		cli.BoolFlag{"gopath, g", "download all packages to GOPATH", ""},
		cli.BoolFlag{"remote, r", "download all packages to gopm local repository", ""},
		cli.BoolFlag{"verbose, v", "show process details", ""},
		cli.BoolFlag{"save, s", "save dependency to gopmfile", ""},
	},
}

var (
	// Saves packages that have been downloaded.
	downloadCache = base.NewSafeMap()
	skipCache     = base.NewSafeMap()
	copyCache     = base.NewSafeMap()
	downloadCount int
	failCount     int
)

// downloadPackage downloads package either use version control tools or not.
func downloadPackage(ctx *cli.Context, n *doc.Node) (*doc.Node, []string, error) {

	// fmt.Println(n.VerString())
	log.Info("Downloading package: %s", n.VerString())
	downloadCache.Set(n.VerString())

	vendor := base.GetTempDir()
	defer os.RemoveAll(vendor)

	var (
		err     error
		imports []string
		srcPath string
	)

	// Check if only need to use VCS tools.
	vcs := doc.GetVcsName(n.InstallGopath)
	// If update, gopath and VCS tools set then use VCS tools to update the package.
	if ctx.Bool("update") && (ctx.Bool("gopath") || ctx.Bool("local")) && len(vcs) > 0 {
		if err = n.UpdateByVcs(vcs); err != nil {
			return nil, nil, fmt.Errorf("fail to update by VCS(%s): %v", n.ImportPath, err)
		}
		srcPath = n.InstallGopath
	} else {
		if !n.IsGetDepsOnly || !n.IsExist() {
			// Get revision value from local records.
			n.Revision = setting.LocalNodes.MustValue(n.RootPath, "value")
			if err = n.DownloadGopm(ctx); err != nil {
				errors.AppendError(errors.NewErrDownload(n.ImportPath + ": " + err.Error()))
				failCount++
				os.RemoveAll(n.InstallPath)
				return nil, nil, nil
			}
		}
		srcPath = n.InstallPath
	}

	if n.IsGetDeps {
		imports, err = getDepList(ctx, n.ImportPath, srcPath, vendor)
		if err != nil {
			return nil, nil, fmt.Errorf("fail to list imports(%s): %v", n.ImportPath, err)
		}
		if setting.Debug {
			log.Debug("New imports: %v", imports)
		}
	}
	return n, imports, err
}

// downloadPackages downloads packages with certain commit,
// if the commit is empty string, then it downloads all dependencies,
// otherwise, it only downloada package with specific commit only.
func downloadPackages(target string, ctx *cli.Context, nodes []*doc.Node) (err error) {
	for _, n := range nodes {
		// Check if it is a valid remote path or C.
		if n.ImportPath == "C" {
			continue
		} else if !base.IsValidRemotePath(n.ImportPath) {
			// Invalid import path.
			if setting.LibraryMode {
				errors.AppendError(errors.NewErrInvalidPackage(n.VerString()))
			}
			log.Error("Skipped invalid package: " + n.VerString())
			failCount++
			continue
		}

		// Valid import path.
		if isSubpackage(n.RootPath, target) {
			continue
		}

		// Indicates whether need to download package or update.
		if n.IsFixed() && n.IsExist() {
			n.IsGetDepsOnly = true
		}

		if downloadCache.Get(n.VerString()) {
			if !skipCache.Get(n.VerString()) {
				skipCache.Set(n.VerString())
				log.Debug("Skipped downloaded package: %s", n.VerString())
			}
			continue
		}

		if !ctx.Bool("update") {
			// Check if package has been downloaded.
			if n.IsExist() {
				if !skipCache.Get(n.VerString()) {
					skipCache.Set(n.VerString())
					log.Info("%s", n.InstallPath)
					log.Debug("Skipped installed package: %s", n.VerString())
				}

				// Only copy when no version control.
				if !copyCache.Get(n.VerString()) && (ctx.Bool("gopath") || ctx.Bool("local")) {
					copyCache.Set(n.VerString())
					if err = n.CopyToGopath(); err != nil {
						return err
					}
				}
				continue
			} else {
				setting.LocalNodes.SetValue(n.RootPath, "value", "")
			}
		}
		// Download package.
		nod, imports, err := downloadPackage(ctx, n)
		if err != nil {
			return err
		}
		for _, name := range imports {
			var gf *goconfig.ConfigFile
			gfPath := path.Join(n.InstallPath, setting.GOPMFILE)

			// Check if has gopmfile.
			if base.IsFile(gfPath) {
				log.Info("Found gopmfile: %s", n.VerString())
				var err error
				gf, _, err = parseGopmfile(gfPath)
				if err != nil {
					return fmt.Errorf("fail to parse gopmfile(%s): %v", gfPath, err)
				}
			}

			// Need to download dependencies.
			// Generate temporary nodes.
			nodes := make([]*doc.Node, len(imports))
			for i := range nodes {
				nodes[i] = doc.NewNode(name, doc.BRANCH, "", !ctx.Bool("download"))

				if gf == nil {
					continue
				}

				// Check if user specified the version.
				if v := gf.MustValue("deps", imports[i]); len(v) > 0 {
					nodes[i].Type, nodes[i].Value, err = validPkgInfo(v)
					if err != nil {
						return err
					}
				}
			}
			if err = downloadPackages(target, ctx, nodes); err != nil {
				return err
			}
		}

		// Only save package information with specific commit.
		if nod == nil {
			continue
		}

		// Save record in local nodes.
		log.Info("Got %s", n.VerString())
		downloadCount++

		// Only save non-commit node.
		if nod.IsEmptyVal() && len(nod.Revision) > 0 {
			setting.LocalNodes.SetValue(nod.RootPath, "value", nod.Revision)
		}

		// If update set downloadPackage will use VSC tools to download the package,
		// else just download to local repository and copy to GOPATH.
		if !nod.HasVcs() && !copyCache.Get(n.RootPath) && (ctx.Bool("gopath") || ctx.Bool("local")) {
			copyCache.Set(n.RootPath)
			if err = nod.CopyToGopath(); err != nil {
				return err
			}
		}
	}
	return nil
}

func getPackages(target string, ctx *cli.Context, nodes []*doc.Node) error {
	if err := downloadPackages(target, ctx, nodes); err != nil {
		return err
	}
	if err := setting.SaveLocalNodes(); err != nil {
		return err
	}

	log.Info("%d package(s) downloaded, %d failed", downloadCount, failCount)
	if ctx.GlobalBool("strict") && failCount > 0 && !setting.LibraryMode {
		return fmt.Errorf("fail to download some packages")
	}
	return nil
}

func getByGopmfile(ctx *cli.Context) error {
	// Make sure gopmfile exists and up-to-date.
	gf, target, err := parseGopmfile(setting.GOPMFILE)
	if err != nil {
		return err
	}

	imports, err := getDepList(ctx, target, setting.WorkDir, setting.DefaultVendor)
	if err != nil {
		return err
	}

	// Check if dependency has version.
	nodes := make([]*doc.Node, 0, len(imports))
	for _, name := range imports {
		name = doc.GetRootPath(name)
		n := doc.NewNode(name, doc.BRANCH, "", !ctx.Bool("download"))

		// Check if user specified the version.
		if v := gf.MustValue("deps", name); len(v) > 0 {
			n.Type, n.Value, err = validPkgInfo(v)
			n = doc.NewNode(name, n.Type, n.Value, !ctx.Bool("download"))
		}
		nodes = append(nodes, n)
	}

	return getPackages(target, ctx, nodes)
}

func getByPaths(ctx *cli.Context) error {
	nodes := make([]*doc.Node, 0, len(ctx.Args()))
	for _, info := range ctx.Args() {
		pkgPath := info
		n := doc.NewNode(pkgPath, doc.BRANCH, "", !ctx.Bool("download"))

		if i := strings.Index(info, "@"); i > -1 {
			pkgPath = info[:i]
			tp, val, err := validPkgInfo(info[i+1:])
			if err != nil {
				return err
			}
			n = doc.NewNode(pkgPath, tp, val, !ctx.Bool("download"))
		}

		// Check package name.
		if !strings.Contains(pkgPath, "/") {
			tmpPath, err := setting.GetPkgFullPath(pkgPath)
			if err != nil {
				return err
			}
			if tmpPath != pkgPath {
				n = doc.NewNode(tmpPath, n.Type, n.Value, n.IsGetDeps)
			}
		}
		nodes = append(nodes, n)
	}
	return getPackages(".", ctx, nodes)
}

func runGet(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	// Check option conflicts.
	hasConflict := false
	names := ""
	switch {
	case ctx.Bool("local") && ctx.Bool("gopath"):
		hasConflict = true
		names = "'--local, -l' and '--gopath, -g'"
	case ctx.Bool("local") && ctx.Bool("remote"):
		hasConflict = true
		names = "'--local, -l' and '--remote, -r'"
	case ctx.Bool("gopath") && ctx.Bool("remote"):
		hasConflict = true
		names = "'--gopath, -g' and '--remote, -r'"
	}
	if hasConflict {
		errors.SetError(fmt.Errorf("Command options have conflicts: %s", names))
		return
	}

	var err error
	// Check number of arguments to decide which function to call.
	if len(ctx.Args()) == 0 {
		if ctx.Bool("download") {
			errors.SetError(fmt.Errorf("Not enough arguments for option: '--download, -d'"))
			return
		}
		err = getByGopmfile(ctx)
	} else {
		err = getByPaths(ctx)
	}
	if err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) > 0 && ctx.Bool("save") {
		gf, _, err := parseGopmfile(setting.GOPMFILE)
		if err != nil {
			errors.SetError(err)
			return
		}

		for _, info := range ctx.Args() {
			if i := strings.Index(info, "@"); i > -1 {
				gf.SetValue("deps", info[:i], info[i+1:])
			} else {
				gf.SetValue("deps", info, "")
			}
		}
		setting.SaveGopmfile(gf, setting.GOPMFILE)
	}
}
