#!/usr/bin/env bash
# generate-master-data.sh — publish 対象のマスターデータを SSoT から再生成する。
# 生成物はコミット済みで SSoT と食い違いうるため、publish 前に再生成して差分を検出できるようにする。
set -euo pipefail

COMMON_REPO="$(.github/scripts/ci/checkout-common.sh)"
export COMMON_REPO

python3 scripts/generate_cards.py
python3 scripts/generate_products.py
