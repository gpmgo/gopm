// Copyright 2013-2014 gopm authors.
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

	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/doc"
	"github.com/gpmgo/gopm/log"
)

var CmdExec = cli.Command{
	Name:  "exec",
	Usage: "execute go binary in specific version",
	Description: `Command exec builds and executes go binary in specific version 
based on the setting of your gopmfile or argument

gopm exec <import path|package name>
gopm exec <import path>@[<tag|commit|branch>:<value>]
gopm exec <package name>@[<tag|commit|branch>:<value>]

Can only specify one each time, and only works for projects that 
contain main package`,
	Action: runExec,
	Flags: []cli.Flag{
		cli.BoolFlag{"update, u", "update pakcage(s) and dependencies if any"},
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

func runExec(ctx *cli.Context) {
	setup(ctx)

	if len(ctx.Args()) == 0 {
		log.Error("exec", "Cannot start command:")
		log.Fatal("", "\tNo package specified")
	}

	installRepoPath = doc.HomeDir + "/repos"
	log.Log("Local repository path: %s", installRepoPath)

	// Parse package version.
	info := ctx.Args()[0]
	pkgPath := info
	node := doc.NewNode(pkgPath, pkgPath, doc.BRANCH, "", true)
	if i := strings.Index(info, "@"); i > -1 {
		// Specify version by argument.
		pkgPath = info[:i]
		node.Type, node.Value = validPath(info[i+1:])
	}

	// Check package name.
	if !strings.Contains(pkgPath, "/") {
		pkgPath = doc.GetPkgFullPath(pkgPath)
	}

	node.ImportPath = pkgPath
	node.DownloadURL = pkgPath

	if len(node.Value) == 0 && com.IsFile(".gopmfile") {
		// Specify version by gopmfile.
		gf := doc.NewGopmfile(".")
		info, err := gf.GetValue("exec", pkgPath)
		if err == nil && len(info) > 0 {
			node.Type, node.Value = validPath(info)
		}
	}

	// Check if binary exists.
	binPath := path.Join(doc.HomeDir, "bins", node.ImportPath, path.Base(node.ImportPath)+versionSuffix(node.Value))
	log.Log("Binary path: %s", binPath)

	if !com.IsFile(binPath) {
		// Calling bin command.
		args := []string{"bin", "-d"}
		if ctx.Bool("verbose") {
			args = append(args, "-v")
		}
		if ctx.Bool("update") {
			args = append(args, "-u")
		}
		args = append(args, node.ImportPath+"@"+node.Type+":"+node.Value)
		args = append(args, path.Dir(binPath))
		stdout, stderr, err := com.ExecCmd("gopm", args...)
		if err != nil {
			log.Error("exec", "Building binary failed:")
			log.Fatal("", "\t"+err.Error())
		}
		if len(stderr) > 0 {
			fmt.Print(stderr)
		}
		if len(stdout) > 0 {
			fmt.Print(stdout)
		}
	}
	fmt.Printf("%+v\n", node)

	return
	stdout, stderr, err := com.ExecCmd(binPath, ctx.Args()[1:]...)
	if err != nil {
		log.Error("exec", "Calling binary failed:")
		log.Fatal("", "\t"+err.Error())
	}
	if len(stderr) > 0 {
		fmt.Print(stderr)
	}
	if len(stdout) > 0 {
		fmt.Print(stdout)
	}
}
