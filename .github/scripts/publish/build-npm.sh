#!/usr/bin/env bash
# 算出済みタグから npm パッケージをビルドし、package.json のバージョンに反映する。
# 入力: TAG (環境変数。接頭辞 `packages/api-card-npm/v` を含む完全なタグ名)、カレントディレクトリは packages/api-card-npm/。
set -euo pipefail

: "${TAG:?TAG env required}"

version="${TAG#packages/api-card-npm/v}"
npm install
npm version "${version}" --no-git-tag-version
npm run build
