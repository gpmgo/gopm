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
	googleRepoRe     = regexp.MustCompile(`id="checkoutcmd">(hg|git|svn)`)
	googleRevisionRe = regexp.MustCompile(`<h2>(?:[^ ]+ - )?Revision *([^:]+):`)
	googleFileRe     = regexp.MustCompile(`<li><a href="([^"/]+)"`)
	googleDirRe      = regexp.MustCompile(`<li><a href="([^".]+)"`)
	googlePattern    = regexp.MustCompile(`^code\.google\.com/p/(?P<repo>[a-z0-9\-]+)(:?\.(?P<subrepo>[a-z0-9\-]+))?(?P<dir>/[a-z0-9A-Z_.\-/]+)?$`)
)

func setupGoogleMatch(match map[string]string) {
	if s := match["subrepo"]; s != "" {
		match["dot"] = "."
		match["query"] = "?repo=" + s
	} else {
		match["dot"] = ""
		match["query"] = ""
	}
}

func getGoogleVCS(client *http.Client, match map[string]string) error {
	// Scrape the HTML project page to find the VCS.
	p, err := com.HttpGetBytes(client, com.Expand("http://code.google.com/p/{repo}/source/checkout", match), nil)
	if err != nil {
		return fmt.Errorf("fail to fetch page: %v", err)
	}
	m := googleRepoRe.FindSubmatch(p)
	if m == nil {
		return com.NotFoundError{"Could not VCS on Google Code project page."}
	}
	match["vcs"] = string(m[1])
	return nil
}

type rawFile struct {
	name   string
	rawURL string
	data   []byte
}

func (rf *rawFile) Name() string {
	return rf.name
}

func (rf *rawFile) RawUrl() string {
	return rf.rawURL
}

func (rf *rawFile) Data() []byte {
	return rf.data
}

func (rf *rawFile) SetData(p []byte) {
	rf.data = p
}

func downloadFiles(
	client *http.Client,
	match map[string]string,
	rootPath, installPath, commit string,
	dirs []string) error {

	suf := "?r=" + commit
	if len(commit) == 0 {
		suf = ""
	}

	for _, d := range dirs {
		p, err := com.HttpGetBytes(client, rootPath+d+suf, nil)
		if err != nil {
			return err
		}

		// Create destination directory.
		os.MkdirAll(path.Join(installPath, d), os.ModePerm)

		// Get source files in current path.
		files := make([]com.RawFile, 0, 5)
		for _, m := range googleFileRe.FindAllSubmatch(p, -1) {
			fname := strings.Split(string(m[1]), "?")[0]
			files = append(files, &rawFile{
				name:   fname,
				rawURL: rootPath + d + fname + suf,
			})
		}

		// Fetch files from VCS.
		if err = com.FetchFiles(client, files, nil); err != nil {
			return err
		}

		// Save files.
		for _, f := range files {
			absPath := path.Join(installPath, d)

			// Create diretory before create file.
			os.MkdirAll(path.Dir(absPath), os.ModePerm)

			// Write data to file
			fw, err := os.Create(path.Join(absPath, f.Name()))
			if err != nil {
				return err
			}

			_, err = fw.Write(f.Data())
			fw.Close()
			if err != nil {
				return err
			}
		}
		files = nil

		subdirs := make([]string, 0, 3)
		// Get subdirectories.
		for _, m := range googleDirRe.FindAllSubmatch(p, -1) {
			dirName := strings.Split(string(m[1]), "?")[0]
			if strings.HasSuffix(dirName, "/") {
				subdirs = append(subdirs, d+dirName)
			}
		}

		if err = downloadFiles(client, match,
			rootPath, installPath, commit, subdirs); err != nil {
			return err
		}
	}
	return nil
}

func getGoogleDoc(
	client *http.Client,
	match map[string]string,
	n *Node,
	ctx *cli.Context) ([]string, error) {

	setupGoogleMatch(match)
	// Check version control.
	if err := getGoogleVCS(client, match); err != nil {
		return nil, fmt.Errorf("fail to get package(%s) VCS: %v", n.ImportPath, err)
	}

	switch n.Type {
	case BRANCH:
		if !n.IsEmptyVal() {
			match["tag"] = n.Value
			break
		}

		match["tag"] = defaultTags[match["vcs"]]
	case TAG, COMMIT:
		match["tag"] = n.Value
	default:
		return nil, fmt.Errorf("invalid node type: %s", n.Type)
	}

	// Get revision.
	p, err := com.HttpGetBytes(client,
		com.Expand("http://{subrepo}{dot}{repo}.googlecode.com/{vcs}{dir}/?r={tag}", match), nil)
	if err != nil {
		log.Warn("GET", "Fail to fetch revision page")
		log.Fatal("", "\t"+err.Error())
	}

	if m := googleRevisionRe.FindSubmatch(p); m == nil {
		log.Warn("GET", "Fail to get revision")
		log.Fatal("", "\t"+err.Error())
	} else {
		etag := string(m[1])
		if n.Type == BRANCH && etag == n.Revision {
			log.Log("GET Package hasn't changed: %s", n.ImportPath)
			return nil, nil
		}
		n.Revision = etag
		match["etag"] = n.Revision
	}

	// Remove old files.
	os.RemoveAll(n.InstallPath)
	os.MkdirAll(path.Dir(n.InstallPath), os.ModePerm)

	if match["vcs"] == "svn" {
		log.Warn("SVN detected, may take very long time to finish.")

		rootPath := com.Expand("http://{subrepo}{dot}{repo}.googlecode.com/{vcs}", match)
		d, f := path.Split(rootPath)

		if err := downloadFiles(client, match, d, n.InstallPath, match["tag"],
			[]string{f + "/"}); err != nil {
			return nil, fmt.Errorf("fail to downlaod(%s): %v", n.ImportPath, err)
		}
	} else {
		// Downlaod archive.
		tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
			n.RootPath+"-"+fmt.Sprintf("%s", time.Nanosecond)+".zip")
		if err := com.HttpGetToFile(client,
			com.Expand("http://{subrepo}{dot}{repo}.googlecode.com/archive/{tag}.zip", match),
			nil, tmpPath); err != nil {
			return nil, fmt.Errorf("fail to download archive(%s): %v", n.ImportPath, err)
		}
		defer os.Remove(tmpPath)

		shaName := com.Expand("{subrepo}{dot}{repo}-{etag}", match)

		if err := zip.ExtractTo(tmpPath, path.Dir(n.InstallPath)); err != nil {
			return nil, fmt.Errorf("fail to extract archive(%s): %v", n.ImportPath, err)
		} else if err = os.Rename(path.Join(path.Dir(n.InstallPath), shaName),
			n.InstallPath); err != nil {
			return nil, fmt.Errorf("fail to rename directory(%s): %v", n.ImportPath, err)
		}
	}

	// Check if need to check imports.
	if !n.IsGetDeps {
		return nil, nil
	}
	return GetImports(n.ImportPath, n.RootPath, n.InstallPath, false), nil
}
