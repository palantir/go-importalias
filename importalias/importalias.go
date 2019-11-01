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

package importalias

import (
	"fmt"
	"go/build"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

func Run(pkgPaths []string, verbose bool, w io.Writer) error {
	projectImportInfo := NewProjectImportInfo()
	for _, pkgPath := range pkgPaths {
		loadedPkg, _ := build.ImportDir(pkgPath, 0)

		var goFilesForPkg []string
		goFilesForPkg = append(goFilesForPkg, loadedPkg.GoFiles...)
		goFilesForPkg = append(goFilesForPkg, loadedPkg.TestGoFiles...)
		goFilesForPkg = append(goFilesForPkg, loadedPkg.XTestGoFiles...)
		sort.Strings(goFilesForPkg)

		for _, currGoFileName := range goFilesForPkg {
			currFile := path.Join(pkgPath, currGoFileName)
			if err := projectImportInfo.AddImportAliasesFromFile(currFile); err != nil {
				return errors.Wrapf(err, "failed to determine imports in file %s", currFile)
			}
		}
	}

	importsToAliases := projectImportInfo.ImportsToAliases()
	var pkgsWithMultipleAliases []string
	pkgsWithMultipleAliasesMap := make(map[string]struct{})
	for k, v := range importsToAliases {
		if len(v) > 1 {
			// package is imported using more than 1 alias
			pkgsWithMultipleAliases = append(pkgsWithMultipleAliases, k)
			pkgsWithMultipleAliasesMap[k] = struct{}{}
		}
	}
	sort.Strings(pkgsWithMultipleAliases)
	if len(pkgsWithMultipleAliases) == 0 {
		return nil
	}

	if verbose {
		for _, k := range pkgsWithMultipleAliases {
			_, _ = fmt.Fprintf(w, "%s is imported using multiple different aliases:\n", k)
			for _, currAliasInfo := range importsToAliases[k] {
				var files []string
				for k, v := range currAliasInfo.Occurrences {
					files = append(files, fmt.Sprintf("%s:%d:%d", k, v.Line, v.Column))
				}
				sort.Strings(files)

				var numFilesMsg string
				if len(currAliasInfo.Occurrences) == 1 {
					numFilesMsg = "(1 file)"
				} else {
					numFilesMsg = fmt.Sprintf("(%d files)", len(currAliasInfo.Occurrences))
				}
				_, _ = fmt.Fprintf(w, "\t%s %s:\n\t\t%s\n", currAliasInfo.Alias, numFilesMsg, strings.Join(files, "\n\t\t"))
			}
		}
	} else {
		filesToAliases := projectImportInfo.FilesToImportAliases()
		var files []string
		for file := range filesToAliases {
			files = append(files, file)
		}
		sort.Strings(files)

		for _, file := range files {
			for _, alias := range filesToAliases[file] {
				if _, ok := pkgsWithMultipleAliasesMap[alias.ImportPath]; !ok {
					continue
				}
				status := projectImportInfo.GetAliasStatus(alias.Alias, alias.ImportPath)
				if status.OK {
					continue
				}
				_, _ = fmt.Fprintf(w, "%s:%d:%d: uses alias %q to import package %s. %s.\n", file, alias.Pos.Line, alias.Pos.Column, alias.Alias, alias.ImportPath, status.Recommendation)
			}
		}
	}
	return fmt.Errorf("")
}
