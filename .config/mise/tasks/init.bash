#!/usr/bin/env bash
#MISE description="initialize the local development environment"
set -o pipefail -o errexit -o nounset

mise install
hk install --mise
mise run init-ci
