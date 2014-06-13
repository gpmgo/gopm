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

package doc

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/Unknwon/cae/zip"
	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var (
	gopmPattern = regexp.MustCompile(`^gopm\.io/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func getGopmPkg(
	client *http.Client,
	match map[string]string,
	n *Node,
	ctx *cli.Context) ([]string, error) {

	if !n.IsEmptyVal() {
		match["sha"] = n.Value
	} else {
		var rel struct {
			Tag   string `json:"tag"`
			Error string `json:"error"`
		}

		if err := com.HttpGetJSON(client,
			com.Expand("http://gopm.io/api/v1/{owner}/{repo}/releases/latest", match),
			&rel); err != nil {
			log.Warn("GET", "Fail to get revision")
			log.Warn("", "\t"+err.Error())
		} else if len(rel.Tag) == 0 {
			log.Warn("GET", "Fail to get revision")
			log.Warn("", "\t"+rel.Error)
		} else {
			if rel.Tag == n.Revision {
				log.Log("Package hasn't changed: %s", n.ImportPath)
				return nil, nil
			}
			n.Revision = rel.Tag
			match["sha"] = n.Revision
		}
	}

	// Downlaod archive.
	tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+fmt.Sprintf("%d", time.Now().Nanosecond())+".zip")
	if err := com.HttpGetToFile(client,
		com.Expand("http://gopm.io/{owner}/{repo}.zip?r={sha}", match),
		nil, tmpPath); err != nil {
		return nil, fmt.Errorf("fail to download archive: %v", n.ImportPath, err)
	}
	defer os.Remove(tmpPath)

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	// To prevent same output folder name, need to extract to temp path then move.
	if err := zip.ExtractTo(tmpPath, n.InstallPath); err != nil {
		return nil, fmt.Errorf("fail to extract archive: %v", err)
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false)
}
