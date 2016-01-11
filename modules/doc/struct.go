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

package doc

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/cae/zip"
	"github.com/gpmgo/gopm/modules/cli"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

// service represents a source code control service.
type service struct {
	pattern *regexp.Regexp
	prefix  string
	get     func(*http.Client, map[string]string, *Node, *cli.Context) ([]string, error)
}

// services is the list of source code control services handled by gopm.
var services = []*service{
// {githubPattern, "github.com/", getGithubPkg},
// {googlePattern, "code.google.com/", getGooglePkg},
// {bitbucketPattern, "bitbucket.org/", getBitbucketPkg},
// {oscPattern, "git.oschina.net/", getOscPkg},
// {gitcafePattern, "gitcafe.com/", getGitcafePkg},
// {launchpadPattern, "launchpad.net/", getLaunchpadPkg},
}

type RevisionType string

const (
	BRANCH RevisionType = "branch"
	COMMIT RevisionType = "commit"
	TAG    RevisionType = "tag"
	LOCAL  RevisionType = "local"
)

// Common default branch names.
const (
	TRUNK   = "trunk"
	MASTER  = "master"
	DEFAULT = "default"
)

// A Pkg represents a remote Go package.
type Pkg struct {
	ImportPath string // Package full import path.
	RootPath   string // Package root path on VCS.
	Type       RevisionType
	Value      string
}

func NewPkg(importPath string, tp RevisionType, val string) *Pkg {
	return &Pkg{importPath, GetRootPath(importPath), tp, val}
}

func NewDefaultPkg(importPath string) *Pkg {
	return NewPkg(importPath, BRANCH, "")
}

// If the package is fixed and no need to updated.
// For commit, tag and local, it's fixed.
func (pkg *Pkg) IsFixed() bool {
	if pkg.Type == BRANCH || len(pkg.Value) == 0 {
		return false
	}
	return true
}

func (pkg *Pkg) IsEmptyVal() bool {
	return len(pkg.Value) == 0
}

func (pkg *Pkg) ValSuffix() string {
	if len(pkg.Value) > 0 {
		return "." + pkg.Value
	}
	return ""
}

func (pkg *Pkg) VerSuffix() string {
	if len(pkg.Value) > 0 {
		return " @ " + string(pkg.Type) + ":" + pkg.Value
	}
	return ""
}

// A Node represents a node object to be fetched from remote.
type Node struct {
	Pkg
	DownloadURL   string // Actual download URL can be different from import path.
	InstallPath   string // Local install path.
	InstallGopath string
	Synopsis      string
	IsGetDeps     bool // False for downloading package itself only.
	IsGetDepsOnly bool // True for skiping download package itself.
	Revision      string
}

// NewNode initializes and returns a new Node representation.
func NewNode(
	importPath string,
	tp RevisionType, val string,
	isGetDeps bool) *Node {

	n := &Node{
		Pkg: Pkg{
			ImportPath: importPath,
			RootPath:   GetRootPath(importPath),
			Type:       tp,
			Value:      val,
		},
		DownloadURL: importPath,
		IsGetDeps:   isGetDeps,
	}
	n.InstallPath = path.Join(setting.InstallRepoPath, n.RootPath) + n.ValSuffix()
	n.InstallGopath = path.Join(setting.InstallGopath, n.RootPath)
	return n
}

// IsExist returns true if package exists in local repository.
func (n *Node) IsExist() bool {
	return base.IsExist(n.InstallPath)
}

// IsExistGopath returns true if package exists in GOPATH.
func (n *Node) IsExistGopath() bool {
	return base.IsExist(n.InstallGopath)
}

func (n *Node) ValString() string {
	if len(n.Value) == 0 {
		return "<UTD>"
	}
	return n.Value
}

func (n *Node) VerString() string {
	return fmt.Sprintf("%s@%s:%s", n.ImportPath, n.Type, n.ValString())
}

func (n *Node) HasVcs() bool {
	return len(GetVcsName(n.InstallGopath)) > 0
}

func (n *Node) CopyToGopath() error {
	if n.HasVcs() {
		log.Warn("Package in GOPATH has version control: %s", n.RootPath)
		return nil
	}

	os.RemoveAll(n.InstallGopath)
	if err := base.CopyDir(n.InstallPath, n.InstallGopath); err != nil {
		if setting.LibraryMode {
			return fmt.Errorf("Fail to copy to GOPATH: %v", err)
		}
		log.Error("", "Fail to copy to GOPATH:")
		log.Fatal("", "\t"+err.Error())
	}
	log.Info("Package copied to GOPATH: %s", n.RootPath)
	return nil
}

// If vcs has been detected, use corresponding command to update package.
func (n *Node) UpdateByVcs(vcs string) error {
	switch vcs {
	case "git":
		branch, stderr, err := base.ExecCmdDir(n.InstallGopath,
			"git", "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			log.Error("", "Error occurs when 'git rev-parse --abbrev-ref HEAD'")
			log.Error("", "\t"+stderr)
			return errors.New(stderr)
		}
		branch = strings.TrimSpace(branch)

		_, stderr, err = base.ExecCmdDir(n.InstallGopath,
			"git", "pull", "origin", branch)
		if err != nil {
			log.Error("", "Error occurs when 'git pull origin "+branch+"'")
			log.Error("", "\t"+stderr)
			return errors.New(stderr)
		}
	case "hg":
		_, stderr, err := base.ExecCmdDir(n.InstallGopath,
			"hg", "pull")
		if err != nil {
			log.Error("", "Error occurs when 'hg pull'")
			log.Error("", "\t"+stderr)
			return errors.New(stderr)
		}

		_, stderr, err = base.ExecCmdDir(n.InstallGopath,
			"hg", "up")
		if err != nil {
			log.Error("", "Error occurs when 'hg up'")
			log.Error("", "\t"+stderr)
			return errors.New(stderr)
		}
	case "svn":
		_, stderr, err := base.ExecCmdDir(n.InstallGopath,
			"svn", "update")
		if err != nil {
			log.Error("", "Error occurs when 'svn update'")
			log.Error("", "\t"+stderr)
			return errors.New(stderr)
		}
	}
	return nil
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if strings.EqualFold(a.Name.Local, name) {
			return a.Value
		}
	}
	return ""
}

func parseMeta(scheme, importPath string, r io.Reader) (map[string]string, error) {
	var match map[string]string

	d := xml.NewDecoder(r)
	d.Strict = false
metaScan:
	for {
		t, tokenErr := d.Token()
		if tokenErr != nil {
			break metaScan
		}
		switch t := t.(type) {
		case xml.EndElement:
			if strings.EqualFold(t.Name.Local, "head") {
				break metaScan
			}
		case xml.StartElement:
			if strings.EqualFold(t.Name.Local, "body") {
				break metaScan
			}
			if !strings.EqualFold(t.Name.Local, "meta") ||
				attrValue(t.Attr, "name") != "go-import" {
				continue metaScan
			}
			f := strings.Fields(attrValue(t.Attr, "content"))
			if len(f) != 3 ||
				!strings.HasPrefix(importPath, f[0]) ||
				!(len(importPath) == len(f[0]) || importPath[len(f[0])] == '/') {
				continue metaScan
			}
			if match != nil {
				return nil, fmt.Errorf("more than one <meta> found at %s://%s", scheme, importPath)
			}

			projectRoot, vcs, repo := f[0], f[1], f[2]

			repo = strings.TrimSuffix(repo, "."+vcs)
			i := strings.Index(repo, "://")
			if i < 0 {
				return nil, fmt.Errorf("bad repo URL in <meta>")
			}
			proto := repo[:i]
			repo = repo[i+len("://"):]

			match = map[string]string{
				// Used in getVCSPkg, same as vcsPattern matches.
				"importPath": importPath,
				"repo":       repo,
				"vcs":        vcs,
				"dir":        importPath[len(projectRoot):],

				// Used in getVCSPkg
				"scheme": proto,

				// Used in getDynamic.
				"projectRoot": projectRoot,
				"projectName": path.Base(projectRoot),
				"projectURL":  scheme + "://" + projectRoot,
			}
		}
	}
	if match == nil {
		return nil, fmt.Errorf("<meta> not found")
	}
	return match, nil
}

func fetchMeta(client *http.Client, importPath string) (map[string]string, error) {
	uri := importPath
	if !strings.Contains(uri, "/") {
		// Add slash for root of domain.
		uri = uri + "/"
	}
	uri = uri + "?go-get=1"

	scheme := "https"
	resp, err := client.Get(scheme + "://" + uri)
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			resp.Body.Close()
		}
		scheme = "http"
		resp, err = client.Get(scheme + "://" + uri)
		if err != nil {
			return nil, fmt.Errorf("fail to make request(%s): %v", strings.SplitN(importPath, "/", 2)[0], err)
		}
	}
	defer resp.Body.Close()
	return parseMeta(scheme, importPath, resp.Body)
}

func (n *Node) getDynamic(client *http.Client, ctx *cli.Context) ([]string, error) {
	match, err := fetchMeta(client, n.ImportPath)
	if err != nil {
		return nil, err
	}

	if match["projectRoot"] != n.ImportPath {
		rootMatch, err := fetchMeta(client, match["projectRoot"])
		if err != nil {
			return nil, err
		}
		if rootMatch["projectRoot"] != match["projectRoot"] {
			return nil, fmt.Errorf("project root mismatch")
		}
	}

	n.DownloadURL = base.Expand("{repo}{dir}", match)
	return n.Download(ctx)
}

// Download downloads remote package without version control.
func (n *Node) Download(ctx *cli.Context) ([]string, error) {
	for _, s := range services {
		if !strings.HasPrefix(n.DownloadURL, s.prefix) {
			continue
		}

		m := s.pattern.FindStringSubmatch(n.DownloadURL)
		if m == nil {
			if s.prefix != "" {
				return nil, errors.New("Cannot match package service prefix by given path")
			}
			continue
		}

		match := map[string]string{"downloadURL": n.DownloadURL}
		for i, n := range s.pattern.SubexpNames() {
			if n != "" {
				match[n] = m[i]
			}
		}
		return s.get(HttpClient, match, n, ctx)

	}

	if n.ImportPath != n.DownloadURL {
		return nil, errors.New("Didn't find any match service")
	}

	log.Info("Cannot match any service, getting dynamic...")
	return n.getDynamic(HttpClient, ctx)
}

type ApiError struct {
	Error string `json:"error"`
}

type ApiResponse struct {
	Sha string `json:"sha"`
}

func init() {
	zip.Verbose = false
}

// DownloadGopm downloads remote package from gopm registry.
func (n *Node) DownloadGopm(ctx *cli.Context) error {
	// Fetch latest version, check if package has been changed.
	if n.Type == BRANCH && n.IsEmptyVal() {
		resp, err := http.Get(fmt.Sprintf("%s%s?pkgname=%s",
			setting.RegistryURL, setting.URL_API_REVISION, n.RootPath))
		if err != nil {
			return fmt.Errorf("fail to make request: %v", err)
		}
		if resp.StatusCode != 200 {
			var apiErr ApiError
			if err = json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
				return fmt.Errorf("fail to decode response JSON: %v", err)
			}
			return errors.New(apiErr.Error)
		}
		var apiResp ApiResponse
		if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return fmt.Errorf("fail to decode response JSON: %v", err)
		}
		if n.Revision == apiResp.Sha {
			log.Info("Package(%s) hasn't been changed", n.RootPath)
			return nil
		}
		n.Revision = apiResp.Sha
	}

	resp, err := http.Get(fmt.Sprintf("%s%s?pkgname=%s&revision=%s",
		setting.RegistryURL, setting.URL_API_DOWNLOAD, n.RootPath, n.Value))
	if err != nil {
		return fmt.Errorf("fail to make request: %v", err)
	}
	if resp.StatusCode != 200 {
		var apiErr ApiError
		if err = json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("fail to decode response JSON: %v", err)
		}
		return errors.New(apiErr.Error)
	}

	tmpPath := path.Join(setting.HomeDir, ".gopm/temp/archive",
		n.RootPath+"-"+base.ToStr(time.Now().Nanosecond())+".zip")
	defer os.Remove(tmpPath)
	if setting.Debug {
		log.Debug("Temp archive path: %s", tmpPath)
	}

	os.MkdirAll(path.Dir(tmpPath), os.ModePerm)
	fw, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, resp.Body); err != nil {
		return fmt.Errorf("fail to save archive: %v", err)
	}
	fw.Close()

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
		return fmt.Errorf("fail to extract archive: %v", err)
	} else if err = os.Rename(path.Join(path.Dir(n.InstallPath), rootDir),
		n.InstallPath); err != nil {
		return fmt.Errorf("fail to rename directory: %v", err)
	}
	return nil
}
