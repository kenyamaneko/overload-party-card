#!/usr/bin/env bash
# resolve-master-data-bucket.sh — publish 先のバケットを決め、対象の指定に漏れが無いか確かめる。
set -euo pipefail

: "${ENVIRONMENT:?ENVIRONMENT env required}"

case "${ENVIRONMENT}" in
  dev) bucket="${DEV_BUCKET:-}" ;;
  stg) bucket="${STG_BUCKET:-}" ;;
  prod) bucket="${PROD_BUCKET:-}" ;;
  *)
    echo "::error::unknown environment: ${ENVIRONMENT}"
    exit 1
    ;;
esac

# 未設定のまま進むと publish 先を誤るため、空なら止める
if [ -z "${bucket}" ]; then
  echo "::error::master data bucket variable is not set for ${ENVIRONMENT}"
  exit 1
fi

# stg / prod は稼働中のイメージと同じコミットのマスターデータを載せる必要がある。
# 省略を許すと main の先端が黙って入るため、指定が無ければ止める。
if [ "${ENVIRONMENT}" != "dev" ] && [ -z "${VERSION:-}" ]; then
  echo "::error::version is required to publish to ${ENVIRONMENT}; pass the tag that the environment is running"
  exit 1
fi

echo "bucket=${bucket}" >> "${GITHUB_OUTPUT}"
