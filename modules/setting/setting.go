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

package setting

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/goconfig"
)

type Error struct {
	HasError bool
	Fatal    error
	Errors   []error
}

type Option struct {
}

const (
	VERSION     = 201602010
	VENDOR      = ".vendor"
	GOPMFILE    = ".gopmfile"
	PKGNAMELIST = "pkgname.list"
	VERINFO     = "data/VERSION.json"
)

const (
	URL_API_DOWNLOAD = "/api/v1/download"
	URL_API_REVISION = "/api/v1/revision"
)

var (
	// Global variables.
	HomeDir          string
	WorkDir          string // The path of gopm was executed.
	PkgNameListFile  string
	LocalNodesFile   string
	DefaultGopmfile  string
	DefaultVendor    string
	DefaultVendorSrc string
	InstallRepoPath  string // The gopm local repository.
	InstallGopath    string
	HttpProxy        string
	RegistryURL      string = "https://gopm.io"

	// System settings.
	IsWindows        bool
	IsWindowsXP      bool
	HasGOPATHSetting bool

	// Global settings.
	Debug        bool
	LibraryMode  bool
	RuntimeError = new(Error)

	// Configuration settings.
	ConfigFile      string
	Cfg             *goconfig.ConfigFile
	PackageNameList = make(map[string]string)
	LocalNodes      *goconfig.ConfigFile

	// TODO: configurable.
	RootPathPairs = map[string]int{
		"github.com":      3,
		"bitbucket.org":   3,
		"git.oschina.net": 3,
		"launchpad.net":   2,
		"golang.org":      3,
	}
	CommonRes = []string{"views", "templates", "static", "public", "conf"}
)

// LoadGopmfile loads and returns given gopmfile.
func LoadGopmfile(fileName string) (*goconfig.ConfigFile, error) {
	if !base.IsFile(fileName) {
		return goconfig.LoadFromData([]byte(""))
	}

	gf, err := goconfig.LoadConfigFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Fail to load gopmfile: %v", err)
	}
	return gf, nil
}

// SaveGopmfile saves gopmfile to given path.
func SaveGopmfile(gf *goconfig.ConfigFile, fileName string) error {
	if err := goconfig.SaveConfigFile(gf, fileName); err != nil {
		return fmt.Errorf("Fail to save gopmfile: %v", err)
	}
	return nil
}

// LoadConfig loads gopm global configuration.
func LoadConfig() (err error) {
	if !base.IsExist(ConfigFile) {
		os.MkdirAll(path.Dir(ConfigFile), os.ModePerm)
		if _, err = os.Create(ConfigFile); err != nil {
			return fmt.Errorf("fail to create config file: %v", err)
		}
	}

	Cfg, err = goconfig.LoadConfigFile(ConfigFile)
	if err != nil {
		return fmt.Errorf("fail to load config file: %v", err)
	}

	HttpProxy = Cfg.MustValue("settings", "HTTP_PROXY")
	return nil
}

// SetConfigValue sets and saves gopm configuration.
func SetConfigValue(section, key, val string) error {
	Cfg.SetValue(section, key, val)
	if err := goconfig.SaveConfigFile(Cfg, ConfigFile); err != nil {
		return fmt.Errorf("fail to set config value(%s:%s=%s): %v", section, key, val, err)
	}
	return nil
}

// DeleteConfigOption deletes and saves gopm configuration.
func DeleteConfigOption(section, key string) error {
	Cfg.DeleteKey(section, key)
	if err := goconfig.SaveConfigFile(Cfg, ConfigFile); err != nil {
		return fmt.Errorf("fail to delete config key(%s:%s): %v", section, key, err)
	}
	return nil
}

// LoadPkgNameList loads package name pairs.
func LoadPkgNameList() error {
	if !base.IsFile(PkgNameListFile) {
		return nil
	}

	data, err := ioutil.ReadFile(PkgNameListFile)
	if err != nil {
		return fmt.Errorf("fail to load package name list: %v", err)
	}

	pkgs := strings.Split(string(data), "\n")
	for i, line := range pkgs {
		infos := strings.Split(line, "=")
		if len(infos) != 2 {
			// Last item might be empty line.
			if i == len(pkgs)-1 {
				continue
			}
			return fmt.Errorf("fail to parse package name: %v", line)
		}
		PackageNameList[strings.TrimSpace(infos[0])] = strings.TrimSpace(infos[1])
	}
	return nil
}

// GetPkgFullPath attmpts to get full path by given package short name.
func GetPkgFullPath(short string) (string, error) {
	name, ok := PackageNameList[short]
	if !ok {
		return "", fmt.Errorf("no match package import path with given short name: %s", short)
	}
	return name, nil
}

func LoadLocalNodes() (err error) {
	if !base.IsFile(LocalNodesFile) {
		os.MkdirAll(path.Dir(LocalNodesFile), os.ModePerm)
		os.Create(LocalNodesFile)
	}

	LocalNodes, err = goconfig.LoadConfigFile(LocalNodesFile)
	if err != nil {
		return fmt.Errorf("fail to load localnodes.list: %v", err)
	}
	return nil
}

func SaveLocalNodes() error {
	if err := goconfig.SaveConfigFile(LocalNodes, LocalNodesFile); err != nil {
		return fmt.Errorf("fail to save localnodes.list: %v", err)
	}
	return nil
}
