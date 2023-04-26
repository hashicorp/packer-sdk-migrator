// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package util

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/ast/astutil"
)

var printConfig = printer.Config{
	Mode:     printer.TabIndent | printer.UseSpaces,
	Tabwidth: 8,
}

func StringSliceContains(ss []string, s string) bool {
	for _, i := range ss {
		if i == s {
			return true
		}
	}
	return false
}

func ReadOneOf(dir string, filenames ...string) (fullpath string, content []byte, err error) {
	for _, filename := range filenames {
		fullpath = filepath.Join(dir, filename)
		content, err = ioutil.ReadFile(fullpath)
		if err == nil {
			break
		}
	}
	return
}

func SearchLines(lines []string, search string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.Contains(lines[i], search) {
			return i
		}
	}
	return -1
}

func SearchLinesPrefix(lines []string, search string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], search) {
			return i
		}
	}
	return -1
}

func GetpluginPath(pluginRepoName string) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		log.Printf("GOPATH is empty")
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	paths := append([]string{wd}, filepath.SplitList(gopath)...)

	for _, p := range paths {
		fullPath := filepath.Join(p, "src", pluginRepoName)
		info, err := os.Stat(fullPath)

		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", fullPath)
			} else {
				return fullPath, nil
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", fmt.Errorf("Could not find %s in working directory or GOPATH: %s", pluginRepoName, gopath)
}

func RewriteGoMod(pluginPath string, sdkVersion string, oldPackagePath string, newPackagePath string) error {
	goModPath := filepath.Join(pluginPath, "go.mod")

	input, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return err
	}

	pf, err := modfile.Parse(goModPath, input, nil)
	if err != nil {
		return err
	}

	err = pf.DropRequire(oldPackagePath)
	if err != nil {
		return err
	}

	pf.AddNewRequire(newPackagePath, sdkVersion, false)

	pf.Cleanup()
	formattedOutput, err := pf.Format()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(goModPath, formattedOutput, 0644)
	if err != nil {
		return err
	}

	return nil
}

type visitFn func(node ast.Node)

func (fn visitFn) Visit(node ast.Node) ast.Visitor {
	fn(node)
	return fn
}

func RewriteImportedPackageImports(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	addImports := map[string]string{}
	deleteImports := map[string]string{}

	for _, impSpec := range f.Imports {
		impPath, err := strconv.Unquote(impSpec.Path.Value)
		if err != nil {
			log.Print(err)
		}
		if newImpPath, ok := ONE_TO_ONE_REPLACEMENTS[impPath]; ok {
			log.Printf("Changing import of %s to %s", impPath, newImpPath)
			impSpec.Path.Value = strconv.Quote(newImpPath)
		} else if newImpPath, ok := PACKAGE_RENAME[impPath]; ok {
			// fix imports
			log.Printf("Changing import of %s to %s", impPath, newImpPath)
			impSpec.Path.Value = strconv.Quote(newImpPath)

			// fix package name in expressions that reference this package
			pathparts := strings.Split(newImpPath, "/")
			newImpName := pathparts[len(pathparts)-1]

			name := impSpec.Name
			oldNameString := ""
			if name != nil {
				oldNameString = name.String()
			}

			if oldNameString == "" {
				pathparts := strings.Split(impPath, "/")
				oldNameString = pathparts[len(pathparts)-1]
			}

			ast.Walk(visitFn(func(n ast.Node) {
				sel, ok := n.(*ast.SelectorExpr)
				if ok {
					id, ok := sel.X.(*ast.Ident)
					if ok {
						if id.Name == oldNameString {
							id.Name = newImpName
							log.Printf("renamed %s to %s", oldNameString, newImpName)
						}
					}
				}
			}), f)
		} else if _, ok := PACKAGE_SPLIT[impPath]; ok {
			log.Printf("Package %s has been refactored into multiple new SDK"+
				"packages; walking the ast to update each object as required.", impPath)
			// We store the package split map with the old import path as the
			// original key so we can find the moved items easily; now we need
			// to remap it so we can iterate over object names instead to figure
			// out where they should end up.
			remap := map[string]map[string]string{}

			for newImportPath, structList := range PACKAGE_SPLIT[impPath] {
				for _, val := range structList {
					remap[val] = map[string]string{impPath: newImportPath}
				}
			}

			// fix package name in expressions that reference this package
			name := impSpec.Name
			oldNameString := ""
			if name != nil {
				oldNameString = name.String()
			}

			if oldNameString == "" {
				pathparts := strings.Split(impPath, "/")
				oldNameString = pathparts[len(pathparts)-1]
			}

			ast.Walk(visitFn(func(n ast.Node) {
				sel, ok := n.(*ast.SelectorExpr)
				if ok {
					if id, ok := sel.X.(*ast.Ident); ok {
						if id.Name == oldNameString {
							// look up correct new import path in map, rename
							// it in this object, and add it to the imports.
							newImpPath := remap[sel.Sel.Name][impPath]
							// newImpPath := impList[impPath]
							pathparts := strings.Split(newImpPath, "/")
							newImpName := pathparts[len(pathparts)-1]
							// if we were importing with a custom name, retain
							// that customization and append new path name.
							if impSpec.Name != nil {
								newImpName = oldNameString + "_" + newImpName
							}
							id.Name = newImpName
							// Instead of copying import spec, create entirely
							// new one.
							if impSpec.Name == nil {
								addImports[newImpPath] = ""
								deleteImports[impPath] = ""
							}
							addImports[newImpPath] = newImpName
							deleteImports[impPath] = oldNameString

						}
					}
				}
			}), f)
		}
	}

	// Cannot add or delete imports to the imports list while looping over the
	// list or we'll get some gross side effeects and miss imports altogether.
	// Instead, loop over our dictionaries after the fact to add and delete
	// the necessary imports only once per import path.
	for impPath, impName := range addImports {
		astutil.AddNamedImport(fset, f, impName, impPath)
	}
	for impPath, impName := range deleteImports {
		deleted := astutil.DeleteNamedImport(fset, f, impName, impPath)
		if !deleted {
			log.Printf("issue deleting import %s; may need to manually delete", impPath)
		}

	}

	// overwrite imports
	ast.SortImports(fset, f)
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	if err := printConfig.Fprint(w, fset, f); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func GoModTidy(pluginPath string) error {
	args := []string{"go", "mod", "tidy"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Dir = pluginPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("[DEBUG] Executing command %q", args)
	err := cmd.Run()
	if err != nil {
		return NewExecError(err, stderr.String())
	}

	return nil
}

type ExecError struct {
	Err    error
	Stderr string
}

func (ee *ExecError) Error() string {
	return fmt.Sprintf("%s\n%s", ee.Err, ee.Stderr)
}

func NewExecError(err error, stderr string) *ExecError {
	return &ExecError{err, stderr}
}
