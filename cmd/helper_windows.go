// Copyright 2013 Unknwon
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

package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Unknwon/com"

	"github.com/gpmgo/gopm/modules/setting"
)

func makeLink(srcPath, destPath string) error {
	srcPath = strings.Replace(srcPath, "/", "\\", -1)
	destPath = strings.Replace(destPath, "/", "\\", -1)

	// Check if Windows version is XP.
	if getWindowsVersion() >= 6 {
		os.Remove(destPath)
		_, stderr, err := com.ExecCmd("cmd", "/c", "mklink", "/j", destPath, srcPath)
		if err != nil {
			return errors.New(stderr)
		}
		return nil
	}

	// XP.
	setting.IsWindowsXP = true
	// if both are ntfs file system
	if volumnType(srcPath) == "NTFS" && volumnType(destPath) == "NTFS" {
		// if has junction command installed
		file, err := exec.LookPath("junction")
		if err == nil {
			path, _ := filepath.Abs(file)
			if com.IsFile(path) {
				_, stderr, err := com.ExecCmd("cmd", "/c", "junction", destPath, srcPath)
				if err != nil {
					return errors.New(stderr)
				}
				return nil
			}
		}
	}
	os.RemoveAll(destPath)
	return com.CopyDir(srcPath, destPath, func(filePath string) bool {
		return strings.Contains(filePath, setting.VENDOR)
	})
}

func volumnType(dir string) string {
	pd := dir[:3]
	dll := syscall.MustLoadDLL("kernel32.dll")
	GetVolumeInformation := dll.MustFindProc("GetVolumeInformationW")

	var volumeNameSize uint32 = 260
	var nFileSystemNameSize, lpVolumeSerialNumber uint32
	var lpFileSystemFlags, lpMaximumComponentLength uint32
	var lpFileSystemNameBuffer, volumeName [260]byte
	var ps *uint16 = syscall.StringToUTF16Ptr(pd)

	_, _, _ = GetVolumeInformation.Call(uintptr(unsafe.Pointer(ps)),
		uintptr(unsafe.Pointer(&volumeName)),
		uintptr(volumeNameSize),
		uintptr(unsafe.Pointer(&lpVolumeSerialNumber)),
		uintptr(unsafe.Pointer(&lpMaximumComponentLength)),
		uintptr(unsafe.Pointer(&lpFileSystemFlags)),
		uintptr(unsafe.Pointer(&lpFileSystemNameBuffer)),
		uintptr(unsafe.Pointer(&nFileSystemNameSize)), 0)

	var bytes []byte
	if lpFileSystemNameBuffer[6] == 0 {
		bytes = []byte{lpFileSystemNameBuffer[0], lpFileSystemNameBuffer[2],
			lpFileSystemNameBuffer[4]}
	} else {
		bytes = []byte{lpFileSystemNameBuffer[0], lpFileSystemNameBuffer[2],
			lpFileSystemNameBuffer[4], lpFileSystemNameBuffer[6]}
	}

	return string(bytes)
}

func getWindowsVersion() int {
	dll := syscall.MustLoadDLL("kernel32.dll")
	p := dll.MustFindProc("GetVersion")
	v, _, _ := p.Call()
	return int(byte(v))
}
