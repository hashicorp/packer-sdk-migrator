package check

import (
	"strings"
)

var sdkPackages = map[string]bool{}

func CheckSDKPackageImports(details *pluginImportDetails) ([]string, error) {
	removedPackagesInUse := []string{}

	for importPath := range details.AllImportPathsHash {
		if !strings.HasPrefix(importPath, "github.com/hashicorp/packer/") {
			continue
		}

		// if isSDK := sdkPackages[importPath]; !isSDK {
		// 	removedPackagesInUse = append(removedPackagesInUse, importPath)
		// }
	}

	return removedPackagesInUse, nil
}
