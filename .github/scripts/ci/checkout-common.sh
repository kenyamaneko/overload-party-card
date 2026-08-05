#!/usr/bin/env bash
# checkout-common.sh — カード生成器が読むゲーム定数を持つ common を取得し、その場所を標準出力へ返す。
# 呼び出し前に private リポジトリを取得できる git 認証が設定されていること。
set -euo pipefail

# 生成物の差分検出が未追跡ファイルとして拾わないよう、card の作業ツリーの外へ置く
target="${RUNNER_TEMP}/overload-party-common"

rm -rf "$target"
git clone --depth 1 --branch main https://github.com/kenyamaneko/overload-party-common.git "$target" >&2

echo "$target"
