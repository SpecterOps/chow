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
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/specterops/chow/internal/license"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getwd, time.Now))
}

func run(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	getwd func() (string, error),
	now func() time.Time,
) int {
	flags := flag.NewFlagSet("license", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check license requirements without changing files")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return 2
	}

	root, err := getwd()
	if err != nil {
		fmt.Fprintf(stderr, "license: %v\n", err)
		return 2
	}

	mode := license.ModeFix
	if *check {
		mode = license.ModeCheck
	}
	result, err := license.Run(license.Options{
		Root: root,
		Year: now().Year(),
		Mode: mode,
	})

	if *check {
		for _, path := range result.Missing {
			fmt.Fprintf(stderr, "missing license requirement: %s\n", path)
		}
	} else {
		for _, path := range result.Changed {
			fmt.Fprintf(stdout, "updated %s\n", path)
		}
	}

	if err != nil {
		fmt.Fprintf(stderr, "license: %v\n", err)

		return 2
	}

	if *check {
		if len(result.Missing) == 0 {
			fmt.Fprintln(stdout, "license check passed")

			return 0
		}

		return 1
	}
	if len(result.Changed) == 0 {
		fmt.Fprintln(stdout, "license files are current")
	}

	return 0
}
