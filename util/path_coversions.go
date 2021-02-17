package util

// The easiest replacements are those where we simply moved a package from the
// Packer core without changing the package name. All we have to do for these
// packages is change the imports and they'll work automagically.
var ONE_TO_ONE_REPLACEMENTS = map[string]string{
	"github.com/hashicorp/packer/common/adapter":                          "github.com/hashicorp/packer-plugin-sdk/adapter",
	"github.com/hashicorp/packer/common/bootcommand":                      "github.com/hashicorp/packer-plugin-sdk/bootcommand",
	"github.com/hashicorp/packer/common/chroot":                           "github.com/hashicorp/packer-plugin-sdk/chroot",
	"github.com/hashicorp/packer/helper/communicator":                     "github.com/hashicorp/packer-plugin-sdk/communicator",
	"github.com/hashicorp/packer/helper/config":                           "github.com/hashicorp/packer-plugin-sdk/template/config",
	"github.com/hashicorp/packer/template/interpolate":                    "github.com/hashicorp/packer-plugin-sdk/template/interpolate",
	"github.com/hashicorp/packer/helper/multistep":                        "github.com/hashicorp/packer-plugin-sdk/multistep",
	"github.com/hashicorp/packer/common/net":                              "github.com/hashicorp/packer-plugin-sdk/net",
	"github.com/hashicorp/packer/packer":                                  "github.com/hashicorp/packer-plugin-sdk/packer",
	"github.com/hashicorp/packer/packer/plugin":                           "github.com/hashicorp/packer-plugin-sdk/plugin",
	"github.com/hashicorp/packer/common/retry":                            "github.com/hashicorp/packer-plugin-sdk/retry",
	"github.com/hashicorp/packer/packer/rpc":                              "github.com/hashicorp/packer-plugin-sdk/rpc",
	"github.com/hashicorp/packer/template/interpolate/aws/secretsmanager": "github.com/hashicorp/packer-plugin-sdk/template/interpolate/aws/secretsmanager",
	"github.com/hashicorp/packer/common/shell":                            "github.com/hashicorp/packer-plugin-sdk/shell",
	"github.com/hashicorp/packer/common/shell-local":                      "github.com/hashicorp/packer-plugin-sdk/shell-local/config.go",
	"github.com/hashicorp/packer/common/shutdowncommand":                  "github.com/hashicorp/packer-plugin-sdk/shutdowncommand",
	"github.com/hashicorp/packer/helper/ssh":                              "github.com/hashicorp/packer-plugin-sdk/communicator/ssh",
	"github.com/hashicorp/packer/helper/communicator/sshkey":              "github.com/hashicorp/packer-plugin-sdk/communicator/sshkey",
	"github.com/hashicorp/packer/template":                                "github.com/hashicorp/packer-plugin-sdk/template",
	"github.com/hashicorp/packer/communicator/winrm":                      "github.com/hashicorp/packer-plugin-sdk/sdk-internals/communicator/winrm",
	"github.com/hashicorp/packer/common/uuid":                             "github.com/hashicorp/packer-plugin-sdk/uuid",
}

// Some packages were moved all as one, but the module name was changed so we
// need to not only change the import path but also how the module is referenced
// in expressions within the file. This requires an ast.Walk, but isn't too
// complex outside that.
var PACKAGE_RENAME = map[string]string{
	"github.com/hashicorp/packer/provisioner":            "github.com/hashicorp/packer-plugin-sdk/guestexec",
	"github.com/hashicorp/packer/builder":                "github.com/hashicorp/packer-plugin-sdk/packerbuilderdata",
	"github.com/hashicorp/packer/helper/builder/testing": "github.com/hashicorp/packer-plugin-sdk/acctest",
}

// For packages that got split up into multiple destination packages, we have
// to hardcode which exported identities will end up where, in order to
// properly fix the file. This requires an ast.Walk() where the visit function
// checks this map to see where a value should end up.
var PACKAGE_SPLIT = map[string]map[string][]string{
	"github.com/hashicorp/packer/common": {
		"github.com/hashicorp/packer-plugin-sdk/common":                []string{"BuildNameConfigKey", "BuilderTypeConfigKey", "CoreVersionConfigKey", "DebugConfigKey", "ForceConfigKey", "OnErrorConfigKey", "TemplatePathKey", "UserVariablesConfigKey", "PackerConfig", "CommandWrapper", "ShellCommand"},
		"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps": []string{"CDConfig", "FloppyConfig", "HTTPConfig", "ISOConfig", "StepCleanupTempKeys", "StepCreateCD", "StepCreateFloppy", "StepDownload", "StepHTTPServer", "StepOutputDir", "StepProvision", "NewGuestCommands", "MultistepDebugFn", "NewRunner", "NewRunnerWithPauseFn", "PopulateProvisionHookData", "MultistepDebugFn", "NewRunner", "NewRunnerWithPauseFn", "PopulateProvisionHookData"},
	},
	"github.com/hashicorp/packer/hcl2template": {
		"github.com/hashicorp/packer-plugin-sdk/hcl2helper":      []string{"NestedMockConfig", "MockTag", "MockConfig", "NamedMapStringString", "NamedString", "FlatMockConfig", "FlatMockTag", "FlatNestedMockConfig", "HCL2ValueFromConfigValue"},
		"github.com/hashicorp/packer-plugin-sdk/template/config": []string{"KeyValue", "KeyValues", "KeyValueFilter", "NameValue", "NameValues", "NameValueFilter", "FlatKeyValue", "FlatKeyValueFilter", "FlatNameValue", "FlatNameValueFilter"},
	},
}
