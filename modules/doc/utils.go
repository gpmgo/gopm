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
	"fmt"
	"go/build"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/gpmgo/gopm/modules/base"
	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

// ParseTarget guesses import path of current package if target is empty,
// otherwise simply returns the value it gets.
func ParseTarget(target string) string {
	if len(target) > 0 {
		return target
	}

	for _, gopath := range base.GetGOPATHs() {
		if strings.HasPrefix(setting.WorkDir, gopath) {
			target = strings.TrimPrefix(setting.WorkDir, path.Join(gopath, "src")+"/")
			log.Info("Guess import path: %s", target)
			return target
		}
	}
	return "."
}

func joinPath(name string, num int) string {
	subdirs := strings.Split(name, "/")
	if len(subdirs) > num {
		return strings.Join(subdirs[:num], "/")
	}
	return name
}

var gopkgPathPattern = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

// GetRootPath returns project root path.
func GetRootPath(name string) string {
	for prefix, num := range setting.RootPathPairs {
		if strings.HasPrefix(name, prefix) {
			return joinPath(name, num)
		}
	}

	if strings.HasPrefix(name, "gopkg.in") {
		m := gopkgPathPattern.FindStringSubmatch(strings.TrimPrefix(name, "gopkg.in"))
		if m == nil {
			return name
		}
		user := m[1]
		repo := m[2]
		return path.Join("gopkg.in", user, repo+"."+m[3])
	}
	return name
}

var standardPath = map[string]bool{
	"builtin": true,

	// go list -f '"{{.ImportPath}}": true,'  std
	"archive/tar":               true,
	"archive/zip":               true,
	"bufio":                     true,
	"bytes":                     true,
	"compress/bzip2":            true,
	"compress/flate":            true,
	"compress/gzip":             true,
	"compress/lzw":              true,
	"compress/zlib":             true,
	"container/heap":            true,
	"container/list":            true,
	"container/ring":            true,
	"context":                   true,
	"crypto":                    true,
	"crypto/aes":                true,
	"crypto/cipher":             true,
	"crypto/des":                true,
	"crypto/dsa":                true,
	"crypto/ecdsa":              true,
	"crypto/elliptic":           true,
	"crypto/hmac":               true,
	"crypto/md5":                true,
	"crypto/rand":               true,
	"crypto/rc4":                true,
	"crypto/rsa":                true,
	"crypto/sha1":               true,
	"crypto/sha256":             true,
	"crypto/sha512":             true,
	"crypto/subtle":             true,
	"crypto/tls":                true,
	"crypto/x509":               true,
	"crypto/x509/pkix":          true,
	"database/sql":              true,
	"database/sql/driver":       true,
	"debug/dwarf":               true,
	"debug/elf":                 true,
	"debug/gosym":               true,
	"debug/macho":               true,
	"debug/pe":                  true,
	"debug/plan9obj":            true,
	"encoding":                  true,
	"encoding/ascii85":          true,
	"encoding/asn1":             true,
	"encoding/base32":           true,
	"encoding/base64":           true,
	"encoding/binary":           true,
	"encoding/csv":              true,
	"encoding/gob":              true,
	"encoding/hex":              true,
	"encoding/json":             true,
	"encoding/pem":              true,
	"encoding/xml":              true,
	"errors":                    true,
	"expvar":                    true,
	"flag":                      true,
	"fmt":                       true,
	"go/ast":                    true,
	"go/build":                  true,
	"go/constant":               true,
	"go/doc":                    true,
	"go/format":                 true,
	"go/importer":               true,
	"go/internal/gccgoimporter": true,
	"go/internal/gcimporter":    true,
	"go/parser":                 true,
	"go/printer":                true,
	"go/scanner":                true,
	"go/token":                  true,
	"go/types":                  true,
	"hash":                      true,
	"hash/adler32":              true,
	"hash/crc32":                true,
	"hash/crc64":                true,
	"hash/fnv":                  true,
	"html":                      true,
	"html/template":             true,
	"image":                     true,
	"image/color":               true,
	"image/color/palette":       true,
	"image/draw":                true,
	"image/gif":                 true,
	"image/internal/imageutil":  true,
	"image/jpeg":                true,
	"image/png":                 true,
	"index/suffixarray":         true,
	"internal/race":             true,
	"internal/singleflight":     true,
	"internal/testenv":          true,
	"internal/trace":            true,
	"io":                        true,
	"io/ioutil":                 true,
	"log":                       true,
	"log/syslog":                true,
	"math":                      true,
	"math/big":                  true,
	"math/cmplx":                true,
	"math/rand":                 true,
	"mime":                      true,
	"mime/multipart":            true,
	"mime/quotedprintable":      true,
	"net":                     true,
	"net/http":                true,
	"net/http/cgi":            true,
	"net/http/cookiejar":      true,
	"net/http/fcgi":           true,
	"net/http/httptest":       true,
	"net/http/httputil":       true,
	"net/http/internal":       true,
	"net/http/pprof":          true,
	"net/internal/socktest":   true,
	"net/mail":                true,
	"net/rpc":                 true,
	"net/rpc/jsonrpc":         true,
	"net/smtp":                true,
	"net/textproto":           true,
	"net/url":                 true,
	"os":                      true,
	"os/exec":                 true,
	"os/signal":               true,
	"os/user":                 true,
	"path":                    true,
	"path/filepath":           true,
	"reflect":                 true,
	"regexp":                  true,
	"regexp/syntax":           true,
	"runtime":                 true,
	"runtime/cgo":             true,
	"runtime/debug":           true,
	"runtime/internal/atomic": true,
	"runtime/internal/sys":    true,
	"runtime/pprof":           true,
	"runtime/race":            true,
	"runtime/trace":           true,
	"sort":                    true,
	"strconv":                 true,
	"strings":                 true,
	"sync":                    true,
	"sync/atomic":             true,
	"syscall":                 true,
	"testing":                 true,
	"testing/iotest":          true,
	"testing/quick":           true,
	"text/scanner":            true,
	"text/tabwriter":          true,
	"text/template":           true,
	"text/template/parse":     true,
	"time":                    true,
	"unicode":                 true,
	"unicode/utf16":           true,
	"unicode/utf8":            true,
	"unsafe":                  true,
}

var goRepoPath = map[string]bool{}

func init() {
	for p := range standardPath {
		for {
			goRepoPath[p] = true
			i := strings.LastIndex(p, "/")
			if i < 0 {
				break
			}
			p = p[:i]
		}
	}
}

// IsGoRepoPath returns true if package is from standard library.
func IsGoRepoPath(importPath string) bool {
	return goRepoPath[importPath]
}

// ListImports checks and returns a list of imports of given import path and options.
func ListImports(importPath, rootPath, vendorPath, srcPath, tags string, isTest bool) ([]string, error) {
	oldGOPATH := os.Getenv("GOPATH")
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}

	ctxt := build.Default
	ctxt.BuildTags = strings.Split(tags, " ")
	ctxt.GOPATH = vendorPath + sep + oldGOPATH
	if setting.Debug {
		log.Debug("Import/root path: %s : %s", importPath, rootPath)
		log.Debug("Context GOPATH: %s", ctxt.GOPATH)
		log.Debug("Source path: %s", srcPath)
	}
	pkg, err := ctxt.Import(importPath, srcPath, build.AllowBinary)
	if err != nil {
		if _, ok := err.(*build.NoGoError); !ok {
			return nil, fmt.Errorf("fail to get imports(%s): %v", importPath, err)
		}
		log.Warn("Getting imports: %v", err)
	}

	rawImports := pkg.Imports
	numImports := len(rawImports)
	if isTest {
		rawImports = append(rawImports, pkg.TestImports...)
		numImports = len(rawImports)
	}
	imports := make([]string, 0, numImports)
	for _, name := range rawImports {
		if IsGoRepoPath(name) {
			continue
		} else if strings.HasPrefix(name, rootPath) {
			moreImports, err := ListImports(name, rootPath, vendorPath, srcPath, tags, isTest)
			if err != nil {
				return nil, err
			}
			imports = append(imports, moreImports...)
			continue
		}
		if setting.Debug {
			log.Debug("Found dependency: %s", name)
		}
		imports = append(imports, name)
	}
	return imports, nil
}

// GetVcsName checks whether dirPath has .git .hg .svn else return ""
func GetVcsName(dirPath string) string {
	switch {
	case base.IsExist(path.Join(dirPath, ".git")):
		return "git"
	case base.IsExist(path.Join(dirPath, ".hg")):
		return "hg"
	case base.IsExist(path.Join(dirPath, ".svn")):
		return "svn"
	}
	return ""
}
