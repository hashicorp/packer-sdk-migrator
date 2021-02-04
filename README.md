# packer-sdk-migrator

The Packer plugin plugin SDK, previously part of the [github.com/hashicorp/packer](https://github.com/hashicorp/packer) "Core" Go module, has been moved to a new Go module, [github.com/hashicorp/packer-plugin-sdk](https://github.com/hashicorp/packer-plugin-sdk). Packer plugins should now import `hashicorp/packer-plugin-sdk`.

`packer-sdk-migrator` is a CLI tool which will migrate a Packer plugin to the new SDK module by rewriting import paths. `packer-sdk-migrator check` checks the eligibility of the plugin for migration.

## Installation

```sh
go install github.com/hashicorp/packer-sdk-migrator
$GOBIN/packer-sdk-migrator
```

## `packer-sdk-migrator check`: check eligibility for migration

Checks whether a Packer plugin is ready to migrate to the newly extracted Packer SDK package.

```sh
packer-sdk-migrator check [--help] [--csv] PATH
```

Outputs a report containing:
 - Go version used in plugin (soft requirement)
 - Whether the plugin uses Go modules
 - Version of `hashicorp/packer` used
 - Whether the plugin uses any `hashicorp/packer` packages that are not in `hashicorp/packer-plugin-sdk`

The `--csv` flag will output values in CSV format.

Exits 0 if the plugin meets all the hard requirements, 1 otherwise.

The Go version requirement is a "soft" requirement: it is strongly recommended to upgrade to Go version 1.12+ before migrating to the new SDK, but the migration can still be performed if this requirement is not met.

## `packer-sdk-migrator migrate`: migrate to standalone SDK

Migrates the Packer plugin to the new extracted SDK (`github.com/hashicorp/packer-plugin-sdk`), replacing references to the old SDK (`github.com/hashicorp/packer`).

**Note: No backup is made before modifying files. Please make sure your VCS staging area is clean.**

```sh
packer-sdk-migrator migrate [--help] PATH
```

The eligibility check will be run first: migration will not proceed if this check fails.

The migration tool will then make the following changes:
 - `go.mod`: replace `github.com/hashicorp/packer` dependency with `github.com/hashicorp/packer-plugin-sdk`
 - rewrite import paths in all plugin `.go` files (except in `vendor/`) accordingly
 - run `go mod tidy`

If you use vendored Go dependencies, you should run `go mod vendor` afterwards.
