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
)

type candidate struct {
	relativePath string
	absolutePath string
	prefix       string
}

type walkDirFunc func(string, fs.WalkDirFunc) error

func commentPrefix(path string) (string, bool) {
	switch filepath.Ext(path) {
	case ".go", ".mod":
		return "//", true
	case ".yml", ".yaml":
		return "#", true
	default:
		return "", false
	}
}

func walkCandidates(root string, walkDir walkDirFunc) ([]candidate, error) {
	candidates := make([]candidate, 0)
	operationErrors := make([]error, 0)
	err := walkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			operationErrors = append(operationErrors, fmt.Errorf("walk %s: %w", traversalPath(root, path), walkErr))

			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		prefix, supported := commentPrefix(path)
		if !supported {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			operationErrors = append(operationErrors, fmt.Errorf("walk %s: %w", traversalPath(root, path), err))

			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			operationErrors = append(operationErrors, fmt.Errorf("walk %s: make path relative: %w", filepath.ToSlash(path), err))

			return nil
		}
		candidates = append(candidates, candidate{
			relativePath: filepath.ToSlash(relativePath),
			absolutePath: path,
			prefix:       prefix,
		})

		return nil
	})
	if err != nil {
		operationErrors = append(operationErrors, fmt.Errorf("walk %s: %w", traversalPath(root, root), err))
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].relativePath < candidates[right].relativePath
	})

	return candidates, errors.Join(operationErrors...)
}

func traversalPath(root, path string) string {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(relativePath)
}

func updateCandidate(candidate candidate, options Options) (bool, error) {
	info, err := os.Lstat(candidate.absolutePath)
	if err != nil {
		return false, err
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, nil
	}
	contents, err := os.ReadFile(candidate.absolutePath)
	if err != nil {
		return false, err
	}
	if hasLicenseHeader(contents) {
		return false, nil
	}
	if options.Mode == ModeCheck {
		return true, nil
	}

	updated := []byte(renderHeader(candidate.prefix, options.Year) + string(contents))
	return true, atomicReplace(candidate.absolutePath, updated, info.Mode().Perm())
}

func atomicReplace(path string, contents []byte, mode fs.FileMode) (operationErr error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".license-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := temporary.Close(); closeErr != nil {
				operationErr = errors.Join(operationErr, fmt.Errorf("close temporary file: %w", closeErr))
			}
		}
		if temporaryPath != "" {
			if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
				operationErr = errors.Join(operationErr, fmt.Errorf("remove temporary file: %w", removeErr))
			}
		}
	}()

	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	closed = true
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	temporaryPath = ""

	return nil
}
