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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:   "Go",
			prefix: "//",
			expected: "// Copyright 2026 Specter Ops, Inc.\n" +
				"//\n" +
				"// Licensed under the Apache License, Version 2.0\n",
		},
		{
			name:   "YAML",
			prefix: "#",
			expected: "# Copyright 2026 Specter Ops, Inc.\n" +
				"#\n" +
				"# Licensed under the Apache License, Version 2.0\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := renderHeader(testCase.prefix, 2026)
			require.True(t, strings.HasPrefix(actual, testCase.expected))
			require.Contains(t, actual, testCase.prefix+" SPDX-License-Identifier: Apache-2.0\n")
			require.True(t, strings.HasSuffix(actual, "\n\n"))
		})
	}
}

func TestRenderHeaderWithoutPrefixProducesUncommentedLicense(t *testing.T) {
	t.Parallel()

	expected := "Copyright 2026 Specter Ops, Inc.\n" +
		"\n" +
		"Licensed under the Apache License, Version 2.0\n"
	actual := renderHeader("", 2026)

	require.True(t, strings.HasPrefix(actual, expected))
	require.Contains(t, actual, "SPDX-License-Identifier: Apache-2.0\n")
	require.True(t, strings.HasSuffix(actual, "\n\n"))
}

func TestHasLicenseHeaderOnlySearchesFirstTwentyLines(t *testing.T) {
	t.Parallel()

	marker := "// SPDX-License-Identifier: Apache-2.0\n"
	require.True(t, hasLicenseHeader([]byte(strings.Repeat("line\n", 19)+marker)))
	require.False(t, hasLicenseHeader([]byte(strings.Repeat("line\n", 20)+marker)))
}

func TestHasLicenseHeaderReturnsFalseWithoutMarker(t *testing.T) {
	t.Parallel()

	require.False(t, hasLicenseHeader([]byte("// Copyright 2026 Specter Ops, Inc.\n")))
}
