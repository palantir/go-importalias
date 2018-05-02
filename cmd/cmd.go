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

package cmd

import (
	"github.com/palantir/pkg/cobracli"
	"github.com/spf13/cobra"

	"github.com/palantir/go-importalias/importalias"
)

var (
	rootCmd = &cobra.Command{
		Use:   "importalias [flags] [packages]",
		Short: "verifies that import aliases are consistent across files and packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return importalias.Run(args, verboseFlagVal, cmd.OutOrStdout())
		},
	}

	verboseFlagVal bool
)

func Execute() int {
	return cobracli.ExecuteWithDefaultParams(rootCmd)
}

func init() {
	rootCmd.Flags().BoolVarP(&verboseFlagVal, "verbose", "v", false, "print verbose analysis of all imports that have multiple aliases")
}
