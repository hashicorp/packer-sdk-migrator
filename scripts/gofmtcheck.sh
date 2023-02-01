#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e

echo "==> Checking that code complies with gofmt requirements..."
gofmt_files=$(gofmt -l `find . -name '*.go' | grep -v vendor`)
if [[ -n ${gofmt_files} ]]; then
    echo 'gofmt needs running on the following files:'
    echo "${gofmt_files}"
    echo "You can use the command: \`gofmt -w\` to reformat code."
    exit 1
fi

exit 0
