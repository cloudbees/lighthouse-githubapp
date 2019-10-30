// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"github.com/cloudbees/jx-tenant-service/pkg/controller/app"
	"strings"

	"github.com/blang/semver"
)

// Build information. Populated at build-time.
var (
	Version   string
	Revision  string
	Sha1      string
	Branch    string
	BuildUser string
	BuildDate string
	GoVersion string
)

// Map provides the iterable version information.
var Map = map[string]string{
	"version":   Version,
	"revision":  Revision,
	"sha1":      Sha1,
	"branch":    Branch,
	"buildUser": BuildUser,
	"buildDate": BuildDate,
	"goVersion": GoVersion,
}

const VersionPrefix = ""

func GetBuildVersion() *string {
	var result = Map["version"]
	return &result
}

func GetCommitHash() *string {
	var result = Map["sha1"]
	return &result
}

func GetSemverVersion() (semver.Version, error) {
	var version = GetBuildVersion()
	return semver.Make(strings.TrimPrefix(*version, VersionPrefix))
}

// GetVersionString returns a formatted string showing the version + sha
func GetVersionString() string {
	var version = GetBuildVersion()
	var commithash = GetCommitHash()

	return "version: v" + *version + "\ncommithash: " + *commithash + "\n"
}

func GetVersionInfo() *app.Info {
	return &app.Info{
		Commithash: GetCommitHash(),
		Version:    GetBuildVersion(),
	}
}
