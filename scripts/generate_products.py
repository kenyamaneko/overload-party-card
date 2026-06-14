#!/usr/bin/env python3
"""Generate product data outputs from the YAML product definition.

Outputs:
  - data/cache/products_gen.json   (Go embedded via data/cache/embed.go)
  - docs/PRODUCTS.md               (human-readable product list)

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
YAML_PATH = ROOT / "data" / "products.yaml"
GO_JSON_OUT = ROOT / "data" / "cache" / "products_gen.json"
MD_OUT = ROOT / "docs" / "PRODUCTS.md"

# ─── Constants ──────────────────────────────────────────
# collectible な陣営は少なくとも 1 つのプロダクトを持つ (陣営:プロダクト = 1:N)。
COLLECTIBLE_FACTIONS = {"SHE", "Tenki", "Sugar", "Tuners"}

# 各プロダクトは区分ごとに 1 つ以上の施策を持つ (ルーチン / スペシャル)。
INITIATIVE_KINDS = ["routine", "special"]

KIND_DISPLAY = {"routine": "ルーチン（1ターン1回）", "special": "スペシャル（1ゲーム1回）"}


# ─── Load ───────────────────────────────────────────────
def load_products():
    """Load products from the YAML definition."""
    if not YAML_PATH.exists():
        print(f"ERROR: {YAML_PATH} not found", file=sys.stderr)
        sys.exit(1)
    with open(YAML_PATH, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f)
    return data.get("products", [])


# ─── Validate ──────────────────────────────────────────
def _validate_initiative(initiative, label, errors):
    for field in ("initiative_id", "kind", "name", "insight_cost", "effect_text", "effect"):
        if field not in initiative:
            errors.append(f"{label}: missing required field '{field}'")
            return

    initiative_id = initiative["initiative_id"]
    if not re.match(r"^IN-\d{4}$", initiative_id):
        errors.append(f"{label}: initiative_id '{initiative_id}' must match IN-NNNN format")
    if initiative["kind"] not in INITIATIVE_KINDS:
        errors.append(f"{label}: invalid kind '{initiative['kind']}'")

    cost = initiative["insight_cost"]
    if not isinstance(cost, int) or cost < 0:
        errors.append(f"{label}: insight_cost must be a non-negative integer, got {cost!r}")

    effect = initiative["effect"]
    if not isinstance(effect, dict) or not ("ops" in effect or "custom" in effect):
        errors.append(f"{label}: effect must contain 'ops' or 'custom'")


def validate(products):
    """Run all validation checks. Returns list of error strings."""
    errors = []
    seen_factions = {}
    seen_product_ids = {}
    seen_initiative_ids = {}

    for product in products:
        product_id = product.get("product_id", "???")
        label = f"{product_id} {product.get('product_name', '???')}"

        for field in ("product_id", "faction", "product_name", "initiatives"):
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

        initiatives = product.get("initiatives", [])
        # 各プロダクトはルーチン・スペシャルをそれぞれ 1 つ以上持つ (数は区分ごとに異なってよい)。
        # デッキはプロダクトを 1 つ選び、その中から各区分 1 つずつセットする。
        kinds = {i.get("kind") for i in initiatives}
        for required_kind in INITIATIVE_KINDS:
            if required_kind not in kinds:
                errors.append(f"{label}: initiatives must include at least one '{required_kind}'")
        for initiative in initiatives:
            iid = initiative.get("initiative_id")
            ilabel = f"{label} / {initiative.get('name', '???')}"
            _validate_initiative(initiative, ilabel, errors)
            if iid is not None:
                if iid in seen_initiative_ids:
                    errors.append(f"{ilabel}: duplicate initiative_id '{iid}' (also used by {seen_initiative_ids[iid]})")
                seen_initiative_ids[iid] = product_id

    missing = COLLECTIBLE_FACTIONS - set(seen_factions)
    if missing:
        errors.append(f"factions without a product: {sorted(missing)}")

    return errors


# ─── Generate JSON ─────────────────────────────────────
def generate_json(products, *, out_path):
    """Generate products_gen.json."""
    output = []
    for product in sorted(products, key=lambda p: p["product_id"]):
        output.append({
            "product_id": product["product_id"],
            "faction": product["faction"],
            "product_name": product["product_name"],
            "initiatives": [
                {
                    "initiative_id": i["initiative_id"],
                    "kind": i["kind"],
                    "name": i["name"],
                    "insight_cost": i["insight_cost"],
                    "effect_text": i["effect_text"],
                    "effect": i["effect"],
                }
                for i in product["initiatives"]
            ],
        })

    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        json.dump(output, f, ensure_ascii=False, indent=2)
        f.write("\n")
    return len(output)


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
    if not products:
        print("ERROR: No products loaded from YAML", file=sys.stderr)
        sys.exit(1)

    errors = validate(products)
    if errors:
        print(f"Validation failed with {len(errors)} error(s):", file=sys.stderr)
        for err in errors:
            print(f"  - {err}", file=sys.stderr)
        sys.exit(1)

    count = generate_json(products, out_path=GO_JSON_OUT)
    print(f"Generated {count} products → {GO_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    count = generate_md(products, out_path=MD_OUT)
    print(f"Generated {count} products → {MD_OUT.relative_to(ROOT)}", file=sys.stderr)


if __name__ == "__main__":
    main()
