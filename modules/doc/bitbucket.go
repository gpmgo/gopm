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
	bitbucketPattern = regexp.MustCompile(`^bitbucket\.org/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
	bitbucketEtagRe  = regexp.MustCompile(`^(hg|git)-`)
)

func getBitbucketDoc(
	client *http.Client,
	match map[string]string,
	n *Node,
	ctx *cli.Context) ([]string, error) {

	// Check version control.
	if m := bitbucketEtagRe.FindStringSubmatch(n.Value); m != nil {
		match["vcs"] = m[1]
	} else {
		var repo struct {
			Scm string
		}
		if err := com.HttpGetJSON(client, com.Expand("https://api.bitbucket.org/1.0/repositories/{owner}/{repo}", match), &repo); err != nil {
			return nil, err
		}
		match["vcs"] = repo.Scm
	}

	tags := make(map[string]string)
	for _, nodeType := range []string{"branches", "tags"} {
		var nodes map[string]struct {
			Node string
		}
		if err := com.HttpGetJSON(client, com.Expand("https://api.bitbucket.org/1.0/repositories/{owner}/{repo}/{0}", match, nodeType), &nodes); err != nil {
			log.Warn("GET", "Fail to fetch revision page")
			log.Fatal("", "\t"+err.Error())
		}
		for t, n := range nodes {
			tags[t] = n.Node
		}
	}

	// Get revision.
	var err error
	match["tag"], match["commit"], err = bestTag(tags, defaultTags[match["vcs"]])
	if err != nil {
		return nil, err
	}

	// Check downlaod type.
	switch n.Type {
	case BRANCH:
		if !n.IsEmptyVal() {
			match["commit"] = n.Value
			break
		}

		if match["commit"] == n.Revision {
			log.Log("GET Package hasn't changed: %s", n.ImportPath)
			return nil, nil
		}
	case TAG, COMMIT:
		match["tag"] = n.Value
	default:
		return nil, fmt.Errorf("invalid node type: %s", n.Type)
	}
	n.Revision = match["commit"]

	// We use .zip here.
	// zip : https://bitbucket.org/{owner}/{repo}/get/{commit}.zip
	// tarball : https://bitbucket.org/{owner}/{repo}/get/{commit}.tar.gz

	// Downlaod archive.
	tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+fmt.Sprintf("%s", time.Nanosecond)+".zip")
	if err := com.HttpGetToFile(client,
		com.Expand("https://bitbucket.org/{owner}/{repo}/get/{commit}.zip", match),
		nil, tmpPath); err != nil {
		return nil, fmt.Errorf("fail to download archive(%s): %v", n.ImportPath, err)
	}
	defer os.Remove(tmpPath)

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	shaName := com.Expand("{owner}-{repo}-{commit}", match)

	if err := zip.ExtractTo(tmpPath, path.Dir(n.InstallPath)); err != nil {
		return nil, fmt.Errorf("fail to extract archive(%s): %v", n.ImportPath, err)
	} else if err = os.Rename(path.Join(path.Dir(n.InstallPath), shaName),
		n.InstallPath); err != nil {
		return nil, fmt.Errorf("fail to rename directory(%s): %v", n.ImportPath, err)
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false), nil
}
