#!/usr/bin/env bash
set -euo pipefail

# NSwag の版が変わると生成結果と必要な実行環境が変わるため、版を固定する
dotnet tool install -g NSwag.ConsoleCore --version 14.7.1

pip install pyyaml \
  "overload-party-doc-tools @ git+https://github.com/kenyamaneko/overload-party-common.git@main#subdirectory=packages/doc-tools"

COMMON_REPO="$(.github/scripts/ci/checkout-common.sh)"
export COMMON_REPO

make generate
