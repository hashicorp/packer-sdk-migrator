package migrate

import (
	"flag"
	"fmt"
	"go/printer"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-sdk-migrator/cmd/check"
	"github.com/hashicorp/packer-sdk-migrator/util"
	"github.com/mitchellh/cli"
)

const (
	CommandName    = "migrate"
	oldPackagePath = "github.com/hashicorp/packer"
	newPackagePath = "github.com/hashicorp/packer-plugin-sdk"
	defaultVersion = "v1.7.0"
)

var printConfig = printer.Config{
	Mode:     printer.TabIndent | printer.UseSpaces,
	Tabwidth: 8,
}

type command struct {
	ui cli.Ui
}

func CommandFactory(ui cli.Ui) func() (cli.Command, error) {
	return func() (cli.Command, error) {
		return &command{ui}, nil
	}
}

func (c *command) Help() string {
	return `Usage: packer-sdk-migrator migrate [--help] [--sdk-version SDK_VERSION] [--force] [IMPORT_PATH]

  Migrates the Packer plugin at PATH to the new Packer plugin
  SDK, defaulting to the git reference ` + defaultVersion + `.

  IMPORT_PATH is resolved relative to $GOPATH/src/IMPORT_PATH. If it is not supplied,
  it is assumed that the current working directory contains a Packer plugin.

  Optionally, an SDK_VERSION can be passed, which is parsed as a Go module
  release version. For example: v0.1.1, latest, master.

  Rewrites import paths and go.mod. No backup is made before files are
  overwritten.

Example:
  packer-sdk-migrator migrate --sdk-version master github.com/my-packer-plugin/packer-builder-local`
}

func (c *command) Synopsis() string {
	return "Migrates a Packer plugin to the new SDK (v0.1)."
}

func (c *command) Run(args []string) int {
	flags := flag.NewFlagSet(CommandName, flag.ExitOnError)
	var sdkVersion string
	flags.StringVar(&sdkVersion, "sdk-version", defaultVersion, "SDK version")
	var forceMigration bool
	flags.BoolVar(&forceMigration, "force", false, "Whether to ignore failing checks and force migration")
	flags.Parse(args)

	var pluginRepoName string
	var pluginPath string
	if flags.NArg() == 1 {
		var err error
		pluginRepoName = flags.Args()[0]
		pluginPath, err = util.GetpluginPath(pluginRepoName)
		if err != nil {
			c.ui.Error(fmt.Sprintf("Error finding plugin %s: %s", pluginRepoName, err))
			return 1
		}
	} else if flags.NArg() == 0 {
		var err error
		pluginPath, err = os.Getwd()
		if err != nil {
			c.ui.Error(fmt.Sprintf("Error finding current working directory: %s", err))
			return 1
		}
	} else {
		return cli.RunResultHelp
	}

	err := check.RunCheck(c.ui, pluginPath, pluginRepoName)
	if err != nil {
		c.ui.Warn(err.Error())
		if forceMigration {
			c.ui.Warn("Ignoring failed eligibility checks")
		} else {
			c.ui.Error("plugin failed eligibility check for migration to the new SDK. Please see messages above.")
			return 1
		}
	}

	c.ui.Output("Rewriting plugin go.mod file...")
	err = util.RewriteGoMod(pluginPath, sdkVersion, oldPackagePath, newPackagePath)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error rewriting go.mod file: %s", err))
		return 1
	}

	c.ui.Output("Rewriting SDK package imports...")
	err = filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			err := util.RewriteImportedPackageImports(path, oldPackagePath, newPackagePath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error rewriting SDK imports: %s", err))
		return 1
	}

	c.ui.Output("Running `go mod tidy`...")
	err = util.GoModTidy(pluginPath)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error running go mod tidy: %s", err))
		return 1
	}

	var prettypluginName string
	if pluginRepoName != "" {
		prettypluginName = " " + pluginRepoName
	}
	c.ui.Info(fmt.Sprintf("Success! plugin%s is migrated to %s %s.",
		prettypluginName, newPackagePath, sdkVersion))

	hasVendor, err := HasVendorFolder(pluginPath)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Failed to check vendor folder: %s", err))
		return 1
	}

	if hasVendor {
		c.ui.Info("\nIt looks like this plugin vendors dependencies. " +
			"Don't forget to run `go mod vendor`.")
	}

	c.ui.Info(fmt.Sprintf("Make sure to review all changes and run all tests."))
	return 0
}

func HasVendorFolder(pluginPath string) (bool, error) {
	vendorPath := filepath.Join(pluginPath, "vendor")
	fs, err := os.Stat(vendorPath)
	if err != nil {
		return false, err
	}
	if !fs.Mode().IsDir() {
		return false, fmt.Errorf("%s is not folder (expected folder)", vendorPath)
	}

	return true, nil
}
