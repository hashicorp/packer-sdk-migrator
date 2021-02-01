package check

import (
	"strings"
)

var sdkPackages = map[string]bool{

	"github.com/hashicorp/packer/template/config":            true,
	"github.com/hashicorp/packer/packer":                     true,
	"github.com/hashicorp/packer/template/interpolate":       true,
	"github.com/hashicorp/packer/helper/builder/testing":     true,
	"github.com/hashicorp/packer/helper/tests":               true,
	"github.com/hashicorp/packer/common/adapter":             true,
	"github.com/hashicorp/packer/common/bootcommand":         true,
	"github.com/hashicorp/packer/common/chroot":              true,
	"github.com/hashicorp/packer/common":                     true,
	"github.com/hashicorp/packer/helper/communicator":        true,
	"github.com/hashicorp/packer/helper/config":              true,
	"github.com/hashicorp/packer/template/interpolate":       true,
	"github.com/hashicorp/packer/communicator/ssh":           true,
	"github.com/hashicorp/packer/helper/communicator/sshkey": true,
	"github.com/hashicorp/packer/provisioner":                true,
	"github.com/hashicorp/packer/common/json":                true,
	"github.com/hashicorp/packer/helper/multistep":           true,
	"github.com/hashicorp/packer/common/net":                 true,
	"github.com/hashicorp/packer/packer":                     true,
	"github.com/hashicorp/packer/builder":                    true,
	"github.com/hashicorp/packer/packer":                     true,
	"github.com/hashicorp/packer/packer/plugin":              true,
	"github.com/hashicorp/packer/common/random":              true,
	"github.com/hashicorp/packer/common":                     true,
	"github.com/hashicorp/packer/packer/rpc":                 true,
	"github.com/hashicorp/packer/communicator":               true,
	"github.com/hashicorp/packer/common/shell":               true,
	"github.com/hashicorp/packer/common/shell-local":         true,
	"github.com/hashicorp/packer/helper/builder/localexec":   true,
	"github.com/hashicorp/packer/common/shutdowncommand":     true,
	"github.com/hashicorp/packer/template":                   true,
	"github.com/hashicorp/packer/packer/tmp":                 true,
	"github.com/hashicorp/packer/helper/useragent":           true,
	"github.com/hashicorp/packer/common/uuid":                true,
}

func CheckSDKPackageImports(details *pluginImportDetails) ([]string, error) {
	removedPackagesInUse := []string{}

	for importPath := range details.AllImportPathsHash {
		if !strings.HasPrefix(importPath, "github.com/hashicorp/packer/") {
			continue
		}

		if isSDK := sdkPackages[importPath]; !isSDK {
			removedPackagesInUse = append(removedPackagesInUse, importPath)
		}
	}

	return removedPackagesInUse, nil
}
