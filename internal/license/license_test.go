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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, root, name, contents string, mode fs.FileMode) string {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), mode))

	return path
}

func TestRunFixCreatesArtifactsAndAddsHeaders(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/fixture\n", 0o644)
	mainPath := writeTestFile(t, root, "main.go", "package main\n", 0o644)

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	require.Equal(t, []string{"LICENSE", "LICENSE.header", "go.mod", "main.go"}, result.Changed)
	require.Empty(t, result.Missing)
	require.Equal(t, apacheLicenseText, readTestFile(t, filepath.Join(root, "LICENSE")))
	require.Equal(t, strings.TrimSuffix(renderHeader("", 2026), "\n"), readTestFile(t, filepath.Join(root, "LICENSE.header")))
	require.Equal(t, renderHeader("//", 2026)+"package main\n", readTestFile(t, mainPath))
}

func TestRunCheckReportsEveryMissingPathWithoutWriting(t *testing.T) {
	root := t.TempDir()
	goModPath := writeTestFile(t, root, "go.mod", "module example.test/fixture\n", 0o644)
	mainPath := writeTestFile(t, root, "main.go", "package main\n", 0o644)
	configPath := writeTestFile(t, root, "config/settings.yaml", "enabled: true\n", 0o644)
	paths := []string{goModPath, mainPath, configPath}
	before := snapshotTestFiles(t, paths)

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeCheck})

	require.NoError(t, err)
	require.Empty(t, result.Changed)
	require.Equal(t, []string{"LICENSE", "LICENSE.header", "config/settings.yaml", "go.mod", "main.go"}, result.Missing)
	for path, contents := range before {
		require.Equal(t, string(contents), readTestFile(t, path))
	}
	_, err = os.Stat(filepath.Join(root, "LICENSE"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	_, err = os.Stat(filepath.Join(root, "LICENSE.header"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRunPreservesExistingLicenseAndHistoricalHeader(t *testing.T) {
	root := t.TempDir()
	licenseContents := "a repository-specific license\n"
	writeTestFile(t, root, "LICENSE", licenseContents, 0o600)
	writeTestFile(t, root, "go.mod", renderHeader("//", 2024)+"module example.test/fixture\n", 0o644)
	historicalContents := renderHeader("//", 2024) + "package main\n"
	mainPath := writeTestFile(t, root, "main.go", historicalContents, 0o644)

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	require.Equal(t, []string{"LICENSE.header"}, result.Changed)
	require.Empty(t, result.Missing)
	require.Equal(t, licenseContents, readTestFile(t, filepath.Join(root, "LICENSE")))
	require.Equal(t, historicalContents, readTestFile(t, mainPath))
}

func TestRunReplacesStaleLicenseHeader(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "LICENSE", "custom license\n", 0o644)
	writeTestFile(t, root, "LICENSE.header", "stale header\n", 0o640)
	writeTestFile(t, root, "go.mod", renderHeader("//", 2026)+"module example.test/fixture\n", 0o644)
	writeTestFile(t, root, "main.go", renderHeader("//", 2026)+"package main\n", 0o644)

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	require.Equal(t, []string{"LICENSE.header"}, result.Changed)
	require.Equal(t, strings.TrimSuffix(renderHeader("", 2026), "\n"), readTestFile(t, filepath.Join(root, "LICENSE.header")))
	info, statErr := os.Stat(filepath.Join(root, "LICENSE.header"))
	require.NoError(t, statErr)
	require.Equal(t, fs.FileMode(0o640), info.Mode().Perm())
}

func TestRunSkipsUnsupportedFilesAndSymlinks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", renderHeader("//", 2026)+"module example.test/fixture\n", 0o644)
	writeTestFile(t, root, "main.go", renderHeader("//", 2026)+"package main\n", 0o644)
	unsupportedPath := writeTestFile(t, root, "notes.txt", "leave this alone\n", 0o644)
	targetPath := writeTestFile(t, t.TempDir(), "target.go", "package target\n", 0o644)
	linkPath := filepath.Join(root, "linked.go")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlinks are unavailable on this platform: %v", err)
	}

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	require.Equal(t, []string{"LICENSE", "LICENSE.header"}, result.Changed)
	require.Equal(t, "leave this alone\n", readTestFile(t, unsupportedPath))
	require.Equal(t, "package target\n", readTestFile(t, targetPath))
	linkInfo, statErr := os.Lstat(linkPath)
	require.NoError(t, statErr)
	require.NotZero(t, linkInfo.Mode()&fs.ModeSymlink)
}

func TestRunCheckFollowsRepositoryRootSymlink(t *testing.T) {
	repositoryRoot := t.TempDir()
	writeTestFile(t, repositoryRoot, "go.mod", "module example.test/fixture\n", 0o644)
	writeTestFile(t, repositoryRoot, "main.go", "package main\n", 0o644)
	root := filepath.Join(t.TempDir(), "repository")
	if err := os.Symlink(repositoryRoot, root); err != nil {
		t.Skipf("symlinks are unavailable on this platform: %v", err)
	}

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeCheck})

	require.NoError(t, err)
	require.Empty(t, result.Changed)
	require.Equal(t, []string{"LICENSE", "LICENSE.header", "go.mod", "main.go"}, result.Missing)
}

func TestRunPreservesFilePermissions(t *testing.T) {
	root := t.TempDir()
	goModPath := writeTestFile(t, root, "go.mod", "module example.test/fixture\n", 0o640)
	mainPath := writeTestFile(t, root, "main.go", "package main\n", 0o750)

	_, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	goModInfo, statErr := os.Stat(goModPath)
	require.NoError(t, statErr)
	require.Equal(t, fs.FileMode(0o640), goModInfo.Mode().Perm())
	mainInfo, statErr := os.Stat(mainPath)
	require.NoError(t, statErr)
	require.Equal(t, fs.FileMode(0o750), mainInfo.Mode().Perm())
}

func TestRunFixIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/fixture\n", 0o644)
	writeTestFile(t, root, "main.go", "package main\n", 0o644)

	_, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})
	require.NoError(t, err)
	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeFix})

	require.NoError(t, err)
	require.Empty(t, result.Changed)
	require.Empty(t, result.Missing)
}

func TestRunSortsFindings(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "go.mod", "module example.test/fixture\n", 0o644)
	writeTestFile(t, root, "b.go", "package fixture\n", 0o644)
	writeTestFile(t, root, "a.yaml", "enabled: true\n", 0o644)

	result, err := Run(Options{Root: root, Year: 2026, Mode: ModeCheck})

	require.NoError(t, err)
	require.Equal(t, []string{"LICENSE", "LICENSE.header", "a.yaml", "b.go", "go.mod"}, result.Missing)
}

func TestRunRejectsNonRepositoryRoot(t *testing.T) {
	result, err := Run(Options{Root: t.TempDir(), Year: 2026, Mode: ModeFix})

	require.Error(t, err)
	require.Empty(t, result.Changed)
	require.Empty(t, result.Missing)
}

func TestRunProcessesCandidatesAndAggregatesTraversalErrors(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "LICENSE", "repository license\n", 0o644)
	writeTestFile(t, root, "LICENSE.header", strings.TrimSuffix(renderHeader("", 2026), "\n"), 0o644)
	writeTestFile(t, root, "go.mod", renderHeader("//", 2026)+"module example.test/fixture\n", 0o644)
	beforePath := writeTestFile(t, root, "before.go", "package fixture\n", 0o644)
	brokenPath := writeTestFile(t, root, "broken.go", "package fixture\n", 0o644)
	afterPath := writeTestFile(t, root, "after.go", "package fixture\n", 0o644)

	beforeInfo, err := os.Lstat(beforePath)
	require.NoError(t, err)
	brokenInfo, err := os.Lstat(brokenPath)
	require.NoError(t, err)
	afterInfo, err := os.Lstat(afterPath)
	require.NoError(t, err)

	walkErr := errors.New("directory entry unavailable")
	infoErr := errors.New("file metadata unavailable")
	walk := func(_ string, visit fs.WalkDirFunc) error {
		require.NoError(t, visit(beforePath, fs.FileInfoToDirEntry(beforeInfo), nil))
		require.NoError(t, visit(filepath.Join(root, "unreadable.go"), nil, walkErr))
		require.NoError(t, visit(brokenPath, failingInfoDirEntry{
			DirEntry: fs.FileInfoToDirEntry(brokenInfo),
			err:      infoErr,
		}, nil))
		require.NoError(t, visit(afterPath, fs.FileInfoToDirEntry(afterInfo), nil))

		return nil
	}

	result, err := run(Options{Root: root, Year: 2026, Mode: ModeFix}, walk)

	require.Equal(t, []string{"after.go", "before.go"}, result.Changed)
	require.Empty(t, result.Missing)
	require.ErrorIs(t, err, walkErr)
	require.ErrorIs(t, err, infoErr)
	require.ErrorContains(t, err, "unreadable.go")
	require.ErrorContains(t, err, "broken.go")
	require.True(t, hasLicenseHeader([]byte(readTestFile(t, beforePath))))
	require.Equal(t, "package fixture\n", readTestFile(t, brokenPath))
	require.True(t, hasLicenseHeader([]byte(readTestFile(t, afterPath))))
}

func TestUpdateCandidatesJoinsAllCandidateErrors(t *testing.T) {
	errOne := errors.New("first update failed")
	errTwo := errors.New("second update failed")
	candidates := []candidate{{relativePath: "one.go"}, {relativePath: "two.go"}}
	var updated []string

	result, err := updateCandidates(candidates, Options{Year: 2026, Mode: ModeFix}, func(item candidate, _ Options) (bool, error) {
		updated = append(updated, item.relativePath)
		switch item.relativePath {
		case "one.go":
			return false, errOne
		case "two.go":
			return false, errTwo
		default:
			return false, nil
		}
	})

	require.Empty(t, result.Changed)
	require.Empty(t, result.Missing)
	require.Equal(t, []string{"one.go", "two.go"}, updated)
	require.ErrorIs(t, err, errOne)
	require.ErrorIs(t, err, errTwo)
	require.ErrorContains(t, err, "update one.go")
	require.ErrorContains(t, err, "update two.go")
}

type failingInfoDirEntry struct {
	fs.DirEntry
	err error
}

func (s failingInfoDirEntry) Info() (fs.FileInfo, error) {
	return nil, s.err
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(contents)
}

func snapshotTestFiles(t *testing.T, paths []string) map[string][]byte {
	t.Helper()

	contents := make(map[string][]byte, len(paths))
	for _, path := range paths {
		contents[path] = []byte(readTestFile(t, path))
	}

	return contents
}
