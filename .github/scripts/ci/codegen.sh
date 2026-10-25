#!/usr/bin/env bash
set -euo pipefail

dotnet tool install -g NSwag.ConsoleCore

pip install pyyaml \
  "overload-party-doc-tools @ git+https://github.com/kenyamaneko/overload-party-common.git@main#subdirectory=packages/doc-tools"

COMMON_REPO="$(.github/scripts/ci/checkout-repo.sh overload-party-common)"
export COMMON_REPO
BATTLE_REPO="$(.github/scripts/ci/checkout-repo.sh overload-party-battle)"
export BATTLE_REPO

make generate
