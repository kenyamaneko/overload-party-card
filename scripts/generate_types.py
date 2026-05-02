#!/usr/bin/env python3
"""data/models.yaml から packages/api-card/*_gen.go を生成する.

共通基盤 `overload-party-codegen-tools` の `CodegenRunner` を使う。card は
出力先が単一 (packages/api-card) で、`type_aliases` と `constants` ブロック
(型注釈つき) をサポートする。

実行: python3 scripts/generate_types.py
"""

from __future__ import annotations

import sys
from pathlib import Path

from codegen_tools import CodegenRunner, GoConstStyle, GoStyle, GoTarget

REPO_ROOT = Path(__file__).resolve().parent.parent
MODELS_YAML = REPO_ROOT / "data" / "models.yaml"


def main() -> int:
    style = GoStyle(
        tag_keys=("json", "db"),
        const_style=GoConstStyle(type_annotation=True),
    )
    # card の自動 import 検出には civil は含めない (元スクリプト互換)。
    style.import_patterns.pop("civil.", None)

    runner = CodegenRunner(
        models_yaml=MODELS_YAML,
        repo_root=REPO_ROOT,
        targets={
            "default": GoTarget(
                out_dir=REPO_ROOT / "packages" / "api-card",
                package="apicard",
                emit_tags=("json", "db"),
            ),
        },
        style=style,
        default_target_key="default",
        single_target_field=None,
        multi_target_field=None,
        trailing_blank_line=True,
    )
    return runner.run()


if __name__ == "__main__":
    sys.exit(main())
