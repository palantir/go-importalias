// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package importalias_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/go-importalias/importalias"
)

func TestImportAliasNoError(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, tc := range []struct {
		name       string
		getPkgArgs func(projectDir string) []string
		files      []gofiles.GoFileSpec
	}{
		{
			name: "no error",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
			},
		},
		{
			name: "no error for multiple files that use the same alias for an import",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import foo "fmt"; func Bar(){ foo.Println() }`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other not using an alias",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import "fmt"; func Bar(){ fmt.Println() }`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other using _",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import _ "fmt"; func Bar(){}`,
				},
			},
		},
		{
			name: "no error if multiple files import same package with one using alias and other using .",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import . "fmt"; func Bar(){}`,
				},
			},
		},
		{
			name: "no error if multiple files import different packages using the same alias",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import foo "io"; func Bar(){ var w foo.Writer; _ = w }`,
				},
			},
		},
	} {
		projectDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		_, err = gofiles.Write(projectDir, tc.files)
		require.NoError(t, err)

		args := tc.getPkgArgs(projectDir)

		buf := bytes.Buffer{}
		importAliasErr := importalias.Run(args, true, &buf)
		assert.NoError(t, importAliasErr, "Case %d (%s)", i, tc.name)
		assert.Equal(t, "", buf.String(), "Case %d (%s)", i, tc.name)
	}
}

func TestImportAliasError(t *testing.T) {
	tmpDir, cleanup, err := dirs.TempDir(".", "")
	defer cleanup()
	require.NoError(t, err)

	for i, tc := range []struct {
		name          string
		getPkgArgs    func(projectDir string) []string
		files         []gofiles.GoFileSpec
		regularOutput func(projectDir string, files map[string]gofiles.GoFile) []string
		verboseOutput func(projectDir string, files map[string]gofiles.GoFile) []string
	}{
		{
			name: "error if multiple files import the same package using a different alias",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
			},
			regularOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					path.Join(projectDir, "bar/bar.go") + `:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					path.Join(projectDir, "foo.go") + `:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					``,
				}
			},
			verboseOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					`"fmt" is imported using multiple different aliases:`,
					`	bar (1 file):`,
					`		` + path.Join(projectDir, "bar/bar.go") + `:1:21`,
					`	foo (1 file):`,
					`		` + path.Join(projectDir, "foo.go") + `:1:22`,
					``,
				}
			},
		},
		{
			name: "error if multiple files import the same package using a different alias for multiple packages",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
					path.Join(projectDir, "baz"),
					path.Join(projectDir, "other"),
					``,
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import baz "io"; func Baz(){ var w baz.Writer; _ = w }`,
				},
				{
					RelPath: "other/other.go",
					Src:     `package other; import other "io"; func Other(){ var w other.Writer; _ = w }`,
				},
			},
			regularOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					path.Join(projectDir, "bar/bar.go") + `:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					path.Join(projectDir, "baz/baz.go") + `:1:21: uses alias "baz" to import package "io". No consensus alias exists for this import in the project ("baz" and "other" are both used once each).`,
					path.Join(projectDir, "foo.go") + `:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar" and "foo" are both used once each).`,
					path.Join(projectDir, "other/other.go") + `:1:23: uses alias "other" to import package "io". No consensus alias exists for this import in the project ("baz" and "other" are both used once each).`,
					``,
				}
			},
			verboseOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					`"fmt" is imported using multiple different aliases:`,
					`	bar (1 file):`,
					`		` + path.Join(projectDir, "bar/bar.go") + `:1:21`,
					`	foo (1 file):`,
					`		` + path.Join(projectDir, "foo.go") + `:1:22`,
					`"io" is imported using multiple different aliases:`,
					`	baz (1 file):`,
					`		` + path.Join(projectDir, "baz/baz.go") + `:1:21`,
					`	other (1 file):`,
					`		` + path.Join(projectDir, "other/other.go") + `:1:23`,
					``,
				}
			},
		},
		{
			name: "if multiple files import the same package using a different alias but one is more common, suggest the more common one",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
					path.Join(projectDir, "baz"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import foo "fmt"; func Baz(){ foo.Println() }`,
				},
			},
			regularOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					path.Join(projectDir, "bar/bar.go") + `:1:21: uses alias "bar" to import package "fmt". Use alias "foo" instead.`,
					``,
				}
			},
			verboseOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					`"fmt" is imported using multiple different aliases:`,
					`	foo (2 files):`,
					`		` + path.Join(projectDir, "baz/baz.go") + `:1:21`,
					`		` + path.Join(projectDir, "foo.go") + `:1:22`,
					`	bar (1 file):`,
					`		` + path.Join(projectDir, "bar/bar.go") + `:1:21`,
					``,
				}
			},
		},
		{
			name: "verify correct message if there are more than 2 aliases used for an import",
			getPkgArgs: func(projectDir string) []string {
				return []string{
					projectDir,
					path.Join(projectDir, "bar"),
					path.Join(projectDir, "baz"),
				}
			},
			files: []gofiles.GoFileSpec{
				{
					RelPath: "foo.go",
					Src:     `package main; import foo "fmt"; func main(){ foo.Println() }`,
				},
				{
					RelPath: "bar/bar.go",
					Src:     `package bar; import bar "fmt"; func Bar(){ bar.Println() }`,
				},
				{
					RelPath: "baz/baz.go",
					Src:     `package baz; import baz "fmt"; func Baz(){ baz.Println() }`,
				},
			},
			regularOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					path.Join(projectDir, "bar/bar.go") + `:1:21: uses alias "bar" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
					path.Join(projectDir, "baz/baz.go") + `:1:21: uses alias "baz" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
					path.Join(projectDir, "foo.go") + `:1:22: uses alias "foo" to import package "fmt". No consensus alias exists for this import in the project ("bar", "baz" and "foo" are all used once each).`,
					``,
				}
			},
			verboseOutput: func(projectDir string, files map[string]gofiles.GoFile) []string {
				return []string{
					`"fmt" is imported using multiple different aliases:`,
					`	bar (1 file):`,
					`		` + path.Join(projectDir, "bar/bar.go") + `:1:21`,
					`	baz (1 file):`,
					`		` + path.Join(projectDir, "baz/baz.go") + `:1:21`,
					`	foo (1 file):`,
					`		` + path.Join(projectDir, "foo.go") + `:1:22`,
					``,
				}
			},
		},
	} {
		projectDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		files, err := gofiles.Write(projectDir, tc.files)
		require.NoError(t, err)

		pkgs := tc.getPkgArgs(projectDir)

		buf := bytes.Buffer{}
		doMainErr := importalias.Run(pkgs, false, &buf)
		require.Error(t, doMainErr, fmt.Sprintf("Case %d (%s)", i, tc.name))
		assert.Equal(t, tc.regularOutput(projectDir, files), strings.Split(buf.String(), "\n"), "Case %d (%s)", i, tc.name)

		buf = bytes.Buffer{}
		doMainErr = importalias.Run(pkgs, true, &buf)
		require.Error(t, doMainErr, fmt.Sprintf("Case %d (%s)", i, tc.name))
		assert.Equal(t, tc.verboseOutput(projectDir, files), strings.Split(buf.String(), "\n"), "Case %d (%s)", i, tc.name)
	}
}
