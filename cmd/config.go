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

	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdConfig = cli.Command{
	Name:  "config",
	Usage: "configure gopm settings",
	Description: `Command config configures gopm settings

gopm config set proxy http://<username:password>@server:port
gopm config github [client_id] [client_secret]
`,
	Action:      runConfig,
	Subcommands: configCommands,
	Flags: []cli.Flag{
		cli.BoolFlag{"verbose, v", "show process details", ""},
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
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 1 {
		errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 1"))
		return
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
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 1 {
		errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 1"))
		return
	}

	var err error
	switch ctx.Args().First() {
	case "proxy":
		err = setting.DeleteConfigOption("settings", "HTTP_PROXY")
	case "github":
		if err = setting.DeleteConfigOption("github", "CLIENT_ID"); err != nil {
			errors.SetError(err)
			return
		}
		err = setting.DeleteConfigOption("github", "CLIENT_SECRET")
	}
	if err != nil {
		errors.SetError(err)
		return
	}
}

func runConfigSetProxy(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 1 {
		errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 1"))
		return
	}
	if err := setting.SetConfigValue("settings", "HTTP_PROXY", ctx.Args().First()); err != nil {
		errors.SetError(err)
		return
	}
}

func runConfigSetGitHub(ctx *cli.Context) {
	if err := setup(ctx); err != nil {
		errors.SetError(err)
		return
	}

	if len(ctx.Args()) != 2 {
		errors.SetError(fmt.Errorf("Incorrect number of arguments for command: should have 2"))
		return
	}
	if err := setting.SetConfigValue("github", "CLIENT_ID", ctx.Args().First()); err != nil {
		errors.SetError(err)
		return
	}
	if err := setting.SetConfigValue("github", "CLIENT_SECRET", ctx.Args().Get(1)); err != nil {
		errors.SetError(err)
		return
	}
}
