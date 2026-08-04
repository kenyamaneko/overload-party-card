#!/usr/bin/env bash
# 環境名から publish 先のバケットを決める。
# 入力: ENVIRONMENT と各環境のバケット名 (env)、出力: GITHUB_OUTPUT に "bucket=<name>"。
set -euo pipefail

: "${ENVIRONMENT:?ENVIRONMENT env required}"

case "${ENVIRONMENT}" in
  dev) bucket="${DEV_BUCKET:-}" ;;
  stg) bucket="${STG_BUCKET:-}" ;;
  prod) bucket="${PROD_BUCKET:-}" ;;
  *)
    echo "::error::unknown environment: ${ENVIRONMENT}" >&2
    exit 1
    ;;
esac

# 未設定のまま進むと publish 先を誤るため、空なら止める
if [ -z "${bucket}" ]; then
  echo "::error::master data bucket variable is not set for ${ENVIRONMENT}" >&2
  exit 1
fi

echo "bucket=${bucket}" >> "${GITHUB_OUTPUT}"
