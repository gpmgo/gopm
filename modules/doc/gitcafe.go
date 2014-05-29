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
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	// "time"

	// "github.com/Unknwon/cae/tz"
	"github.com/Unknwon/com"
	"github.com/codegangsta/cli"

	"github.com/gpmgo/gopm/modules/log"
	// "github.com/gpmgo/gopm/modules/setting"
)

var (
	gitcafeRevisionRe = regexp.MustCompile(`<i class="icon-push"></i>[a-z0-9A-Z]*`)
	gitcafePattern    = regexp.MustCompile(`^gitcafe\.com/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func getGitcafeDoc(
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
			com.Expand("http://gitcafe.com/{owner}/{repo}/tree/{sha}", match), nil)
		if err != nil {
			log.Warn("GET", "Fail to fetch revision page")
			log.Fatal("", "\t"+err.Error())
		}

		if m := gitcafeRevisionRe.FindSubmatch(p); m == nil {
			log.Warn("GET", "Fail to get revision")
			log.Fatal("", "\t"+err.Error())
		} else {
			etag := strings.TrimPrefix(string(m[0]), `<i class="icon-push"></i>`)
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

	// tar.gz: http://{projectRoot}/tarball/{sha}

	// Downlaod archive.
	p, err := com.HttpGetBytes(client, com.Expand("http://gitcafe.com/{owner}/{repo}/tarball/{sha}", match), nil)
	if err != nil {
		return nil, err
	}

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	tr := tar.NewReader(bytes.NewReader(p))

	var rootPath string
	// Get source file data.
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		fname := h.Name
		if fname == "pax_global_header" {
			continue
		}

		if len(rootPath) == 0 {
			rootPath = fname[:strings.Index(fname, match["repo"])+len(match["repo"])]
		}
		absPath := strings.Replace(fname, rootPath, n.InstallPath, 1)

		switch {
		case h.FileInfo().IsDir():
			// Create diretory before create file.
			os.MkdirAll(absPath+"/", os.ModePerm)
		case !strings.HasPrefix(fname, "."):
			// Get data from archive.
			fbytes := make([]byte, h.Size)
			if _, err := io.ReadFull(tr, fbytes); err != nil {
				return nil, err
			}

			if err = com.WriteFile(absPath, fbytes); err != nil {
				return nil, err
			}
		}
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false), nil
}
