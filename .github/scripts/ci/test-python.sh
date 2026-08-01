#!/usr/bin/env bash
set -euo pipefail

pip install pyyaml pytest

python3 -m pytest scripts/tests/ -v --junitxml=pytest-report.xml
