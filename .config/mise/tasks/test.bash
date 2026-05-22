#!/usr/bin/env bash
#MISE description="run Go tests"
set -o errexit -o nounset

cd "$(git rev-parse --show-toplevel)"
go test ./...
