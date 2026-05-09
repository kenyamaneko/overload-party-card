#!/usr/bin/env bash
# generate_types.sh — data/openapi.yaml から packages/api-card の Go 型を再生成する。
#
# REST 部分のみ。card は Pub/Sub event を publish しないため AsyncAPI は無い
# (subscriber は consume する shop / scenario 側 spec から型を import する)。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$REPO_ROOT/packages/api-card"
oapi-codegen -config openapi-codegen.yaml ../../data/openapi.yaml
