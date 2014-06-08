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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/doc"
	"github.com/gpmgo/gopm/modules/errors"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var CmdUpdate = cli.Command{
	Name:  "update",
	Usage: "check and update gopm resources including itself",
	Description: `Command update checks updates of resources and gopm itself.

gopm update

Resources will be updated automatically after executed this command,
but you have to confirm before updaing gopm itself.`,
	Action: runUpdate,
	Flags: []cli.Flag{
		cli.BoolFlag{"verbose, v", "show process details"},
	},
}

type version struct {
	Gopm            int `json:"gopm"`
	PackageNameList int `json:"package_name_list"`
}

func loadLocalVerInfo() (ver version) {
	verPath := path.Join(setting.HomeDir, setting.VERINFO)

	// First time run should not exist.
	if !com.IsExist(verPath) {
		return ver
	}

	f, err := os.Open(verPath)
	if err != nil {
		log.Error("Update", "Fail to open VERSION.json")
		log.Fatal("", err.Error())
	}

	if err := json.NewDecoder(f).Decode(&ver); err != nil {
		log.Error("Update", "Fail to decode VERSION.json")
		log.Fatal("", err.Error())
	}
	return ver
}

func runUpdate(ctx *cli.Context) {
	if setting.LibraryMode {
		errors.SetError(fmt.Errorf("Library mode does not support update command"))
		return
	}
	setup(ctx)

	isAnythingUpdated := false
	localVerInfo := loadLocalVerInfo()

	// Get remote version info.
	var remoteVerInfo version
	if err := com.HttpGetJSON(doc.HttpClient, "http://gopm.io/VERSION.json", &remoteVerInfo); err != nil {
		log.Error("Update", "Fail to fetch VERSION.json")
		log.Fatal("", err.Error())
	}

	// Package name list.
	if remoteVerInfo.PackageNameList > localVerInfo.PackageNameList {
		log.Log("Updating pkgname.list...%v > %v",
			localVerInfo.PackageNameList, remoteVerInfo.PackageNameList)
		data, err := com.HttpGetBytes(doc.HttpClient, "https://raw2.github.com/gpmgo/docs/master/pkgname.list", nil)
		if err != nil {
			log.Error("Update", "Fail to update pkgname.list")
			log.Fatal("", err.Error())
		}

		if err = com.WriteFile(path.Join(setting.HomeDir, setting.PkgNamesFile), data); err != nil {
			log.Error("Update", "Fail to save pkgname.list")
			log.Fatal("", err.Error())
		}
		log.Log("Update pkgname.list to %v succeed!", remoteVerInfo.PackageNameList)
		isAnythingUpdated = true
	}

	// Gopm.
	if remoteVerInfo.Gopm != setting.VERSION {
		log.Log("Updating gopm...%v > %v", setting.VERSION, remoteVerInfo.Gopm)

		tmpBinPath := path.Join(setting.GopmTempPath, "gopm")
		if runtime.GOOS == "windows" {
			tmpBinPath += ".exe"
		}

		os.MkdirAll(path.Dir(tmpBinPath), os.ModePerm)
		os.Remove(tmpBinPath)

		// Fetch code.
		args := []string{"bin", "-u", "-r", "-d=" + setting.GopmTempPath}
		if ctx.Bool("verbose") {
			args = append(args, "-v")
		}
		args = append(args, "github.com/gpmgo/gopm")
		stdout, stderr, err := com.ExecCmd("gopm", args...)
		if err != nil {
			log.Error("Update", "Fail to execute 'bin -u -r -d=/Users/jiahuachen/.gopm/temp -v github.com/gpmgo/gopm")
			log.Fatal("", "\t"+stderr)
		}
		if len(stdout) > 0 {
			fmt.Print(stdout)
		}

		// Check if previous steps were successful.
		if !com.IsExist(tmpBinPath) {
			log.Error("Update", "Fail to continue command")
			log.Fatal("", "Previous steps weren't successful, no binary produced")
		}

		movePath := path.Join(setting.WorkDir, path.Base(os.Args[0]))
		log.Log("New binary will be replaced for %s", movePath)
		// Move binary to given directory.
		if runtime.GOOS != "windows" {
			err := os.Rename(tmpBinPath, movePath)
			if err != nil {
				log.Error("Update", "Fail to move binary")
				log.Fatal("", err.Error())
			}
			os.Chmod(movePath+"/"+path.Base(tmpBinPath), os.ModePerm)
		} else {
			batPath := path.Join(setting.GopmTempPath, "update.bat")
			f, err := os.Create(batPath)
			if err != nil {
				log.Error("Update", "Fail to generate bat file")
				log.Fatal("", err.Error())
			}
			f.WriteString("@echo off\r\n")
			f.WriteString(fmt.Sprintf("ping -n 1 127.0.0.1>nul\r\ncopy \"%v\" \"%v\" >nul\r\ndel \"%v\" >nul\r\n\r\n",
				tmpBinPath, movePath, tmpBinPath))
			f.Close()

			attr := &os.ProcAttr{
				Dir:   setting.WorkDir,
				Env:   os.Environ(),
				Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			}

			if _, err = os.StartProcess(batPath, []string{batPath}, attr); err != nil {
				log.Error("Update", "Fail to start bat process")
				log.Fatal("", err.Error())
			}
		}

		log.Success("SUCC", "Update", "Command execute successfully!")
		isAnythingUpdated = true
	}

	if isAnythingUpdated {
		// Save JSON.
		f, err := os.Create(setting.VERINFO)
		if err != nil {
			log.Error("Update", "Fail to create VERSION.json")
			log.Fatal("", err.Error())
		}
		if err := json.NewEncoder(f).Encode(&remoteVerInfo); err != nil {
			log.Error("Update", "Fail to encode VERSION.json")
			log.Fatal("", err.Error())
		}
	} else {
		log.Log("Nothing need to be updated")
	}
	log.Log("Exit old gopm")
}
