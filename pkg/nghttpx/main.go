/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/**
 * Copyright 2016, Z Lab Corporation. All rights reserved.
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

package nghttpx

import (
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/util/flowcontrol"
)

var (
	// Base directory that contains the mounted secrets with TLS certificates, keys and
	tlsDirectory = "/etc/nghttpx-tls"
)

// nghttpxConfiguration is set of configuration we apply to nghttpx instance.
type NghttpxConfiguration struct {
	// https://nghttp2.org/documentation/nghttpx.1.html#cmdoption-nghttpx-n
	// Set the number of worker threads.
	Workers string
	// ExtraConfig is the extra configurations in a format that nghttpx accepts in --conf.
	ExtraConfig string
}

// Manager ...
type Manager struct {
	// nghttpx main configuration file path
	ConfigFile string
	// nghttpx backend configuration file path
	BackendConfigFile string
	// httpClient is used to issue backend API request to nghttpx
	httpClient *http.Client

	reloadRateLimiter flowcontrol.RateLimiter

	// template loaded ready to be used to generate the nghttpx configuration file
	template *template.Template

	// template for backend configuraiton.  This is a part of
	// nghttpx configuration, and included from main one (template
	// above).  We have separate template for backend to change
	// backend configuration without reloading nghttpx if main
	// configuration has not changed.
	backendTemplate *template.Template

	reloadLock *sync.Mutex
}

// defaultConfiguration returns the default configuration contained
// in the file default-conf.json
func newDefaultNghttpxCfg() NghttpxConfiguration {
	cfg := NghttpxConfiguration{
		Workers: strconv.Itoa(runtime.NumCPU()),
	}

	return cfg
}

// NewManager ...
func NewManager() *Manager {
	ngx := &Manager{
		ConfigFile:        "/etc/nghttpx/nghttpx.conf",
		BackendConfigFile: "/etc/nghttpx/nghttpx-backend.conf",
		reloadLock:        &sync.Mutex{},
		reloadRateLimiter: flowcontrol.NewTokenBucketRateLimiter(0.1, 1),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}

	ngx.createCertsDir(tlsDirectory)

	ngx.loadTemplate()

	return ngx
}

func (nghttpx *Manager) createCertsDir(base string) {
	if err := os.Mkdir(base, os.ModeDir); err != nil {
		if os.IsExist(err) {
			glog.Infof("%v already exists", err)
			return
		}
		glog.Fatalf("Couldn't create directory %v: %v", base, err)
	}
}
