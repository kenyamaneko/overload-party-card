#!/usr/bin/env bash
set -euo pipefail

pip install pyyaml pytest testcontainers "psycopg[binary]"

python3 -m pytest scripts/tests/ -v --junitxml=pytest-report.xml
