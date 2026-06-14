#!/usr/bin/env python3
"""Generate product data outputs from the YAML product / initiative definitions.

Inputs (SSoT, edited by humans):
  - data/products.yaml      (products only)
  - data/initiatives.yaml   (initiatives, referencing products by product_id)

Outputs:
  - data/cache/products_gen.json     (Go embedded via data/cache/embed.go)
  - data/cache/initiatives_gen.json  (Go embedded via data/cache/embed.go)
  - db/seed/products_seed.sql        (PostgreSQL UPSERT seed: products)
  - db/seed/initiatives_seed.sql     (PostgreSQL UPSERT seed: initiatives)
  - docs/PRODUCTS.md                 (human-readable product list)

Usage:
    python3 scripts/generate_products.py
"""

import json
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("ERROR: pyyaml is required. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

# ─── Paths ──────────────────────────────────────────────
ROOT = Path(__file__).resolve().parent.parent
PRODUCTS_YAML_PATH = ROOT / "data" / "products.yaml"
INITIATIVES_YAML_PATH = ROOT / "data" / "initiatives.yaml"
PRODUCTS_JSON_OUT = ROOT / "data" / "cache" / "products_gen.json"
INITIATIVES_JSON_OUT = ROOT / "data" / "cache" / "initiatives_gen.json"
PRODUCTS_SEED_OUT = ROOT / "db" / "seed" / "products_seed.sql"
INITIATIVES_SEED_OUT = ROOT / "db" / "seed" / "initiatives_seed.sql"
MD_OUT = ROOT / "docs" / "PRODUCTS.md"

# ─── Constants ──────────────────────────────────────────
# collectible な陣営は少なくとも 1 つのプロダクトを持つ (陣営:プロダクト = 1:N)。
COLLECTIBLE_FACTIONS = {"SHE", "Tenki", "Sugar", "Tuners"}

# 各プロダクトは区分ごとに 1 つ以上の施策を持つ (ルーチン / スペシャル)。
INITIATIVE_KINDS = ["routine", "special"]

KIND_DISPLAY = {"routine": "ルーチン（1ターン1回）", "special": "スペシャル（1ゲーム1回）"}


# ─── Load ───────────────────────────────────────────────
def _load_yaml(path, key):
    """Load a top-level list from a YAML file."""
    if not path.exists():
        print(f"ERROR: {path} not found", file=sys.stderr)
        sys.exit(1)
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f)
    return data.get(key, [])


def load_products():
    """Load products from data/products.yaml."""
    return _load_yaml(PRODUCTS_YAML_PATH, "products")


def load_initiatives():
    """Load initiatives from data/initiatives.yaml."""
    return _load_yaml(INITIATIVES_YAML_PATH, "initiatives")


# ─── Validate ──────────────────────────────────────────
def _validate_initiative(initiative, label, product_ids, errors):
    for field in ("initiative_id", "product_id", "kind", "name", "insight_cost", "effect_text", "effect"):
        if field not in initiative:
            errors.append(f"{label}: missing required field '{field}'")
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


def validate(products, initiatives):
    """Run all validation checks. Returns list of error strings."""
    errors = []
    seen_factions = {}
    seen_product_ids = {}
    seen_initiative_ids = {}

    for product in products:
        product_id = product.get("product_id", "???")
        label = f"{product_id} {product.get('product_name', '???')}"

        for field in ("product_id", "faction", "product_name"):
            if field not in product:
                errors.append(f"{label}: missing required field '{field}'")

        if not re.match(r"^PD-\d{4}$", product_id):
            errors.append(f"{label}: product_id '{product_id}' must match PD-NNNN format")
        if product_id in seen_product_ids:
            errors.append(f"{label}: duplicate product_id")
        seen_product_ids[product_id] = True

        # 陣営はプロダクトを複数持てる (1:N)。重複は許容し、faction → product の存在のみ追跡する。
        faction = product.get("faction", "")
        if faction not in COLLECTIBLE_FACTIONS:
            errors.append(f"{label}: invalid faction '{faction}'")
        seen_factions[faction] = product_id

    kinds_by_product = {pid: set() for pid in seen_product_ids}
    for initiative in initiatives:
        iid = initiative.get("initiative_id")
        ilabel = f"{initiative.get('product_id', '???')} / {initiative.get('name', '???')}"
        _validate_initiative(initiative, ilabel, seen_product_ids, errors)
        if iid is not None:
            if iid in seen_initiative_ids:
                errors.append(f"{ilabel}: duplicate initiative_id '{iid}' (also used by {seen_initiative_ids[iid]})")
            seen_initiative_ids[iid] = initiative.get("product_id")
        pid = initiative.get("product_id")
        if pid in kinds_by_product:
            kinds_by_product[pid].add(initiative.get("kind"))

    # 各プロダクトはルーチン・スペシャルをそれぞれ 1 つ以上持つ (数は区分ごとに異なってよい)。
    for product_id, kinds in kinds_by_product.items():
        for required_kind in INITIATIVE_KINDS:
            if required_kind not in kinds:
                errors.append(f"{product_id}: must have at least one '{required_kind}' initiative")

    missing = COLLECTIBLE_FACTIONS - set(seen_factions)
    if missing:
        errors.append(f"factions without a product: {sorted(missing)}")

    return errors


# ─── Assemble ──────────────────────────────────────────
def assemble(products, initiatives):
    """Attach each product's initiatives (in file order) under an 'initiatives' key."""
    by_product = {}
    for initiative in initiatives:
        by_product.setdefault(initiative["product_id"], []).append(initiative)

    assembled = []
    for product in products:
        assembled.append({**product, "initiatives": by_product.get(product["product_id"], [])})
    return assembled


# ─── Generate JSON ─────────────────────────────────────
def _write_json(out_path, output):
    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        json.dump(output, f, ensure_ascii=False, indent=2)
        f.write("\n")


def generate_products_json(products, *, out_path):
    """Generate products_gen.json (flat products)."""
    output = [
        {
            "product_id": p["product_id"],
            "faction": p["faction"],
            "product_name": p["product_name"],
        }
        for p in sorted(products, key=lambda p: p["product_id"])
    ]
    _write_json(out_path, output)
    return len(output)


def generate_initiatives_json(initiatives, *, out_path):
    """Generate initiatives_gen.json (flat initiatives)."""
    output = [
        {
            "initiative_id": i["initiative_id"],
            "product_id": i["product_id"],
            "kind": i["kind"],
            "name": i["name"],
            "insight_cost": i["insight_cost"],
            "effect_text": i["effect_text"],
            "effect": i["effect"],
        }
        for i in sorted(initiatives, key=lambda i: i["initiative_id"])
    ]
    _write_json(out_path, output)
    return len(output)


# ─── Generate Seed SQL ─────────────────────────────────
def _sql_escape(s):
    """Escape a string for a PostgreSQL single-quoted literal."""
    if s is None:
        return "NULL"
    return "'" + s.replace("'", "''") + "'"


def _write_sql(out_path, lines):
    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
        f.write("\n")


def generate_products_seed_sql(products, *, out_path):
    """Generate db/seed/products_seed.sql with UPSERT statements for products."""
    sorted_products = sorted(products, key=lambda p: p["product_id"])

    lines = []
    lines.append("-- Auto-generated by scripts/generate_products.py. DO NOT EDIT.")
    lines.append("")
    lines.append("INSERT INTO card.products (product_id, faction, product_name)")
    lines.append("VALUES")
    product_rows = [
        f"  ({_sql_escape(p['product_id'])}, {_sql_escape(p['faction'])}, {_sql_escape(p['product_name'])})"
        for p in sorted_products
    ]
    lines.append(",\n".join(product_rows))
    lines.append("ON CONFLICT (product_id) DO UPDATE SET")
    lines.append("  faction      = EXCLUDED.faction,")
    lines.append("  product_name = EXCLUDED.product_name,")
    lines.append("  updated_at   = now();")
    lines.append("")

    _write_sql(out_path, lines)
    return len(sorted_products)


def generate_initiatives_seed_sql(initiatives, *, out_path):
    """Generate db/seed/initiatives_seed.sql with UPSERT statements for initiatives.

    initiatives は products への FK を持つため products_seed.sql を先に適用すること。
    """
    sorted_initiatives = sorted(initiatives, key=lambda i: i["initiative_id"])

    lines = []
    lines.append("-- Auto-generated by scripts/generate_products.py. DO NOT EDIT.")
    lines.append("-- 先に products_seed.sql を適用すること (product_id への FK 制約)。")
    lines.append("")
    lines.append("INSERT INTO card.initiatives")
    lines.append("  (initiative_id, product_id, kind, name, insight_cost, effect_text, effect)")
    lines.append("VALUES")

    rows = []
    for initiative in sorted_initiatives:
        effect_json = _sql_escape(json.dumps(initiative["effect"], ensure_ascii=False)) + "::jsonb"
        rows.append(
            f"  ({_sql_escape(initiative['initiative_id'])}, {_sql_escape(initiative['product_id'])}, "
            f"{_sql_escape(initiative['kind'])}, {_sql_escape(initiative['name'])}, "
            f"{initiative['insight_cost']}, {_sql_escape(initiative['effect_text'])},\n"
            f"   {effect_json})"
        )
    lines.append(",\n".join(rows))
    lines.append("ON CONFLICT (initiative_id) DO UPDATE SET")
    lines.append("  product_id   = EXCLUDED.product_id,")
    lines.append("  kind         = EXCLUDED.kind,")
    lines.append("  name         = EXCLUDED.name,")
    lines.append("  insight_cost = EXCLUDED.insight_cost,")
    lines.append("  effect_text  = EXCLUDED.effect_text,")
    lines.append("  effect       = EXCLUDED.effect,")
    lines.append("  updated_at   = now();")
    lines.append("")

    _write_sql(out_path, lines)
    return len(sorted_initiatives)


# ─── Generate PRODUCTS.md ──────────────────────────────
def generate_md(products, *, out_path):
    """Generate docs/PRODUCTS.md."""
    lines = []
    lines.append("<!-- This file is auto-generated by scripts/generate_products.py. DO NOT EDIT. -->")
    lines.append("")
    lines.append("# Overload Party — Product List")
    lines.append("")

    for product in sorted(products, key=lambda p: p["product_id"]):
        lines.append(f"## {product['product_name']}（{product['faction']}）— {product['product_id']}")
        lines.append("")
        lines.append("| ID | 区分 | 施策 | Insight コスト | 効果 |")
        lines.append("|------|------|------|------|------|")
        for kind in INITIATIVE_KINDS:
            for initiative in [i for i in product["initiatives"] if i["kind"] == kind]:
                lines.append(
                    f"| {initiative['initiative_id']} | {KIND_DISPLAY[kind]} | {initiative['name']} "
                    f"| {initiative['insight_cost']} | {initiative['effect_text']} |"
                )
        lines.append("")

    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
        f.write("\n")
    return len(products)


# ─── Main ──────────────────────────────────────────────
def main():
    products = load_products()
    initiatives = load_initiatives()
    if not products:
        print("ERROR: No products loaded from YAML", file=sys.stderr)
        sys.exit(1)
    if not initiatives:
        print("ERROR: No initiatives loaded from YAML", file=sys.stderr)
        sys.exit(1)

    errors = validate(products, initiatives)
    if errors:
        print(f"Validation failed with {len(errors)} error(s):", file=sys.stderr)
        for err in errors:
            print(f"  - {err}", file=sys.stderr)
        sys.exit(1)

    assembled = assemble(products, initiatives)

    count = generate_products_json(products, out_path=PRODUCTS_JSON_OUT)
    print(f"Generated {count} products → {PRODUCTS_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_initiatives_json(initiatives, out_path=INITIATIVES_JSON_OUT)
    print(f"Generated {count} initiatives → {INITIATIVES_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_products_seed_sql(products, out_path=PRODUCTS_SEED_OUT)
    print(f"Generated {count} products → {PRODUCTS_SEED_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_initiatives_seed_sql(initiatives, out_path=INITIATIVES_SEED_OUT)
    print(f"Generated {count} initiatives → {INITIATIVES_SEED_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_md(assembled, out_path=MD_OUT)
    print(f"Generated {count} products → {MD_OUT.relative_to(ROOT)}", file=sys.stderr)


if __name__ == "__main__":
    main()
