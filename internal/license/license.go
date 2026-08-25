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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Mode uint8

const (
	ModeFix Mode = iota
	ModeCheck
)

type Options struct {
	Root string
	Year int
	Mode Mode
}

type Result struct {
	Changed []string
	Missing []string
}

type candidateUpdater func(candidate, Options) (bool, error)

func Run(options Options) (Result, error) {
	return run(options, filepath.WalkDir)
}

func run(options Options, walkDir walkDirFunc) (Result, error) {
	if options.Root == "" {
		return Result{}, errors.New("license root is required")
	}
	if options.Year < 1 {
		return Result{}, errors.New("license year must be at least 1")
	}
	if options.Mode != ModeFix && options.Mode != ModeCheck {
		return Result{}, fmt.Errorf("unknown license mode %d", options.Mode)
	}

	root, err := filepath.Abs(options.Root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve license root: %w", err)
	}
	options.Root = filepath.Clean(root)
	options.Root, err = filepath.EvalSymlinks(options.Root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve license root: %w", err)
	}
	if err := validateRepositoryRoot(options.Root); err != nil {
		return Result{}, err
	}

	result := Result{}
	operationErrors := make([]error, 0)
	changed, missing, err := updateLicenseArtifact(options)
	appendArtifactResult(&result, options.Mode, "LICENSE", changed, missing, err, &operationErrors)
	changed, missing, err = updateLicenseHeaderArtifact(options)
	appendArtifactResult(&result, options.Mode, "LICENSE.header", changed, missing, err, &operationErrors)

	candidates, err := walkCandidates(options.Root, walkDir)
	if err != nil {
		operationErrors = append(operationErrors, err)
	}
	candidateResult, candidateErr := updateCandidates(candidates, options, updateCandidate)
	result.Changed = append(result.Changed, candidateResult.Changed...)
	result.Missing = append(result.Missing, candidateResult.Missing...)
	if candidateErr != nil {
		operationErrors = append(operationErrors, candidateErr)
	}

	result.Changed = sortAndDeduplicate(result.Changed)
	result.Missing = sortAndDeduplicate(result.Missing)

	return result, errors.Join(operationErrors...)
}

func validateRepositoryRoot(root string) error {
	info, err := os.Lstat(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Errorf("repository root %q: %w", root, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("repository root %q does not contain a regular go.mod", root)
	}

	return nil
}

func updateLicenseArtifact(options Options) (bool, bool, error) {
	path := filepath.Join(options.Root, "LICENSE")
	_, err := os.Lstat(path)
	if err == nil {
		return false, false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return false, false, err
	}
	if options.Mode == ModeCheck {
		return false, true, nil
	}

	return true, false, atomicReplace(path, []byte(apacheLicenseText), 0o644)
}

func updateLicenseHeaderArtifact(options Options) (bool, bool, error) {
	path := filepath.Join(options.Root, "LICENSE.header")
	expected := []byte(strings.TrimSuffix(renderHeader("", options.Year), "\n"))
	info, err := os.Lstat(path)
	mode := fs.FileMode(0o644)
	if err == nil {
		if !info.Mode().IsRegular() {
			return false, false, fmt.Errorf("LICENSE.header is not a regular file")
		}
		mode = info.Mode().Perm()
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return false, false, readErr
		}
		if string(contents) == string(expected) {
			return false, false, nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, false, err
	}

	if options.Mode == ModeCheck {
		return false, true, nil
	}

	return true, false, atomicReplace(path, expected, mode)
}

func appendArtifactResult(result *Result, mode Mode, path string, changed, missing bool, err error, operationErrors *[]error) {
	if err != nil {
		*operationErrors = append(*operationErrors, fmt.Errorf("update %s: %w", path, err))
		return
	}
	if mode == ModeFix && changed {
		result.Changed = append(result.Changed, path)
	}
	if mode == ModeCheck && missing {
		result.Missing = append(result.Missing, path)
	}
}

func updateCandidates(candidates []candidate, options Options, update candidateUpdater) (Result, error) {
	result := Result{}
	operationErrors := make([]error, 0)
	for _, candidate := range candidates {
		needsUpdate, err := update(candidate, options)
		if err != nil {
			operationErrors = append(operationErrors, fmt.Errorf("update %s: %w", candidate.relativePath, err))
			continue
		}
		if !needsUpdate {
			continue
		}
		if options.Mode == ModeFix {
			result.Changed = append(result.Changed, candidate.relativePath)
		} else {
			result.Missing = append(result.Missing, candidate.relativePath)
		}
	}

	return result, errors.Join(operationErrors...)
}

func sortAndDeduplicate(paths []string) []string {
	sort.Strings(paths)
	if len(paths) < 2 {
		return paths
	}

	unique := paths[:1]
	for _, path := range paths[1:] {
		if path != unique[len(unique)-1] {
			unique = append(unique, path)
		}
	}

	return unique
}
