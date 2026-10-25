#!/usr/bin/env bash
set -euo pipefail

# Testcontainers はテスト中にイメージを自動で取得するが、時間がかかりタイムアウトの原因になるため事前に取得する
docker pull postgres:16-alpine
