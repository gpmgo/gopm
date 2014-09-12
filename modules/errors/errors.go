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

package errors

import (
	"github.com/gpmgo/gopm/modules/setting"
)

type ErrDownload struct {
	pkgName string
}

func (err ErrDownload) Error() string {
	return err.pkgName
}

func NewErrDownload(name string) ErrDownload {
	return ErrDownload{name}
}

type ErrInvalidPackage struct {
	pkgName string
}

func (err ErrInvalidPackage) Error() string {
	return err.pkgName
}

func NewErrInvalidPackage(name string) ErrInvalidPackage {
	return ErrInvalidPackage{name}
}

type ErrCopyResource struct {
	resName string
}

func (err ErrCopyResource) Error() string {
	return err.resName
}

func NewErrCopyResource(name string) ErrCopyResource {
	return ErrCopyResource{name}
}

func SetError(err error) {
	setting.RuntimeError.HasError = true
	setting.RuntimeError.Fatal = err
}

func AppendError(err error) {
	setting.RuntimeError.HasError = true
	setting.RuntimeError.Errors = append(setting.RuntimeError.Errors, err)
}
