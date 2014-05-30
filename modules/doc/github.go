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
	"strings"
	"time"

	"github.com/Unknwon/cae/zip"
	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

func init() {
	zip.Verbose = false
}

var (
	githubPattern = regexp.MustCompile(`^github\.com/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func GetGithubCredentials() string {
	return "client_id=" + setting.Cfg.MustValue("github", "CLIENT_ID") +
		"&client_secret=" + setting.Cfg.MustValue("github", "CLIENT_SECRET")
}

func getGithubDoc(
	client *http.Client,
	match map[string]string,
	n *Node,
	ctx *cli.Context) ([]string, error) {

	match["cred"] = GetGithubCredentials()

	// Check downlaod type.
	switch n.Type {
	case BRANCH:
		if !n.IsEmptyVal() {
			match["sha"] = n.Value
			break
		}

		match["sha"] = MASTER

		// Only get and check revision with the latest version.
		var refs []*struct {
			Ref    string
			Object struct {
				Sha string
			}
		}

		if err := com.HttpGetJSON(client,
			com.Expand("https://api.github.com/repos/{owner}/{repo}/git/refs?{cred}", match),
			&refs); err != nil {
			if strings.Contains(err.Error(), "403") {
				// NOTE: get revision from center repository.
				break
			}
			log.Warn("GET", "Fail to get revision")
			log.Warn("", "\t"+err.Error())
			break
		}

		var etag string
		for _, ref := range refs {
			if strings.HasPrefix(ref.Ref, "refs/heads/master") {
				etag = ref.Object.Sha
				break
			}
		}
		if etag == n.Revision {
			log.Log("GET Package hasn't changed: %s", n.ImportPath)
			return nil, nil
		}
		n.Revision = etag
	case TAG, COMMIT:
		match["sha"] = n.Value
	default:
		return nil, fmt.Errorf("invalid node type: %s", n.Type)
	}

	// We use .zip here.
	// zip: https://github.com/{owner}/{repo}/archive/{sha}.zip
	// tarball: https://github.com/{owner}/{repo}/tarball/{sha}

	// Downlaod archive.
	tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+fmt.Sprintf("%s", time.Now().Nanosecond())+".zip")
	if err := com.HttpGetToFile(client,
		com.Expand("https://github.com/{owner}/{repo}/archive/{sha}.zip", match),
		nil, tmpPath); err != nil {
		return nil, fmt.Errorf("fail to download archive: %v", n.ImportPath, err)
	}
	defer os.Remove(tmpPath)

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	var rootDir string
	var extractFn = func(fullName string, fi os.FileInfo) error {
		if len(rootDir) == 0 {
			rootDir = strings.Split(fullName, "/")[0]
		}
		return nil
	}

	if err := zip.ExtractToFunc(tmpPath, path.Dir(n.InstallPath), extractFn); err != nil {
		return nil, fmt.Errorf("fail to extract archive: %v", err)
	} else if err = os.Rename(path.Join(path.Dir(n.InstallPath), rootDir),
		n.InstallPath); err != nil {
		return nil, fmt.Errorf("fail to rename directory: %v", err)
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false), nil
}
