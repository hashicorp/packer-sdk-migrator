package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"

	"github.com/hashicorp/packer-sdk-migrator/util"
	goList "github.com/kmoe/go-list"
	refsParser "github.com/radeksimko/go-refs/parser"
)

type Offence struct {
	IdentDeprecation *identDeprecation
	Positions        []*token.Position
}

type identDeprecation struct {
	ImportPath string
	Identifier *ast.Ident
	Message    string
}

var deprecations = []*identDeprecation{
	// {
	// 	"github.com/hashicorp/terraform/httpclient",
	// 	ast.NewIdent("UserAgentString"),
	// 	"This function has been removed, please use httpclient.TerraformUserAgent(version) instead",
	// },
}

// pluginImports is a data structure we parse the `go list` output into
// for efficient searching
type pluginImportDetails struct {
	AllImportPathsHash map[string]bool
	Packages           map[string]pluginPackage
}

type pluginPackage struct {
	Dir         string
	ImportPath  string
	GoFiles     []string
	TestGoFiles []string
	Imports     []string
	TestImports []string
}

func GoListPackageImports(pluginPath string) (*pluginImportDetails, error) {
	packages, err := goList.GoList(pluginPath, "./...", "-mod=vendor")
	if err != nil {
		return nil, err
	}

	allImportPathsHash := make(map[string]bool)
	pluginPackages := make(map[string]pluginPackage)

	for _, p := range packages {
		for _, i := range p.Imports {
			allImportPathsHash[i] = true
		}

		pluginPackages[p.ImportPath] = pluginPackage{
			Dir:         p.Dir,
			ImportPath:  p.ImportPath,
			GoFiles:     p.GoFiles,
			TestGoFiles: p.TestGoFiles,
			Imports:     p.Imports,
			TestImports: p.TestImports,
		}
	}

	return &pluginImportDetails{
		AllImportPathsHash: allImportPathsHash,
		Packages:           pluginPackages,
	}, nil
}

func CheckSDKPackageRefs(pluginImportDetails *pluginImportDetails) ([]*Offence, error) {
	offences := make([]*Offence, 0, 0)

	for _, d := range deprecations {
		fset := token.NewFileSet()
		files, err := filesWhichImport(pluginImportDetails, d.ImportPath)
		if err != nil {
			return nil, err
		}

		foundPositions := make([]*token.Position, 0, 0)

		for _, filePath := range files {
			f, err := parser.ParseFile(fset, filePath, nil, 0)
			if err != nil {
				return nil, err
			}

			identifiers, err := refsParser.FindPackageReferences(f, d.ImportPath)
			if err != nil {
				// package not imported in this file
				continue
			}

			positions, err := findIdentifierPositions(fset, identifiers, d.Identifier)
			if err != nil {
				return nil, err
			}

			if len(positions) > 0 {
				foundPositions = append(foundPositions, positions...)
			}
		}

		if len(foundPositions) > 0 {
			offences = append(offences, &Offence{
				IdentDeprecation: d,
				Positions:        foundPositions,
			})
		}
	}

	return offences, nil
}

func findIdentifierPositions(fset *token.FileSet, nodes []ast.Node, ident *ast.Ident) ([]*token.Position, error) {
	positions := make([]*token.Position, 0, 0)

	for _, node := range nodes {
		nodeName := fmt.Sprint(node)
		if nodeName == ident.String() {
			position := fset.Position(node.Pos())
			positions = append(positions, &position)
		}
	}

	return positions, nil
}

func filesWhichImport(pluginImportDetails *pluginImportDetails, importPath string) (files []string, e error) {
	files = []string{}
	for _, p := range pluginImportDetails.Packages {
		if util.StringSliceContains(p.Imports, importPath) {
			files = append(files, prependDirToFilePaths(p.GoFiles, p.Dir)...)
		}
		if util.StringSliceContains(p.TestImports, importPath) {
			files = append(files, prependDirToFilePaths(p.TestGoFiles, p.Dir)...)
		}
	}

	return files, nil
}

func prependDirToFilePaths(filePaths []string, dir string) []string {
	newFilePaths := []string{}
	for _, f := range filePaths {
		newFilePaths = append(newFilePaths, path.Join(dir, f))
	}
	return newFilePaths
}
