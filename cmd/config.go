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

	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdConfig = cli.Command{
	Name:  "config",
	Usage: "configurate gopm settings",
	Description: `Command config configurates gopm settings

gopm config set proxy http://<username:password>@server:port
gopm config github [client_id] [client_secret]
`,
	Action:      runConfig,
	Subcommands: configCommands,
	Flags: []cli.Flag{
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

var configCommands = []cli.Command{
	{
		Name:        "set",
		Usage:       "Change gopm settings",
		Action:      runConfigSet,
		Subcommands: configSetCommands,
	},
	{
		Name:   "get",
		Usage:  "Display gopm settings",
		Action: runConfigGet,
	},
	{
		Name:   "unset",
		Usage:  "remove gopm settings",
		Action: runConfigUnset,
	},
}

func runConfig(ctx *cli.Context) {
}

var configSetCommands = []cli.Command{
	{
		Name:  "proxy",
		Usage: "Change HTTP proxy setting",
		Description: `Command proxy changes gopm HTTP proxy setting

gopm config set proxy http://<username:password>@server:port
`,
		Action: runConfigSetProxy,
	},
	{
		Name:  "github",
		Usage: "Change GitHub credentials setting",
		Description: `Command github changes GitHub credentials setting

gopm config set github [client_id] [client_secret]
`,
		Action: runConfigSetGitHub,
	},
}

func runConfigSet(ctx *cli.Context) {

}

func showSettingString(section, key string) {
	fmt.Printf("%s = %s\n", key, setting.Cfg.MustValue(section, key))
}

func runConfigGet(ctx *cli.Context) {
	setup(ctx)
	if len(ctx.Args()) != 1 {
		log.Error("config", "Incorrect number of arguments for command")
		log.Error("", "\t'get' should have 1")
		log.Help("Try 'gopm config get -h' to get more information")
	}
	switch ctx.Args().First() {
	case "proxy":
		fmt.Printf("[%s]\n", "settings")
		showSettingString("settings", "HTTP_PROXY")
	case "github":
		fmt.Printf("[%s]\n", "github")
		showSettingString("github", "CLIENT_ID")
		showSettingString("github", "CLIENT_SECRET")
	}
}

func runConfigUnset(ctx *cli.Context) {
	setup(ctx)
	if len(ctx.Args()) != 1 {
		log.Error("config", "Incorrect number of arguments for command")
		log.Error("", "\t'unset' should have 1")
		log.Help("Try 'gopm config unset -h' to get more information")
	}
	switch ctx.Args().First() {
	case "proxy":
		setting.DeleteConfigOption("settings", "HTTP_PROXY")
	case "github":
		setting.DeleteConfigOption("github", "CLIENT_ID")
		setting.DeleteConfigOption("github", "CLIENT_SECRET")
	}
}

func runConfigSetProxy(ctx *cli.Context) {
	setup(ctx)
	if len(ctx.Args()) != 1 {
		log.Error("config", "Incorrect number of arguments for command")
		log.Error("", "\t'set proxy' should have 1")
		log.Help("Try 'gopm config set help proxy' to get more information")
	}
	setting.SetConfigValue("settings", "HTTP_PROXY", ctx.Args().First())
}

func runConfigSetGitHub(ctx *cli.Context) {
	setup(ctx)
	if len(ctx.Args()) != 2 {
		log.Error("config", "Incorrect number of arguments for command")
		log.Error("", "\t'set github' should have 2")
		log.Help("Try 'gopm config set help github' to get more information")
	}
	setting.SetConfigValue("github", "CLIENT_ID", ctx.Args().First())
	setting.SetConfigValue("github", "CLIENT_SECRET", ctx.Args().Get(1))
}
