#!/usr/bin/env bash
# data/openapi.yaml から packages/api-card (Go) と packages/api-card-npm (TS) の型を再生成する。
# REST 部分は oapi-codegen / openapi-typescript を使う。card は Pub/Sub event を publish
# しないため AsyncAPI は無い (subscriber は consume する shop / scenario 側 spec から型を import する)。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$REPO_ROOT/packages/api-card"
oapi-codegen -config openapi-codegen.yaml ../../data/openapi.yaml

cd "$REPO_ROOT"
npx --yes openapi-typescript@7 \
  data/openapi.yaml \
  --output packages/api-card-npm/src/openapi.gen.ts
