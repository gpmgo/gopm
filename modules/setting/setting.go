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

	VERSION = 2014052701
)

var (
	Debug bool

	HomeDir         string
	WorkDir         string // The path of gopm was executed.
	InstallRepoPath string // The gopm local repository.
	InstallGopath   string
	GopmLocalRepo   string

	LocalNodesFile string
	LocalNodes     *goconfig.ConfigFile

	PkgNamesFile    string
	PackageNameList = make(map[string]string)

	ConfigFile string
	Cfg        *goconfig.ConfigFile

	IsWindowsXP bool

	RootPathPairs = map[string]int{
		"github.com":      3,
		"code.google.com": 3,
		"bitbucket.org":   3,
		"git.oschina.net": 3,
		"gitcafe.com":     3,
		"code.csdn.net":   3,
		"launchpad.net":   2,
		"labix.org":       3,
	}
	CommonRes = []string{"views", "templates", "static", "public", "conf"}
)

func LoadLocalNodes() {
	if !com.IsFile(LocalNodesFile) {
		os.MkdirAll(path.Dir(LocalNodesFile), os.ModePerm)
		os.Create(LocalNodesFile)
	}

	var err error
	LocalNodes, err = goconfig.LoadConfigFile(LocalNodesFile)
	if err != nil {
		log.Error("", "Fail to load localnodes.list")
		log.Fatal("", "\t"+err.Error())
	}
}

func SaveLocalNodes() {
	if err := goconfig.SaveConfigFile(LocalNodes, LocalNodesFile); err != nil {
		log.Error("", "Fail to save localnodes.list:")
		log.Error("", "\t"+err.Error())
	}
}

func LoadPkgNameList() {
	if !com.IsFile(PkgNamesFile) {
		return
	}

	data, err := ioutil.ReadFile(PkgNamesFile)
	if err != nil {
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
			log.Error("", "Fail to parse package name: "+line)
			log.Fatal("", "Invalid package information")
		}
		PackageNameList[strings.TrimSpace(infos[0])] = strings.TrimSpace(infos[1])
	}
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

func LoadConfig() {
	var err error
	if !com.IsExist(ConfigFile) {
		os.MkdirAll(path.Dir(ConfigFile), os.ModePerm)
		if _, err = os.Create(ConfigFile); err != nil {
			log.Error("", "Fail to create gopm config file")
			log.Fatal("", "\t"+err.Error())
		}
	}
	Cfg, err = goconfig.LoadConfigFile(ConfigFile)
	if err != nil {
		log.Error("", "Fail to load gopm config file")
		log.Fatal("", "\t"+err.Error())
	}

	HttpProxy = Cfg.MustValue("settings", "HTTP_PROXY")
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
