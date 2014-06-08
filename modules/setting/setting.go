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

package setting

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"

	"github.com/gpmgo/gopm/modules/log"
)

const (
	GOPMFILE = ".gopmfile"
	VERINFO  = "data/VERSION.json"

	VERSION = 201406081
)

type Error struct {
	HasError bool
	Fatal    error
	Errors   []error
}

var (
	Debug bool

	HomeDir         string
	WorkDir         string // The path of gopm was executed.
	InstallRepoPath string // The gopm local repository.
	InstallGopath   string

	GopmTempPath string

	LocalNodesFile string
	LocalNodes     *goconfig.ConfigFile

	PkgNamesFile    string
	PackageNameList = make(map[string]string)

	ConfigFile string
	Cfg        *goconfig.ConfigFile

	IsWindows   bool
	IsWindowsXP bool

	LibraryMode   bool
	RuntimeError  = new(Error)
	RootPathPairs = map[string]int{
		"github.com":      3,
		"code.google.com": 3,
		"bitbucket.org":   3,
		"git.oschina.net": 3,
		"gitcafe.com":     3,
		"launchpad.net":   2,
		"labix.org":       3,
	}
	CommonRes = []string{"views", "templates", "static", "public", "conf"}
)

func LoadLocalNodes() (err error) {
	if !com.IsFile(LocalNodesFile) {
		os.MkdirAll(path.Dir(LocalNodesFile), os.ModePerm)
		os.Create(LocalNodesFile)
	}

	LocalNodes, err = goconfig.LoadConfigFile(LocalNodesFile)
	if err != nil {
		if LibraryMode {
			return fmt.Errorf("Fail to load localnodes.list: %v", err)
		}
		log.Error("", "Fail to load localnodes.list:")
		log.Fatal("", "\t"+err.Error())
	}
	return nil
}

func SaveLocalNodes() error {
	if err := goconfig.SaveConfigFile(LocalNodes, LocalNodesFile); err != nil {
		if LibraryMode {
			return fmt.Errorf("Fail to save localnodes.list: %v", err)
		}
		log.Error("", "Fail to save localnodes.list:")
		log.Error("", "\t"+err.Error())
	}
	return nil
}

func LoadPkgNameList() error {
	if !com.IsFile(PkgNamesFile) {
		return nil
	}

	data, err := ioutil.ReadFile(PkgNamesFile)
	if err != nil {
		if LibraryMode {
			return fmt.Errorf("Fail to load pkgname.list: %v", err)
		}
		log.Error("", "Fail to load pkgname.list:")
		log.Fatal("", "\t"+err.Error())
	}

	pkgs := strings.Split(string(data), "\n")
	for i, line := range pkgs {
		infos := strings.Split(line, "=")
		if len(infos) != 2 {
			// Last item might be empty line.
			if i == len(pkgs)-1 {
				continue
			}
			if LibraryMode {
				return fmt.Errorf("Fail to parse package name: %v", line)
			}
			log.Error("", "Fail to parse package name: "+line)
			log.Fatal("", "Invalid package information")
		}
		PackageNameList[strings.TrimSpace(infos[0])] = strings.TrimSpace(infos[1])
	}
	return nil
}

func GetPkgFullPath(short string) string {
	name, ok := PackageNameList[short]
	if !ok {
		log.Error("", "Invalid package name")
		log.Error("", "It's not a invalid import path and no match in the package name list:")
		log.Fatal("", "\t"+short)
	}
	return name
}

var (
	HttpProxy string
)

func LoadConfig() error {
	var err error
	if !com.IsExist(ConfigFile) {
		os.MkdirAll(path.Dir(ConfigFile), os.ModePerm)
		if _, err = os.Create(ConfigFile); err != nil {
			if LibraryMode {
				return fmt.Errorf("Fail to create gopm config file: %v", err)
			}
			log.Error("", "Fail to create gopm config file:")
			log.Fatal("", "\t"+err.Error())
		}
	}
	Cfg, err = goconfig.LoadConfigFile(ConfigFile)
	if err != nil {
		if LibraryMode {
			return fmt.Errorf("Fail to load gopm config file: %v", err)
		}
		log.Error("", "Fail to load gopm config file")
		log.Fatal("", "\t"+err.Error())
	}

	HttpProxy = Cfg.MustValue("settings", "HTTP_PROXY")
	return nil
}

func SetConfigValue(section, key, val string) {
	Cfg.SetValue(section, key, val)
	if err := goconfig.SaveConfigFile(Cfg, ConfigFile); err != nil {
		log.Error("", "Fail to save gopm config file:")
		log.Fatal("", "\t"+err.Error())
	}
}

func DeleteConfigOption(section, key string) {
	Cfg.DeleteKey(section, key)
	if err := goconfig.SaveConfigFile(Cfg, ConfigFile); err != nil {
		log.Error("", "Fail to save gopm config file:")
		log.Fatal("", "\t"+err.Error())
	}
}
