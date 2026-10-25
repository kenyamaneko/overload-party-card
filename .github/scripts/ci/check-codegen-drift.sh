#!/usr/bin/env bash
# コード生成の後に、差分や未追跡ファイルが無いかを検証する。
set -euo pipefail

if ! git diff --exit-code; then
  echo "::error::Generated files are out of sync. Run 'make generate' and commit."
  exit 1
fi
if [ -n "$(git ls-files --others --exclude-standard)" ]; then
  echo "::error::Codegen produced untracked files:"
  git ls-files --others --exclude-standard
  exit 1
fi
