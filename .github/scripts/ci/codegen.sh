#!/usr/bin/env bash
set -euo pipefail

dotnet tool install -g NSwag.ConsoleCore

pip install pyyaml \
  "overload-party-doc-tools @ git+https://github.com/kenyamaneko/overload-party-common.git@main#subdirectory=packages/doc-tools"

COMMON_REPO="$(.github/scripts/ci/checkout-common.sh)"
export COMMON_REPO

make generate
