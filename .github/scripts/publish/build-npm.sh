#!/usr/bin/env bash
# 算出済みタグから npm パッケージを build し package.json に version を反映する。
# 入力: TAG (env、prefix `packages/api-card-npm/v` を含む完全タグ名)、PWD は packages/api-card-npm/。
set -euo pipefail

: "${TAG:?TAG env required}"

version="${TAG#packages/api-card-npm/v}"
npm install
npm version "${version}" --no-git-tag-version
npm run build
