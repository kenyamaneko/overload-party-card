#!/usr/bin/env python3
"""YAML のプロダクト定義と施策定義から、配布用 JSON・初期投入 SQL・プロダクト一覧文書を生成する。

    python3 scripts/generate_products.py
"""

import re
import sys
from pathlib import Path

from generation_io import (
    exit_with_error,
    exit_with_validation_errors,
    format_sql_boolean,
    load_yaml_file,
    quote_sql_jsonb,
    quote_sql_string,
    write_json_file,
    write_lines_file,
)

ROOT = Path(__file__).resolve().parent.parent
PRODUCTS_YAML_PATH = ROOT / "data" / "products.yaml"
INITIATIVES_YAML_PATH = ROOT / "data" / "initiatives.yaml"
PRODUCTS_JSON_OUT = ROOT / "data" / "cache" / "products_gen.json"
INITIATIVES_JSON_OUT = ROOT / "data" / "cache" / "initiatives_gen.json"
PRODUCTS_SEED_OUT = ROOT / "db" / "seed" / "products_seed.sql"
INITIATIVES_SEED_OUT = ROOT / "db" / "seed" / "initiatives_seed.sql"
MD_OUT = ROOT / "docs" / "PRODUCTS.md"

COLLECTIBLE_FACTIONS = {"SHE", "Tenki", "Sugar", "Tuners"}
INITIATIVE_KINDS = ["routine", "special"]
KIND_DISPLAY = {"routine": "ルーチン（1ターン1回）", "special": "スペシャル（1ゲーム1回）"}
IS_ACTIVE_WHEN_OMITTED = True


def _load_top_level_list(path, key):
    return load_yaml_file(path).get(key, [])


def load_products():
    """プロダクト定義を読み込む。

    Raises:
        FileNotFoundError: プロダクト定義のファイルが無いとき。
    """
    return _load_top_level_list(PRODUCTS_YAML_PATH, "products")


def load_initiatives():
    """施策定義を読み込む。

    Raises:
        FileNotFoundError: 施策定義のファイルが無いとき。
    """
    return _load_top_level_list(INITIATIVES_YAML_PATH, "initiatives")


def _validate_initiative(initiative, label, product_ids, errors):
    for field_name in ("initiative_id", "product_id", "kind", "name", "insight_cost", "effect_text", "effect"):
        if field_name not in initiative:
            errors.append(f"{label}: missing required field '{field_name}'")
            return

    initiative_id = initiative["initiative_id"]
    if not re.match(r"^IN-\d{4}$", initiative_id):
        errors.append(f"{label}: initiative_id '{initiative_id}' must match IN-NNNN format")
    if initiative["product_id"] not in product_ids:
        errors.append(f"{label}: product_id '{initiative['product_id']}' does not reference a known product")
    if initiative["kind"] not in INITIATIVE_KINDS:
        errors.append(f"{label}: invalid kind '{initiative['kind']}'")

    cost = initiative["insight_cost"]
    if not isinstance(cost, int) or cost < 0:
        errors.append(f"{label}: insight_cost must be a non-negative integer, got {cost!r}")

    effect = initiative["effect"]
    if not isinstance(effect, dict) or not ("ops" in effect or "custom" in effect):
        errors.append(f"{label}: effect must contain 'ops' or 'custom'")

    if "is_active" in initiative and not isinstance(initiative["is_active"], bool):
        errors.append(f"{label}: is_active must be a boolean")


def validate_products_and_initiatives(products, initiatives):
    """プロダクト定義と施策定義を検証し、誤りの文字列のリストを返す。"""
    errors = []
    seen_factions = {}
    seen_product_ids = {}
    seen_initiative_ids = {}

    for product in products:
        product_id = product.get("product_id", "???")
        label = f"{product_id} {product.get('product_name', '???')}"

        for field_name in ("product_id", "faction", "product_name"):
            if field_name not in product:
                errors.append(f"{label}: missing required field '{field_name}'")

        if not re.match(r"^PD-\d{4}$", product_id):
            errors.append(f"{label}: product_id '{product_id}' must match PD-NNNN format")
        if product_id in seen_product_ids:
            errors.append(f"{label}: duplicate product_id")
        seen_product_ids[product_id] = True

        # 陣営は複数のプロダクトを持てるため、陣営の重複は誤りにしない
        faction = product.get("faction", "")
        if faction not in COLLECTIBLE_FACTIONS:
            errors.append(f"{label}: invalid faction '{faction}'")
        seen_factions[faction] = product_id

        if "is_active" in product and not isinstance(product["is_active"], bool):
            errors.append(f"{label}: is_active must be a boolean")

    kinds_by_product = {product_id: set() for product_id in seen_product_ids}
    for initiative in initiatives:
        initiative_id = initiative.get("initiative_id")
        initiative_label = f"{initiative.get('product_id', '???')} / {initiative.get('name', '???')}"
        _validate_initiative(initiative, initiative_label, seen_product_ids, errors)
        if initiative_id is not None:
            if initiative_id in seen_initiative_ids:
                errors.append(
                    f"{initiative_label}: duplicate initiative_id '{initiative_id}' "
                    f"(also used by {seen_initiative_ids[initiative_id]})"
                )
            seen_initiative_ids[initiative_id] = initiative.get("product_id")
        referenced_product_id = initiative.get("product_id")
        if referenced_product_id in kinds_by_product:
            kinds_by_product[referenced_product_id].add(initiative.get("kind"))

    for product_id, kinds in kinds_by_product.items():
        for required_kind in INITIATIVE_KINDS:
            if required_kind not in kinds:
                errors.append(f"{product_id}: must have at least one '{required_kind}' initiative")

    missing = COLLECTIBLE_FACTIONS - set(seen_factions)
    if missing:
        errors.append(f"factions without a product: {sorted(missing)}")

    return errors


def _attach_initiatives_to_products(products, initiatives):
    initiatives_by_product = {product["product_id"]: [] for product in products}
    for initiative in initiatives:
        initiatives_by_product[initiative["product_id"]].append(initiative)

    return [
        {**product, "initiatives": initiatives_by_product[product["product_id"]]}
        for product in products
    ]


def generate_products_json(products, *, out_path):
    """プロダクトの配布用 JSON を product_id の順に書き出す。

    Returns:
        書き出したプロダクトの件数。
    """
    output = [
        {
            "product_id": product["product_id"],
            "faction": product["faction"],
            "product_name": product["product_name"],
            "is_active": product.get("is_active", IS_ACTIVE_WHEN_OMITTED),
        }
        for product in sorted(products, key=lambda product: product["product_id"])
    ]
    write_json_file(out_path, output)
    return len(output)


def generate_initiatives_json(initiatives, *, out_path):
    """施策の配布用 JSON を initiative_id の順に書き出す。

    Returns:
        書き出した施策の件数。
    """
    output = [
        {
            "initiative_id": initiative["initiative_id"],
            "product_id": initiative["product_id"],
            "kind": initiative["kind"],
            "name": initiative["name"],
            "insight_cost": initiative["insight_cost"],
            "effect_text": initiative["effect_text"],
            "effect": initiative["effect"],
            "is_active": initiative.get("is_active", IS_ACTIVE_WHEN_OMITTED),
        }
        for initiative in sorted(initiatives, key=lambda initiative: initiative["initiative_id"])
    ]
    write_json_file(out_path, output)
    return len(output)


def generate_products_seed_sql(products, *, out_path):
    """プロダクトの初期投入 SQL を product_id の順に書き出す。

    Returns:
        書き出したプロダクトの件数。
    """
    sorted_products = sorted(products, key=lambda product: product["product_id"])

    lines = []
    lines.append("-- Auto-generated by scripts/generate_products.py. DO NOT EDIT.")
    lines.append("")
    lines.append("INSERT INTO card.products (product_id, faction, product_name, is_active)")
    lines.append("VALUES")
    product_rows = [
        f"  ({quote_sql_string(product['product_id'])}, {quote_sql_string(product['faction'])}, "
        f"{quote_sql_string(product['product_name'])}, "
        f"{format_sql_boolean(product.get('is_active', IS_ACTIVE_WHEN_OMITTED))})"
        for product in sorted_products
    ]
    lines.append(",\n".join(product_rows))
    lines.append("ON CONFLICT (product_id) DO UPDATE SET")
    lines.append("  faction      = EXCLUDED.faction,")
    lines.append("  product_name = EXCLUDED.product_name,")
    lines.append("  is_active    = EXCLUDED.is_active,")
    lines.append("  updated_at   = now();")
    lines.append("")

    write_lines_file(out_path, lines)
    return len(sorted_products)


def generate_initiatives_seed_sql(initiatives, *, out_path):
    """施策の初期投入 SQL を initiative_id の順に書き出す。

    Returns:
        書き出した施策の件数。
    """
    sorted_initiatives = sorted(initiatives, key=lambda initiative: initiative["initiative_id"])

    lines = []
    lines.append("-- Auto-generated by scripts/generate_products.py. DO NOT EDIT.")
    lines.append("-- 先に products_seed.sql を適用すること (product_id への FK 制約)。")
    lines.append("")
    lines.append("INSERT INTO card.initiatives")
    lines.append("  (initiative_id, product_id, kind, name, insight_cost, effect_text, effect, is_active)")
    lines.append("VALUES")

    rows = []
    for initiative in sorted_initiatives:
        effect_sql = quote_sql_jsonb(initiative["effect"])
        is_active = format_sql_boolean(initiative.get("is_active", IS_ACTIVE_WHEN_OMITTED))
        rows.append(
            f"  ({quote_sql_string(initiative['initiative_id'])}, {quote_sql_string(initiative['product_id'])}, "
            f"{quote_sql_string(initiative['kind'])}, {quote_sql_string(initiative['name'])}, "
            f"{initiative['insight_cost']}, {quote_sql_string(initiative['effect_text'])},\n"
            f"   {effect_sql}, {is_active})"
        )
    lines.append(",\n".join(rows))
    lines.append("ON CONFLICT (initiative_id) DO UPDATE SET")
    lines.append("  product_id   = EXCLUDED.product_id,")
    lines.append("  kind         = EXCLUDED.kind,")
    lines.append("  name         = EXCLUDED.name,")
    lines.append("  insight_cost = EXCLUDED.insight_cost,")
    lines.append("  effect_text  = EXCLUDED.effect_text,")
    lines.append("  effect       = EXCLUDED.effect,")
    lines.append("  is_active    = EXCLUDED.is_active,")
    lines.append("  updated_at   = now();")
    lines.append("")

    write_lines_file(out_path, lines)
    return len(sorted_initiatives)


def generate_products_md(products, initiatives, *, out_path):
    """プロダクト一覧文書を書き出す。

    Args:
        products: 検証済みのプロダクト定義。
        initiatives: 検証済みの施策定義。各プロダクトの節に、そのプロダクトの施策が載る。
        out_path: 書き出す場所。

    Returns:
        書き出したプロダクトの件数。
    """
    lines = []
    lines.append("<!-- This file is auto-generated by scripts/generate_products.py. DO NOT EDIT. -->")
    lines.append("")
    lines.append("# Overload Party Product List")
    lines.append("")

    attached_products = _attach_initiatives_to_products(products, initiatives)
    for product in sorted(attached_products, key=lambda product: product["product_id"]):
        lines.append(f"## {product['product_name']}（{product['faction']}、{product['product_id']}）")
        lines.append("")
        lines.append("| ID | 区分 | 施策 | 使用コスト | 効果 |")
        lines.append("|------|------|------|------|------|")
        for kind in INITIATIVE_KINDS:
            kind_initiatives = [
                initiative for initiative in product["initiatives"] if initiative["kind"] == kind
            ]
            for initiative in kind_initiatives:
                lines.append(
                    f"| {initiative['initiative_id']} | {KIND_DISPLAY[kind]} | {initiative['name']} "
                    f"| {initiative['insight_cost']} | {initiative['effect_text']} |"
                )
        lines.append("")

    write_lines_file(out_path, lines)
    return len(products)


def main():
    """プロダクト定義と施策定義を検証し、配布用 JSON・初期投入 SQL・プロダクト一覧文書を生成する。"""
    try:
        products = load_products()
        initiatives = load_initiatives()
    except FileNotFoundError as err:
        exit_with_error(f"{err.filename} not found")
    if not products:
        exit_with_error("No products loaded from YAML")
    if not initiatives:
        exit_with_error("No initiatives loaded from YAML")

    errors = validate_products_and_initiatives(products, initiatives)
    if errors:
        exit_with_validation_errors(errors)

    count = generate_products_json(products, out_path=PRODUCTS_JSON_OUT)
    print(f"Generated {count} products → {PRODUCTS_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_initiatives_json(initiatives, out_path=INITIATIVES_JSON_OUT)
    print(f"Generated {count} initiatives → {INITIATIVES_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_products_seed_sql(products, out_path=PRODUCTS_SEED_OUT)
    print(f"Generated {count} products → {PRODUCTS_SEED_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_initiatives_seed_sql(initiatives, out_path=INITIATIVES_SEED_OUT)
    print(f"Generated {count} initiatives → {INITIATIVES_SEED_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_products_md(products, initiatives, out_path=MD_OUT)
    print(f"Generated {count} products → {MD_OUT.relative_to(ROOT)}", file=sys.stderr)


if __name__ == "__main__":
    main()
