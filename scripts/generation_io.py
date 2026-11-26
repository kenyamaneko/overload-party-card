"""生成スクリプトが共有する、定義ファイルの読み込みと生成物の書き出し。"""

import json
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("ERROR: pyyaml is required. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

JSON_INDENT_WIDTH = 2


def load_yaml_file(path):
    """YAML ファイルを読み込む。

    Raises:
        FileNotFoundError: path のファイルが無いとき。
    """
    with open(path, "r", encoding="utf-8") as file:
        return yaml.safe_load(file)


def write_json_file(out_path, value):
    """value を JSON として out_path に書き出す。出力先のディレクトリが無ければ作る。"""
    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as file:
        json.dump(value, file, ensure_ascii=False, indent=JSON_INDENT_WIDTH)
        file.write("\n")


def write_lines_file(out_path, lines):
    """lines を改行でつないで out_path に書き出す。出力先のディレクトリが無ければ作る。"""
    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as file:
        file.write("\n".join(lines))
        file.write("\n")


def quote_sql_string(text):
    """text を、単引用符をエスケープした SQL の文字列リテラルにする。None は NULL にする。"""
    if text is None:
        return "NULL"
    return "'" + text.replace("'", "''") + "'"


def quote_sql_jsonb(value):
    """value を JSON にして、SQL の jsonb リテラルにする。"""
    return quote_sql_string(json.dumps(value, ensure_ascii=False)) + "::jsonb"


def format_sql_boolean(flag):
    """flag の真偽を、SQL の true / false にする。"""
    return "true" if flag else "false"


def exit_with_error(message):
    """ERROR 行を標準エラー出力に書いて、終了コード 1 で終了する。"""
    print(f"ERROR: {message}", file=sys.stderr)
    sys.exit(1)


def exit_with_validation_errors(errors):
    """検証の誤りを 1 件ずつ標準エラー出力に書いて、終了コード 1 で終了する。"""
    print(f"Validation failed with {len(errors)} error(s):", file=sys.stderr)
    for error in errors:
        print(f"  - {error}", file=sys.stderr)
    sys.exit(1)
