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
	"sync"
)

type SafeMap struct {
	locker *sync.RWMutex
	data   map[string]bool
}

func (s *SafeMap) Set(verstr string) {
	s.locker.Lock()
	defer s.locker.Unlock()
	s.data[verstr] = true
}

func (s *SafeMap) Get(verstr string) bool {
	s.locker.RLock()
	defer s.locker.RUnlock()
	return s.data[verstr]
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		locker: &sync.RWMutex{},
		data:   make(map[string]bool),
	}
}
