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

var (
	oscTagRe      = regexp.MustCompile(`/repository/archive\?ref=(.*)">`)
	oscRevisionRe = regexp.MustCompile(`<span class='sha'>[a-z0-9A-Z]*`)
	oscPattern    = regexp.MustCompile(`^git\.oschina\.net/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func getOscDoc(
	client *http.Client,
	match map[string]string,
	n *Node,
	ctx *cli.Context) ([]string, error) {

	// Check downlaod type.
	switch n.Type {
	case BRANCH:
		if !n.IsEmptyVal() {
			match["sha"] = n.Value
			break
		}

		match["sha"] = MASTER

		// Get revision.
		p, err := com.HttpGetBytes(client,
			com.Expand("http://git.oschina.net/{owner}/{repo}/tree/{sha}", match), nil)
		if err != nil {
			log.Warn("GET", "Fail to fetch revision page")
			log.Fatal("", "\t"+err.Error())
		}

		if m := oscRevisionRe.FindSubmatch(p); m == nil {
			log.Warn("GET", "Fail to get revision")
			log.Fatal("", "\t"+err.Error())
		} else {
			etag := strings.TrimPrefix(string(m[0]), `<span class='sha'>`)
			if etag == n.Revision {
				log.Log("GET Package hasn't changed: %s", n.ImportPath)
				return nil, nil
			}
			n.Revision = etag
		}
	case TAG, COMMIT:
		match["sha"] = n.Value
	default:
		return nil, fmt.Errorf("invalid node type: %s", n.Type)
	}

	// zip: http://{projectRoot}/repository/archive?ref={sha}

	// Downlaod archive.
	tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+fmt.Sprintf("%d", time.Now().Nanosecond())+".zip")
	if err := com.HttpGetToFile(client,
		com.Expand("http://git.oschina.net/{owner}/{repo}/repository/archive?ref={sha}", match),
		nil, tmpPath); err != nil {
		return nil, fmt.Errorf("fail to download archive: %v", n.ImportPath, err)
	}
	defer os.Remove(tmpPath)

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	tmpExtPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+fmt.Sprintf("%s", time.Now().Nanosecond()))
	// To prevent same output folder name, need to extract to temp path then move.
	if err := zip.ExtractTo(tmpPath, tmpExtPath); err != nil {
		return nil, fmt.Errorf("fail to extract archive: %v", n.ImportPath, err)
	} else if err = os.Rename(path.Join(tmpExtPath, com.Expand("{repo}", match)),
		n.InstallPath); err != nil {
		return nil, fmt.Errorf("fail to rename directory: %v", n.ImportPath, err)
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false), nil
}
