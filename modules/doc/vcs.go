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
	"github.com/Unknwon/com"
)

var defaultTags = map[string]string{"git": MASTER, "hg": DEFAULT, "svn": TRUNK}

func bestTag(tags map[string]string, defaultTag string) (string, string, error) {
	if commit, ok := tags[defaultTag]; ok {
		return defaultTag, commit, nil
	}
	return "", "", com.NotFoundError{"Tag or branch not found."}
}
