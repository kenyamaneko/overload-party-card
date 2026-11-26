#!/usr/bin/env bash
# 算出済みタグを git で打ち、リモートへプッシュする。
# 入力: TAG (env)
set -euo pipefail

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git tag "${TAG}"
git push origin "${TAG}"
