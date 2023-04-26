// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func testFixture(n ...string) string {
	paths := []string{"test-fixtures"}
	paths = append(paths, n...)
	return filepath.Join(paths...)
}

func CopyInputFile(t *testing.T, inputPath, outputPath string) {
	input, err := ioutil.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("Error reading file to modify")
	}

	err = ioutil.WriteFile(outputPath, input, 0644)
	if err != nil {
		t.Fatalf("Error copying file to modify")
	}
}

func Test_sdk_migrate(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	_ = cwd

	tc := []struct {
		folder string
	}{
		{"sdk_migrate_basic"},
	}

	for _, tc := range tc {
		t.Run(tc.folder, func(t *testing.T) {
			inputPath := filepath.Join(testFixture(tc.folder, "input.go.txt"))
			outputPath := filepath.Join(testFixture(tc.folder, "actual.go.txt"))
			CopyInputFile(t, inputPath, outputPath)

			expectedPath := filepath.Join(testFixture(tc.folder, "expected.go.txt"))
			RewriteImportedPackageImports(outputPath)

			expected := mustBytes(ioutil.ReadFile(expectedPath))
			actual := mustBytes(ioutil.ReadFile(outputPath))

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Fatalf("unexpected output: %s", diff)
			}
			os.Remove(outputPath)
		})
	}
}

func mustBytes(b []byte, e error) []byte {
	if e != nil {
		panic(e)
	}
	return b
}
