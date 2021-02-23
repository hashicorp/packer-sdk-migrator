package check

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/packer-sdk-migrator/util"
	"github.com/mitchellh/cli"
)

const (
	CommandName = "check"

	goVersionConstraint = ">=1.12"

	packerModPath           = "github.com/hashicorp/packer"
	packerVersionConstraint = ">=1.5.0"

	sdkModPath           = "github.com/hashicorp/packer-plugin-sdk"
	sdkVersionConstraint = ">=0.0.11"
)

type AlreadyMigrated struct {
	sdkVersion string
}

func (am *AlreadyMigrated) Error() string {
	return fmt.Sprintf("plugin already migrated to SDK version %s", am.sdkVersion)
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
	return `Usage: packer-sdk-migrator check [--help] [PATH]

  Checks whether the Packer plugin at PATH is ready to be migrated to the
  new Packer plugin SDK (v0.1).

  PATH is resolved relative to $GOPATH/src/PATH. If it is not supplied,
  it is assumed that the current working directory contains a Packer plugin.

  By default, outputs a human-readable report and exits 0 if the plugin is
  ready for migration, 1 otherwise.

Example:
  packer-sdk-migrator check github.com/my-packer-plugin/packer-builder-local
`
}

func (c *command) Synopsis() string {
	return "Checks whether a Packer plugin is ready to be migrated to the new SDK (v0.0.11)."
}

func (c *command) Run(args []string) int {
	flags := flag.NewFlagSet(CommandName, flag.ExitOnError)
	var csv bool
	flags.BoolVar(&csv, "csv", false, "CSV output")
	flags.Parse(args)

	var pluginRepoName string
	var pluginPath string
	if flags.NArg() == 1 {
		var err error
		pluginRepoName := flags.Args()[0]
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

	err := runCheck(c.ui, pluginPath, pluginRepoName, csv)
	if err != nil {
		msg, alreadyMigrated := err.(*AlreadyMigrated)
		if alreadyMigrated {
			c.ui.Info(msg.Error())
			return 0
		}

		if !csv {
			c.ui.Error(err.Error())
		}
		return 1
	}

	return 0
}

func RunCheck(ui cli.Ui, pluginPath, repoName string) error {
	return runCheck(ui, pluginPath, repoName, false)
}

func runCheck(ui cli.Ui, pluginPath, repoName string, csv bool) error {
	if !csv {
		ui.Output("Checking Go runtime version ...")
	}
	goVersion, goVersionSatisfied := CheckGoVersion(pluginPath)
	if !csv {
		if goVersionSatisfied {
			ui.Info(fmt.Sprintf("Go version %s: OK.", goVersion))
		} else {
			ui.Warn(fmt.Sprintf("Go version does not satisfy constraint %s. Found Go version: %s.", goVersionConstraint, goVersion))
		}
	}

	if !csv {
		ui.Output("Checking whether plugin uses Go modules...")
	}
	goModulesUsed := CheckForGoModules(pluginPath)
	if !csv {
		if goModulesUsed {
			ui.Info("Go modules in use: OK.")
		} else {
			ui.Warn("Go modules not in use. plugin must use Go modules.")
		}
	}

	if !csv {
		ui.Output(fmt.Sprintf("Checking version of %s to determine if plugin was already migrated...", sdkModPath))
	}
	sdkVersion, sdkVersionSatisfied, err := CheckDependencyVersion(pluginPath, sdkModPath, sdkVersionConstraint)
	if err != nil {
		return fmt.Errorf("Error getting SDK version for plugin %s: %s", pluginPath, err)
	}
	if !csv {
		if sdkVersionSatisfied {
			return &AlreadyMigrated{sdkVersion}
		} else if sdkVersion != "" {
			return fmt.Errorf("plugin already migrated, but SDK version %s does not satisfy constraint %s.",
				sdkVersion, sdkVersionConstraint)
		}
	}

	if !csv {
		ui.Output(fmt.Sprintf("Checking version of %s used in plugin...", packerModPath))
	}
	packerVersion, packerVersionSatisfied, err := CheckDependencyVersion(pluginPath, packerModPath, packerVersionConstraint)
	if err != nil {
		return fmt.Errorf("Error getting Packer version for plugin %s: %s", pluginPath, err)
	}
	if !csv {
		if packerVersionSatisfied {
			ui.Info(fmt.Sprintf("Packer version %s: OK.", packerVersion))
		} else if packerVersion != "" {
			ui.Warn(fmt.Sprintf("Packer version does not satisfy constraint %s. Found Packer version: %s", packerVersionConstraint, packerVersion))
		} else {
			return fmt.Errorf("This directory (%s) doesn't seem to be a Packer plugin.\nplugins depend on %s", pluginPath, packerModPath)
		}
	}

	if !csv {
		ui.Output("Checking whether plugin uses deprecated SDK packages or identifiers...")
	}
	removedPackagesInUse, removedIdentsInUse, err := CheckSDKPackageImportsAndRefs(pluginPath)
	if err != nil {
		return err
	}
	usesRemovedPackagesOrIdents := len(removedPackagesInUse) > 0 || len(removedIdentsInUse) > 0
	if !csv {
		if err != nil {
			return fmt.Errorf("Error determining use of deprecated SDK packages and identifiers: %s", err)
		}
		if !usesRemovedPackagesOrIdents {
			ui.Info("No imports of deprecated SDK packages or identifiers: OK.")
		}
		formatRemovedPackages(ui, removedPackagesInUse)
		formatRemovedIdents(ui, removedIdentsInUse)
	}
	constraintsSatisfied := goVersionSatisfied && goModulesUsed && packerVersionSatisfied && !usesRemovedPackagesOrIdents
	if csv {
		ui.Output(fmt.Sprintf("go_version,go_version_satisfies_constraint,uses_go_modules,sdk_version,sdk_version_satisfies_constraint,does_not_use_removed_packages,all_constraints_satisfied\n%s,%t,%t,%s,%t,%t,%t",
			goVersion, goVersionSatisfied, goModulesUsed, packerVersion, packerVersionSatisfied, !usesRemovedPackagesOrIdents, constraintsSatisfied))
	} else {
		var prettypluginName string
		if repoName != "" {
			prettypluginName = " " + repoName
		}
		if constraintsSatisfied {
			ui.Info(fmt.Sprintf("\nAll constraints satisfied. plugin%s can be migrated to the new SDK.\n", prettypluginName))
			return nil
		} else if goModulesUsed && packerVersionSatisfied && !usesRemovedPackagesOrIdents {
			ui.Info(fmt.Sprintf("\nplugin%s can be migrated to the new SDK, but Go version %s is recommended.\n", prettypluginName, goVersionConstraint))
			return nil
		}
	}

	return fmt.Errorf("\nSome constraints not satisfied. Please resolve these before migrating to the new SDK.")
}

func formatRemovedPackages(ui cli.Ui, removedPackagesInUse []string) {
	if len(removedPackagesInUse) == 0 {
		return
	}

	ui.Warn("Deprecated SDK packages in use:")
	for _, pkg := range removedPackagesInUse {
		ui.Warn(fmt.Sprintf(" * %s", pkg))
	}
}

func formatRemovedIdents(ui cli.Ui, removedIdentsInUse []*Offence) {
	if len(removedIdentsInUse) == 0 {
		return
	}
	ui.Warn("Deprecated SDK identifiers in use:")
	for _, ident := range removedIdentsInUse {
		d := ident.IdentDeprecation
		ui.Warn(fmt.Sprintf(" * %s (%s)", d.Identifier.Name, d.ImportPath))

		for _, pos := range ident.Positions {
			ui.Warn(fmt.Sprintf("   * %s", pos))
		}
	}
}

func CheckGoVersion(pluginPath string) (goVersion string, satisfiesConstraint bool) {
	c, err := version.NewConstraint(goVersionConstraint)

	runtimeVersion := strings.TrimLeft(runtime.Version(), "go")
	v, err := version.NewVersion(runtimeVersion)
	if err != nil {
		log.Printf("[WARN] Could not parse Go version %s", runtimeVersion)
		return "", false
	}

	return runtimeVersion, c.Check(v)
}

func CheckForGoModules(pluginPath string) (usingModules bool) {
	if _, err := os.Stat(filepath.Join(pluginPath, "go.mod")); err != nil {
		log.Printf("[WARN] 'go.mod' file not found - plugin %s is not using Go modules", pluginPath)
		return false
	}
	return true
}

func CheckSDKPackageImportsAndRefs(pluginPath string) (removedPackagesInUse []string, packageRefsOffences []*Offence, err error) {
	var pluginImportDetails *pluginImportDetails

	pluginImportDetails, err = GoListPackageImports(pluginPath)
	if err != nil {
		return nil, nil, err
	}

	removedPackagesInUse, err = CheckSDKPackageImports(pluginImportDetails)
	if err != nil {
		return nil, nil, err
	}

	packageRefsOffences, err = CheckSDKPackageRefs(pluginImportDetails)
	if err != nil {
		return nil, nil, err
	}

	return
}
