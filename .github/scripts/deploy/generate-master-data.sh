#!/usr/bin/env bash
# 公開対象のマスターデータを SSoT から再生成する。
# 生成物はコミット済みで SSoT と食い違いうるため、公開前に再生成して差分を検出できるようにする。
set -euo pipefail

COMMON_REPO="$(.github/scripts/ci/checkout-repo.sh overload-party-common)"
export COMMON_REPO
BATTLE_REPO="$(.github/scripts/ci/checkout-repo.sh overload-party-battle)"
export BATTLE_REPO

python3 scripts/generate_cards.py
python3 scripts/generate_products.py
