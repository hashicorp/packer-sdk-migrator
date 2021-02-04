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

	revisedImports := []*ast.ImportSpec{}
	for _, impSpec := range f.Imports {
		impPath, err := strconv.Unquote(impSpec.Path.Value)
		if err != nil {
			log.Print(err)
		}

		if ONE_TO_ONE_REPLACEMENTS[impPath] != "" {
			newImpPath := ONE_TO_ONE_REPLACEMENTS[impPath]

			// copy impspec into revised imports, replacing the import path.
			newImpSpec := &ast.ImportSpec{}
			*newImpSpec = *impSpec

			newImpSpec.Path.Value = strconv.Quote(newImpPath)
			revisedImports = append(revisedImports, newImpSpec)
		} else if PACKAGE_RENAME[impPath] != "" {
			// fix imports
			newImpPath := PACKAGE_RENAME[impPath]

			// copy impspec into revised imports, replacing the import path.
			newImpSpec := &ast.ImportSpec{}
			*newImpSpec = *impSpec

			newImpSpec.Path.Value = strconv.Quote(newImpPath)
			revisedImports = append(revisedImports, newImpSpec)

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
							id.Name = newImpName
							// add import path into revisedimports
							newImpSpec := &ast.ImportSpec{}
							*newImpSpec = *impSpec

							newImpSpec.Path.Value = strconv.Quote(newImpPath)
							revisedImports = append(revisedImports, newImpSpec)

							log.Printf("renamed %s to %s", oldNameString, newImpName)

						}
					}
				}
			}), f)
		} else {
			revisedImports = append(revisedImports, impSpec)
		}
	}

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
