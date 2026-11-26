"""db/schema.sql からスキーマ定義書 (docs/DATA_DESIGN.md) の生成部を更新する。"""
import sys
from pathlib import Path

from doc_tools import generate_schema_doc

generate_schema_doc.run(
    Path("db/schema.sql"),
    Path("docs/DATA_DESIGN.md"),
    do_add_markers="--add-markers" in sys.argv,
)
