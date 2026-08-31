// Copyright 2026 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0
package license

import (
	_ "embed"
	"strconv"
	"strings"
)

const (
	spdxMarker    = "SPDX-License-Identifier: Apache-2.0"
	headerYearTag = "XXXX"
	headerLines   = 20
)

//go:embed templates/LICENSE.txt
var apacheLicenseText string

//go:embed templates/header.txt
var headerTemplate string

func renderHeader(prefix string, year int) string {
	text := strings.Replace(headerTemplate, headerYearTag, strconv.Itoa(year), 1)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if prefix != "" {
		for index, line := range lines {
			if line == "" {
				lines[index] = prefix
			} else {
				lines[index] = prefix + " " + line
			}
		}
	}

	return strings.Join(lines, "\n") + "\n\n"
}

func hasLicenseHeader(contents []byte) bool {
	lines := strings.SplitN(string(contents), "\n", headerLines+1)
	if len(lines) > headerLines {
		lines = lines[:headerLines]
	}

	return strings.Contains(strings.Join(lines, "\n"), spdxMarker)
}
