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

package doc

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gpmgo/gopm/modules/log"
	"github.com/gpmgo/gopm/modules/setting"
)

var (
	dialTimeout    = flag.Duration("dial_timeout", 10*time.Second, "Timeout for dialing an HTTP connection.")
	requestTimeout = flag.Duration("request_timeout", 20*time.Second, "Time out for roundtripping an HTTP request.")
)

func timeoutDial(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, *dialTimeout)
}

type transport struct {
	t http.Transport
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := time.AfterFunc(*requestTimeout, func() {
		t.t.CancelRequest(req)
		log.Warn("Canceled request for %s, please interrupt the program.", req.URL)
	})
	defer timer.Stop()
	resp, err := t.t.RoundTrip(req)
	return resp, err
}

func (t *transport) SetProxy(proxy string) error {
	if len(proxy) == 0 {
		return nil
	}

	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		if setting.LibraryMode {
			return fmt.Errorf("Fail to set HTTP proxy: %v", err)
		}
		log.Error("", "Fail to set HTTP proxy:")
		log.Fatal("", "\t"+err.Error())
	}
	t.t.Proxy = http.ProxyURL(proxyUrl)
	return nil
}

var (
	httpTransport = &transport{
		t: http.Transport{
			Dial: timeoutDial,
			ResponseHeaderTimeout: *requestTimeout / 2,
		},
	}
	HttpClient = &http.Client{Transport: httpTransport}
)

func SetProxy(proxy string) error {
	return httpTransport.SetProxy(proxy)
}
