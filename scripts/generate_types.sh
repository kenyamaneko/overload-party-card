#!/usr/bin/env bash
# data/openapi.yaml から packages/api-card (Go)、api-card-npm (TypeScript)、api-card-dotnet (C#) の型を再生成する。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$REPO_ROOT/packages/api-card"
oapi-codegen -config openapi-codegen.yaml ../../data/openapi.yaml

cd "$REPO_ROOT"
npx --yes openapi-typescript@7 \
  data/openapi.yaml \
  --output packages/api-card-npm/src/openapi.gen.ts

cd "$REPO_ROOT/packages/api-card-dotnet"
nswag run nswag.json
