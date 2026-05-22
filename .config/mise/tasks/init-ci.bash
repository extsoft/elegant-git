#!/usr/bin/env bash
#MISE description="initialize the CI environment"
set -o pipefail -o errexit -o nounset

cd "$(git rev-parse --show-toplevel)"
go mod download
