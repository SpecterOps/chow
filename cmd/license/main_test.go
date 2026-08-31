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
package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/specterops/chow/internal/license"
	"github.com/stretchr/testify/require"
)

func TestRunCheckReportsCompliantRepository(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod":  "module example.test/fixture\n",
		"main.go": "package main\n",
	})
	_, err := license.Run(license.Options{Root: root, Year: 2026, Mode: license.ModeFix})
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"-check"}, &stdout, &stderr, fixedGetwd(root), fixedNow)

	require.Equal(t, 0, exitCode)
	require.Equal(t, "license check passed\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunCheckReportsMissingRequirementsWithoutWriting(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod":        "module example.test/fixture\n",
		"z.go":          "package fixture\n",
		"config/a.yaml": "enabled: true\n",
	})
	paths := []string{
		filepath.Join(root, "go.mod"),
		filepath.Join(root, "z.go"),
		filepath.Join(root, "config", "a.yaml"),
	}
	before := snapshotFiles(t, paths)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"-check"}, &stdout, &stderr, fixedGetwd(root), fixedNow)

	require.Equal(t, 1, exitCode)
	require.Empty(t, stdout.String())
	require.Equal(t, "missing license requirement: LICENSE\n"+
		"missing license requirement: LICENSE.header\n"+
		"missing license requirement: config/a.yaml\n"+
		"missing license requirement: go.mod\n"+
		"missing license requirement: z.go\n", stderr.String())
	for path, contents := range before {
		actual, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.Equal(t, contents, actual)
	}
	_, statErr := os.Stat(filepath.Join(root, "LICENSE"))
	require.ErrorIs(t, statErr, fs.ErrNotExist)
}

func TestRunFixUpdatesRequirementsInSortedOrder(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod":        "module example.test/fixture\n",
		"z.go":          "package fixture\n",
		"config/a.yaml": "enabled: true\n",
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr, fixedGetwd(root), fixedNow)

	require.Equal(t, 0, exitCode)
	require.Equal(t, "updated LICENSE\n"+
		"updated LICENSE.header\n"+
		"updated config/a.yaml\n"+
		"updated go.mod\n"+
		"updated z.go\n", stdout.String())
	require.Empty(t, stderr.String())
	result, err := license.Run(license.Options{Root: root, Year: 2026, Mode: license.ModeCheck})
	require.NoError(t, err)
	require.Empty(t, result.Missing)
}

func TestRunFixReportsCurrentFiles(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod":  "module example.test/fixture\n",
		"main.go": "package main\n",
	})
	_, err := license.Run(license.Options{Root: root, Year: 2026, Mode: license.ModeFix})
	require.NoError(t, err)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr, fixedGetwd(root), fixedNow)

	require.Equal(t, 0, exitCode)
	require.Equal(t, "license files are current\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"-unknown"}, &stdout, &stderr, fixedGetwd(t.TempDir()), fixedNow)

	require.Equal(t, 2, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "flag provided but not defined: -unknown\n")
	require.Contains(t, stderr.String(), "Usage of license:\n")
}

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"unexpected"}, &stdout, &stderr, fixedGetwd(t.TempDir()), fixedNow)

	require.Equal(t, 2, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "Usage of license:\n")
}

func TestRunReportsGetwdError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr, func() (string, error) {
		return "", errors.New("working directory unavailable")
	}, fixedNow)

	require.Equal(t, 2, exitCode)
	require.Empty(t, stdout.String())
	require.Equal(t, "license: working directory unavailable\n", stderr.String())
}

func TestRunReportsOperationalError(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr, fixedGetwd(root), fixedNow)

	require.Equal(t, 2, exitCode)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "license: repository root ")
	require.Contains(t, stderr.String(), "go.mod")
}

func TestRunCheckPrintsFindingsBeforeOperationalError(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod": "module example.test/fixture\n",
	})
	require.NoError(t, os.Mkdir(filepath.Join(root, "LICENSE.header"), 0o755))
	var output bytes.Buffer

	exitCode := run([]string{"-check"}, &output, &output, fixedGetwd(root), fixedNow)

	require.Equal(t, 2, exitCode)
	require.Equal(t, "missing license requirement: LICENSE\n"+
		"missing license requirement: go.mod\n"+
		"license: update LICENSE.header: LICENSE.header is not a regular file\n", output.String())
	require.Equal(t, "module example.test/fixture\n", readFile(t, filepath.Join(root, "go.mod")))
	_, err := os.Stat(filepath.Join(root, "LICENSE"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRunFixPrintsChangesBeforeOperationalError(t *testing.T) {
	root := newRepository(t, map[string]string{
		"go.mod": "module example.test/fixture\n",
	})
	require.NoError(t, os.Mkdir(filepath.Join(root, "LICENSE.header"), 0o755))
	var output bytes.Buffer

	exitCode := run(nil, &output, &output, fixedGetwd(root), fixedNow)

	require.Equal(t, 2, exitCode)
	require.Equal(t, "updated LICENSE\n"+
		"updated go.mod\n"+
		"license: update LICENSE.header: LICENSE.header is not a regular file\n", output.String())
	require.Contains(t, readFile(t, filepath.Join(root, "go.mod")), "SPDX-License-Identifier: Apache-2.0")
}

func newRepository(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	}

	return root
}

func snapshotFiles(t *testing.T, paths []string) map[string][]byte {
	t.Helper()

	contents := make(map[string][]byte, len(paths))
	for _, path := range paths {
		actual, err := os.ReadFile(path)
		require.NoError(t, err)
		contents[path] = actual
	}

	return contents
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(contents)
}

func fixedGetwd(root string) func() (string, error) {
	return func() (string, error) {
		return root, nil
	}
}

func fixedNow() time.Time {
	return time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
}
