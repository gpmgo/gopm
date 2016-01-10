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

package base

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// IsFile returns true if given path is a file,
// or returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// IsDir returns true if given path is a directory,
// or returns false when it's a file or does not exist.
func IsDir(dir string) bool {
	f, e := os.Stat(dir)
	if e != nil {
		return false
	}
	return f.IsDir()
}

func statDir(dirPath, recPath string, includeDir, isDirOnly bool) ([]string, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	fis, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	statList := make([]string, 0)
	for _, fi := range fis {
		if strings.Contains(fi.Name(), ".DS_Store") {
			continue
		}

		relPath := path.Join(recPath, fi.Name())
		curPath := path.Join(dirPath, fi.Name())
		if fi.IsDir() {
			if includeDir {
				statList = append(statList, relPath+"/")
			}
			s, err := statDir(curPath, relPath, includeDir, isDirOnly)
			if err != nil {
				return nil, err
			}
			statList = append(statList, s...)
		} else if !isDirOnly {
			statList = append(statList, relPath)
		}
	}
	return statList, nil
}

// StatDir gathers information of given directory by depth-first.
// It returns slice of file list and includes subdirectories if enabled;
// it returns error and nil slice when error occurs in underlying functions,
// or given path is not a directory or does not exist.
//
// Slice does not include given path itself.
// If subdirectories is enabled, they will have suffix '/'.
func StatDir(rootPath string, includeDir ...bool) ([]string, error) {
	if !IsDir(rootPath) {
		return nil, errors.New("not a directory or does not exist: " + rootPath)
	}

	isIncludeDir := false
	if len(includeDir) >= 1 {
		isIncludeDir = includeDir[0]
	}
	return statDir(rootPath, "", isIncludeDir, false)
}

// Copy copies file from source to target path.
func Copy(src, dest string) error {
	// Gather file information to set back later.
	si, err := os.Lstat(src)
	if err != nil {
		return err
	}

	// Handle symbolic link.
	if si.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		// NOTE: os.Chmod and os.Chtimes don't recoganize symbolic link,
		// which will lead "no such file or directory" error.
		return os.Symlink(target, dest)
	}

	sr, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sr.Close()

	dw, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dw.Close()

	if _, err = io.Copy(dw, sr); err != nil {
		return err
	}

	// Set back file information.
	if err = os.Chtimes(dest, si.ModTime(), si.ModTime()); err != nil {
		return err
	}
	return os.Chmod(dest, si.Mode())
}

// CopyDir copy files recursively from source to target directory.
//
// The filter accepts a function that process the path info.
// and should return true for need to filter.
//
// It returns error when error occurs in underlying functions.
func CopyDir(srcPath, destPath string, filters ...func(filePath string) bool) error {
	// Check if target directory exists.
	if IsExist(destPath) {
		return errors.New("file or directory alreay exists: " + destPath)
	}

	err := os.MkdirAll(destPath, os.ModePerm)
	if err != nil {
		return err
	}

	// Gather directory info.
	infos, err := StatDir(srcPath, true)
	if err != nil {
		return err
	}

	var filter func(filePath string) bool
	if len(filters) > 0 {
		filter = filters[0]
	}

	for _, info := range infos {
		if filter != nil && filter(info) {
			continue
		}

		curPath := path.Join(destPath, info)
		if strings.HasSuffix(info, "/") {
			err = os.MkdirAll(curPath, os.ModePerm)
		} else {
			err = Copy(path.Join(srcPath, info), curPath)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ExecCmdDirBytes executes system command in given directory
// and return stdout, stderr in bytes type, along with possible error.
func ExecCmdDirBytes(dir, cmdName string, args ...string) ([]byte, []byte, error) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	cmd := exec.Command(cmdName, args...)
	cmd.Dir = dir
	cmd.Stdout = bufOut
	cmd.Stderr = bufErr

	err := cmd.Run()
	return bufOut.Bytes(), bufErr.Bytes(), err
}

// ExecCmdBytes executes system command
// and return stdout, stderr in bytes type, along with possible error.
func ExecCmdBytes(cmdName string, args ...string) ([]byte, []byte, error) {
	return ExecCmdDirBytes("", cmdName, args...)
}

// ExecCmdDir executes system command in given directory
// and return stdout, stderr in string type, along with possible error.
func ExecCmdDir(dir, cmdName string, args ...string) (string, string, error) {
	bufOut, bufErr, err := ExecCmdDirBytes(dir, cmdName, args...)
	return string(bufOut), string(bufErr), err
}

// ExecCmd executes system command
// and return stdout, stderr in string type, along with possible error.
func ExecCmd(cmdName string, args ...string) (string, string, error) {
	return ExecCmdDir("", cmdName, args...)
}

// Expand replaces {k} in template with match[k] or subs[atoi(k)] if k is not in match.
func Expand(template string, match map[string]string, subs ...string) string {
	var p []byte
	var i int
	for {
		i = strings.Index(template, "{")
		if i < 0 {
			break
		}
		p = append(p, template[:i]...)
		template = template[i+1:]
		i = strings.Index(template, "}")
		if s, ok := match[template[:i]]; ok {
			p = append(p, s...)
		} else {
			j, _ := strconv.Atoi(template[:i])
			if j >= len(subs) {
				p = append(p, []byte("Missing")...)
			} else {
				p = append(p, subs[j]...)
			}
		}
		template = template[i+1:]
	}
	p = append(p, template...)
	return string(p)
}

// GetGOPATHs returns all paths in GOPATH variable.
func GetGOPATHs() []string {
	gopath := os.Getenv("GOPATH")
	var paths []string
	if runtime.GOOS == "windows" {
		gopath = strings.Replace(gopath, "\\", "/", -1)
		paths = strings.Split(gopath, ";")
	} else {
		paths = strings.Split(gopath, ":")
	}
	return paths
}

// HomeDir returns path of '~'(in Linux) on Windows,
// it returns error when the variable does not exist.
func HomeDir() (home string, _ error) {
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	} else {
		home = os.Getenv("HOME")
	}

	if len(home) == 0 {
		return "", errors.New("Cannot specify home directory because it's empty")
	}

	return home, nil
}

// IsSliceContainsStr returns true if the string exists in given slice.
func IsSliceContainsStr(sl []string, str string) bool {
	str = strings.ToLower(str)
	for _, s := range sl {
		if strings.ToLower(s) == str {
			return true
		}
	}
	return false
}

var validTLD = map[string]bool{
	// curl http://data.iana.org/TLD/tlds-alpha-by-domain.txt | sed  -e '/#/ d' -e 's/.*/"&": true,/' | tr [:upper:] [:lower:]
	".ac":                     true,
	".ad":                     true,
	".ae":                     true,
	".aero":                   true,
	".af":                     true,
	".ag":                     true,
	".ai":                     true,
	".al":                     true,
	".am":                     true,
	".an":                     true,
	".ao":                     true,
	".aq":                     true,
	".ar":                     true,
	".arpa":                   true,
	".as":                     true,
	".asia":                   true,
	".at":                     true,
	".au":                     true,
	".aw":                     true,
	".ax":                     true,
	".az":                     true,
	".ba":                     true,
	".bb":                     true,
	".bd":                     true,
	".be":                     true,
	".bf":                     true,
	".bg":                     true,
	".bh":                     true,
	".bi":                     true,
	".biz":                    true,
	".bj":                     true,
	".bm":                     true,
	".bn":                     true,
	".bo":                     true,
	".br":                     true,
	".bs":                     true,
	".bt":                     true,
	".bv":                     true,
	".bw":                     true,
	".by":                     true,
	".bz":                     true,
	".ca":                     true,
	".cat":                    true,
	".cc":                     true,
	".cd":                     true,
	".cf":                     true,
	".cg":                     true,
	".ch":                     true,
	".ci":                     true,
	".ck":                     true,
	".cl":                     true,
	".cm":                     true,
	".cn":                     true,
	".co":                     true,
	".com":                    true,
	".coop":                   true,
	".cr":                     true,
	".cu":                     true,
	".cv":                     true,
	".cw":                     true,
	".cx":                     true,
	".cy":                     true,
	".cz":                     true,
	".de":                     true,
	".dj":                     true,
	".dk":                     true,
	".dm":                     true,
	".do":                     true,
	".dz":                     true,
	".ec":                     true,
	".edu":                    true,
	".ee":                     true,
	".eg":                     true,
	".er":                     true,
	".es":                     true,
	".et":                     true,
	".eu":                     true,
	".fi":                     true,
	".fj":                     true,
	".fk":                     true,
	".fm":                     true,
	".fo":                     true,
	".fr":                     true,
	".ga":                     true,
	".gb":                     true,
	".gd":                     true,
	".ge":                     true,
	".gf":                     true,
	".gg":                     true,
	".gh":                     true,
	".gi":                     true,
	".gl":                     true,
	".gm":                     true,
	".gn":                     true,
	".gov":                    true,
	".gp":                     true,
	".gq":                     true,
	".gr":                     true,
	".gs":                     true,
	".gt":                     true,
	".gu":                     true,
	".gw":                     true,
	".gy":                     true,
	".hk":                     true,
	".hm":                     true,
	".hn":                     true,
	".hr":                     true,
	".ht":                     true,
	".hu":                     true,
	".id":                     true,
	".ie":                     true,
	".il":                     true,
	".im":                     true,
	".in":                     true,
	".info":                   true,
	".int":                    true,
	".io":                     true,
	".iq":                     true,
	".ir":                     true,
	".is":                     true,
	".it":                     true,
	".je":                     true,
	".jm":                     true,
	".jo":                     true,
	".jobs":                   true,
	".jp":                     true,
	".ke":                     true,
	".kg":                     true,
	".kh":                     true,
	".ki":                     true,
	".km":                     true,
	".kn":                     true,
	".kp":                     true,
	".kr":                     true,
	".kw":                     true,
	".ky":                     true,
	".kz":                     true,
	".la":                     true,
	".lb":                     true,
	".lc":                     true,
	".li":                     true,
	".lk":                     true,
	".lr":                     true,
	".ls":                     true,
	".lt":                     true,
	".lu":                     true,
	".lv":                     true,
	".ly":                     true,
	".ma":                     true,
	".mc":                     true,
	".md":                     true,
	".me":                     true,
	".mg":                     true,
	".mh":                     true,
	".mil":                    true,
	".mk":                     true,
	".ml":                     true,
	".mm":                     true,
	".mn":                     true,
	".mo":                     true,
	".mobi":                   true,
	".mp":                     true,
	".mq":                     true,
	".mr":                     true,
	".ms":                     true,
	".mt":                     true,
	".mu":                     true,
	".museum":                 true,
	".mv":                     true,
	".mw":                     true,
	".mx":                     true,
	".my":                     true,
	".mz":                     true,
	".na":                     true,
	".name":                   true,
	".nc":                     true,
	".ne":                     true,
	".net":                    true,
	".nf":                     true,
	".ng":                     true,
	".ni":                     true,
	".nl":                     true,
	".no":                     true,
	".np":                     true,
	".nr":                     true,
	".nu":                     true,
	".nz":                     true,
	".om":                     true,
	".org":                    true,
	".pa":                     true,
	".pe":                     true,
	".pf":                     true,
	".pg":                     true,
	".ph":                     true,
	".pk":                     true,
	".pl":                     true,
	".pm":                     true,
	".pn":                     true,
	".post":                   true,
	".pr":                     true,
	".pro":                    true,
	".ps":                     true,
	".pt":                     true,
	".pw":                     true,
	".py":                     true,
	".qa":                     true,
	".re":                     true,
	".ro":                     true,
	".rs":                     true,
	".ru":                     true,
	".rw":                     true,
	".sa":                     true,
	".sb":                     true,
	".sc":                     true,
	".sd":                     true,
	".se":                     true,
	".sg":                     true,
	".sh":                     true,
	".si":                     true,
	".sj":                     true,
	".sk":                     true,
	".sl":                     true,
	".sm":                     true,
	".sn":                     true,
	".so":                     true,
	".sr":                     true,
	".st":                     true,
	".su":                     true,
	".sv":                     true,
	".sx":                     true,
	".sy":                     true,
	".sz":                     true,
	".tc":                     true,
	".td":                     true,
	".tel":                    true,
	".tf":                     true,
	".tg":                     true,
	".th":                     true,
	".tj":                     true,
	".tk":                     true,
	".tl":                     true,
	".tm":                     true,
	".tn":                     true,
	".to":                     true,
	".tp":                     true,
	".tr":                     true,
	".travel":                 true,
	".tt":                     true,
	".tv":                     true,
	".tw":                     true,
	".tz":                     true,
	".ua":                     true,
	".ug":                     true,
	".uk":                     true,
	".us":                     true,
	".uy":                     true,
	".uz":                     true,
	".va":                     true,
	".vc":                     true,
	".ve":                     true,
	".vg":                     true,
	".vi":                     true,
	".vn":                     true,
	".vu":                     true,
	".wf":                     true,
	".ws":                     true,
	".xn--0zwm56d":            true,
	".xn--11b5bs3a9aj6g":      true,
	".xn--3e0b707e":           true,
	".xn--45brj9c":            true,
	".xn--80akhbyknj4f":       true,
	".xn--80ao21a":            true,
	".xn--90a3ac":             true,
	".xn--9t4b11yi5a":         true,
	".xn--clchc0ea0b2g2a9gcd": true,
	".xn--deba0ad":            true,
	".xn--fiqs8s":             true,
	".xn--fiqz9s":             true,
	".xn--fpcrj9c3d":          true,
	".xn--fzc2c9e2c":          true,
	".xn--g6w251d":            true,
	".xn--gecrj9c":            true,
	".xn--h2brj9c":            true,
	".xn--hgbk6aj7f53bba":     true,
	".xn--hlcj6aya9esc7a":     true,
	".xn--j6w193g":            true,
	".xn--jxalpdlp":           true,
	".xn--kgbechtv":           true,
	".xn--kprw13d":            true,
	".xn--kpry57d":            true,
	".xn--lgbbat1ad8j":        true,
	".xn--mgb9awbf":           true,
	".xn--mgbaam7a8h":         true,
	".xn--mgbayh7gpa":         true,
	".xn--mgbbh1a71e":         true,
	".xn--mgbc0a9azcg":        true,
	".xn--mgberp4a5d4ar":      true,
	".xn--mgbx4cd0ab":         true,
	".xn--o3cw4h":             true,
	".xn--ogbpf8fl":           true,
	".xn--p1ai":               true,
	".xn--pgbs0dh":            true,
	".xn--s9brj9c":            true,
	".xn--wgbh1c":             true,
	".xn--wgbl6a":             true,
	".xn--xkc2al3hye2a":       true,
	".xn--xkc2dl3a5ee0h":      true,
	".xn--yfro4i67o":          true,
	".xn--ygbi2ammx":          true,
	".xn--zckzah":             true,
	".xxx":                    true,
	".ye":                     true,
	".yt":                     true,
	".za":                     true,
	".zm":                     true,
	".zw":                     true,
}

var validHost = regexp.MustCompile(`^[-a-z0-9]+(?:\.[-a-z0-9]+)+$`)
var validPathElement = regexp.MustCompile(`^[-A-Za-z0-9~+][-A-Za-z0-9_.]*$`)

func isValidPathElement(s string) bool {
	return validPathElement.MatchString(s) && s != "testdata"
}

// IsValidRemotePath returns true if importPath is structurally valid for "go get".
func IsValidRemotePath(importPath string) bool {
	parts := strings.Split(importPath, "/")
	if len(parts) <= 1 {
		// Import path must contain at least one "/".
		return false
	}

	if !validTLD[path.Ext(parts[0])] {
		return false
	}

	if !validHost.MatchString(parts[0]) {
		return false
	}

	for _, part := range parts[1:] {
		if !isValidPathElement(part) {
			return false
		}
	}
	return true
}

func IsGoTool(path string) bool {
	return strings.HasPrefix(path, "golang.org/x/tools/cmd/") ||
		strings.HasPrefix(path, "code.google.com/p/go.tools/cmd/")
}

// Convert string to specify type.
type StrTo string

func (f StrTo) Exist() bool {
	return string(f) != string(0x1E)
}

func (f StrTo) Uint8() (uint8, error) {
	v, err := strconv.ParseUint(f.String(), 10, 8)
	return uint8(v), err
}

func (f StrTo) Int() (int, error) {
	v, err := strconv.ParseInt(f.String(), 10, 32)
	return int(v), err
}

func (f StrTo) Int64() (int64, error) {
	v, err := strconv.ParseInt(f.String(), 10, 64)
	return int64(v), err
}

func (f StrTo) MustUint8() uint8 {
	v, _ := f.Uint8()
	return v
}

func (f StrTo) MustInt() int {
	v, _ := f.Int()
	return v
}

func (f StrTo) MustInt64() int64 {
	v, _ := f.Int64()
	return v
}

func (f StrTo) String() string {
	if f.Exist() {
		return string(f)
	}
	return ""
}

// Convert any type to string.
func ToStr(value interface{}, args ...int) (s string) {
	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case float32:
		s = strconv.FormatFloat(float64(v), 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 32))
	case float64:
		s = strconv.FormatFloat(v, 'f', argInt(args).Get(0, -1), argInt(args).Get(1, 64))
	case int:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int8:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int16:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int32:
		s = strconv.FormatInt(int64(v), argInt(args).Get(0, 10))
	case int64:
		s = strconv.FormatInt(v, argInt(args).Get(0, 10))
	case uint:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint8:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint16:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint32:
		s = strconv.FormatUint(uint64(v), argInt(args).Get(0, 10))
	case uint64:
		s = strconv.FormatUint(v, argInt(args).Get(0, 10))
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		s = fmt.Sprintf("%v", v)
	}
	return s
}

type argInt []int

func (a argInt) Get(i int, args ...int) (r int) {
	if i >= 0 && i < len(a) {
		r = a[i]
	} else if len(args) > 0 {
		r = args[0]
	}
	return
}

//GetTempDir generates and returns time-based unique temporary path.
func GetTempDir() string {
	return path.Join(os.TempDir(), ToStr(time.Now().Nanosecond()))
}

var UserAgent = "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/29.0.1541.0 Safari/537.36"

// HttpGet gets the specified resource. ErrNotFound is returned if the
// server responds with status 404.
func HttpGet(client *http.Client, url string, header http.Header) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	for k, vs := range header {
		req.Header[k] = vs
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fail to make request: %v", err)
	}
	if resp.StatusCode == 200 {
		return resp.Body, nil
	}
	resp.Body.Close()
	if resp.StatusCode == 404 { // 403 can be rate limit error.  || resp.StatusCode == 403 {
		err = fmt.Errorf("resource not found: %s", url)
	} else {
		err = fmt.Errorf("get %s -> %d", url, resp.StatusCode)
	}
	return nil, err
}

// HttpGetBytes gets the specified resource. ErrNotFound is returned if the server
// responds with status 404.
func HttpGetBytes(client *http.Client, url string, header http.Header) ([]byte, error) {
	rc, err := HttpGet(client, url, header)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

// HttpGetJSON gets the specified resource and mapping to struct.
// ErrNotFound is returned if the server responds with status 404.
func HttpGetJSON(client *http.Client, url string, v interface{}) error {
	rc, err := HttpGet(client, url, nil)
	if err != nil {
		return err
	}
	defer rc.Close()
	err = json.NewDecoder(rc).Decode(v)
	if _, ok := err.(*json.SyntaxError); ok {
		return fmt.Errorf("JSON syntax error at %s", url)
	}
	return nil
}
