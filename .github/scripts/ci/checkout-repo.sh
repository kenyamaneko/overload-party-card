#!/usr/bin/env bash
# 生成器が読む定数を持つリポジトリ (引数: リポジトリ名) を取得し、その場所を標準出力へ返す。
# 呼び出し前に、非公開リポジトリを取得できる git の認証が設定されていること。
set -euo pipefail

repo="$1"

# 生成物の差分検出が未追跡ファイルとして拾わないよう、card の作業ツリーの外へ置く
target="${RUNNER_TEMP}/${repo}"

rm -rf "$target"
git clone --depth 1 --branch main "https://github.com/kenyamaneko/${repo}.git" "$target" >&2

echo "$target"
