#!/usr/bin/env python3
"""Generate card data outputs from YAML card definitions.

Outputs:
  - data/cache/cards_gen.json                     (Go embedded via data/cache/embed.go)
  - docs/CARDS.md                                 (human-readable card list)
  - db/seed/cards_seed.sql                        (PostgreSQL UPSERT seed)

Usage:
    python3 scripts/generate_cards.py
"""

import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path

try:
    import yaml
except ImportError:
    print("ERROR: pyyaml is required. Install with: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

# ─── Paths ──────────────────────────────────────────────
ROOT = Path(__file__).resolve().parent.parent
YAML_DIR = ROOT / "data" / "cards"
MD_OUT = ROOT / "docs" / "CARDS.md"
SEED_OUT = ROOT / "db" / "seed" / "cards_seed.sql"
GO_JSON_OUT = ROOT / "data" / "cache" / "cards_gen.json"
STARTER_DECKS_YAML = ROOT / "data" / "mock" / "starter_decks.yaml"
STARTER_DECKS_JSON_OUT = ROOT / "data" / "cache" / "starter_decks_gen.json"

# CI では common を兄弟ディレクトリに置けないため、場所を環境変数で差し替えられるようにする
COMMON_REPO_ENV = "COMMON_REPO"
DEFAULT_COMMON_REPO = ROOT.parent / "overload-party-common"
GAME_DESIGN_CONSTANTS_RELPATH = Path("data") / "game_design_constants.yaml"

# ─── Constants ──────────────────────────────────────────
# 3階層分類（Zone / CardType / Subtype）。Phase 2b で導入。
# 参考: overload-party-battle/memory/project_card_taxonomy.md
CARD_TYPE_COMPUTE = "Compute"
CARD_TYPE_DATA = "DataResource"
CARD_TYPE_LOG = "Log"
RESOURCE_CARD_TYPES = {CARD_TYPE_COMPUTE, CARD_TYPE_DATA}
SUPPORT_CARD_TYPES = {"Platform", "Attachment", "Strategy", "Reactive", "Incident"}
ALL_CARD_TYPES = RESOURCE_CARD_TYPES | SUPPORT_CARD_TYPES | {CARD_TYPE_LOG}

# Subtype を持つ CardType と、その許容 Subtype 集合。
SUBTYPES_BY_CARD_TYPE = {
    CARD_TYPE_COMPUTE: {"VM", "Container", "Orchestrator", "Serverless", "AI/ML"},
    CARD_TYPE_DATA:    {"Database", "ObjectStorage", "CacheDB"},
}

EFFECT_GROUP_KEY = "group"
EFFECT_CARD_TYPE_KEY = "card_type"
EFFECT_SUBTYPE_KEY = "subtype"
SUBTYPES_KEY = "subtypes"
RESOURCE_GROUPS_KEY = "resource_groups"
CARD_TYPES_KEY = "card_types"

VALID_RESTRICTION = {"unlimited", "limited", "semi_limited", "forbidden"}
VALID_FACTIONS = {"SHE", "Tenki", "Sugar", "Tuners", "Neutral"}

COMPUTE_STAT_KEYS = {"throughput", "availability", "maintenance_cost", "sla_penalty"}
DATA_STAT_KEYS = {"yield", "availability", "maintenance_cost", "sla_penalty"}

FACTION_ORDER = ["SHE", "Tenki", "Sugar", "Tuners", "Neutral"]
FACTION_FILE_MAP = {"SHE": "sd"}

# 各表示カテゴリのフィルタ条件: (by_field, allowed_values, table_kind)
# table_kind: "compute" | "data" | "support" — Markdown 表のフォーマットを決める
CATEGORY_ORDER = [
    ("コンピュート系リソース", "subtype",   {"VM", "Container", "Orchestrator", "Serverless"}, "compute"),
    ("AI/ML系リソース",        "subtype",   {"AI/ML"},                                          "compute"),
    ("DB系リソース",           "subtype",   {"Database", "CacheDB"},                            "data"),
    ("オブジェクトストレージ",  "subtype",   {"ObjectStorage"},                                  "data"),
    ("プラットフォーム",        "card_type", {"Platform"},                                       "support"),
    ("アタッチメント",          "card_type", {"Attachment"},                                     "support"),
    ("ストラテジー",            "card_type", {"Strategy"},                                       "support"),
    ("インシデント",            "card_type", {"Incident"},                                       "support"),
    ("リアクティブ",            "card_type", {"Reactive"},                                       "support"),
    ("ログ",                   "card_type", {"Log"},                                            "support"),
]


# ─── Load ───────────────────────────────────────────────
def load_all_cards():
    """Load all YAML files and return (all_cards, faction_data)."""
    all_cards = []
    faction_data = {}

    for faction in FACTION_ORDER:
        file_stem = FACTION_FILE_MAP.get(faction, faction.lower())
        filepath = YAML_DIR / f"{file_stem}.yaml"
        if not filepath.exists():
            print(f"WARNING: {filepath} not found, skipping", file=sys.stderr)
            continue

        with open(filepath, "r", encoding="utf-8") as f:
            data = yaml.safe_load(f)

        faction_name = data.get("faction", faction)
        display_name = data.get("display_name", faction_name)
        cards = data.get("cards", [])

        for card in cards:
            card.setdefault("faction", faction_name)

        faction_data[faction_name] = {
            "display_name": display_name,
            "cards": cards,
            "restriction_cards": data.get("restriction_cards", []),
        }
        all_cards.extend(cards)

    return all_cards, faction_data


# ─── Game design constants (common SSoT) ───────────────
class CardGenerationError(Exception):
    """Raised when card generation cannot proceed because its inputs are malformed."""


@dataclass(frozen=True)
class EffectTaxonomy:
    """Canonical card_type / subtype values and the resource groups that expand into them."""

    card_types: frozenset
    subtypes: frozenset
    resource_groups: dict


def resolve_common_constants_path():
    """Locate the game design constants owned by common.

    Returns:
        Path to game_design_constants.yaml, taken from the COMMON_REPO environment
        variable when set and from the sibling checkout of common otherwise.
    """
    repo_root = os.environ.get(COMMON_REPO_ENV)
    base = Path(repo_root) if repo_root else DEFAULT_COMMON_REPO
    return base / GAME_DESIGN_CONSTANTS_RELPATH


def load_effect_taxonomy(constants_path):
    """Load the canonical card_type / subtype values and the resource groups from common.

    Args:
        constants_path: Path to common's game_design_constants.yaml.

    Returns:
        EffectTaxonomy built from the file.

    Raises:
        CardGenerationError: When the file is missing or lacks a required key.
    """
    path = Path(constants_path)
    if not path.exists():
        raise CardGenerationError(
            f"game design constants not found at {path}. "
            f"Set {COMMON_REPO_ENV} to the overload-party-common checkout."
        )

    with open(path, "r", encoding="utf-8") as f:
        constants = yaml.safe_load(f)

    for key in (CARD_TYPES_KEY, SUBTYPES_KEY, RESOURCE_GROUPS_KEY):
        if key not in constants:
            raise CardGenerationError(f"game design constants at {path} is missing '{key}'")

    subtypes = frozenset(s for values in constants[SUBTYPES_KEY].values() for s in values)
    return EffectTaxonomy(
        card_types=frozenset(constants[CARD_TYPES_KEY]),
        subtypes=subtypes,
        resource_groups=constants[RESOURCE_GROUPS_KEY],
    )


# ─── Expand resource groups ────────────────────────────
def expand_card_resource_groups(cards, resource_groups):
    """Replace every card's effects with a tree whose resource groups are expanded.

    Args:
        cards: Cards to rewrite in place.
        resource_groups: Group name to its card_type / subtypes definition.

    Raises:
        CardGenerationError: When a card names an undefined group, combines a group
            with card_type / subtype, or the group definition is malformed.
    """
    for card in cards:
        if not card.get("effects"):
            continue
        label = f"{card.get('card_id', '???')} {card.get('card_name', '???')}"
        card["effects"] = _expand_resource_groups(card["effects"], resource_groups, label)


def _expand_resource_groups(node, resource_groups, label):
    if isinstance(node, list):
        return [_expand_resource_groups(item, resource_groups, label) for item in node]
    if not isinstance(node, dict):
        return node

    expanded = {}
    for key, value in node.items():
        if key == EFFECT_GROUP_KEY:
            expanded.update(_resolve_resource_group(value, resource_groups, node, label))
        else:
            expanded[key] = _expand_resource_groups(value, resource_groups, label)
    return expanded


def _resolve_resource_group(name, resource_groups, node, label):
    definition = resource_groups.get(name) if isinstance(name, str) else None
    if definition is None:
        raise CardGenerationError(f"{label}: unknown resource group '{name}'")

    combined = [key for key in (EFFECT_CARD_TYPE_KEY, EFFECT_SUBTYPE_KEY) if key in node]
    if combined:
        raise CardGenerationError(
            f"{label}: resource group '{name}' cannot be combined with {', '.join(combined)}"
        )

    defined = [key for key in (EFFECT_CARD_TYPE_KEY, SUBTYPES_KEY) if key in definition]
    if len(defined) != 1:
        raise CardGenerationError(
            f"{label}: resource group '{name}' must define exactly one of "
            f"'{EFFECT_CARD_TYPE_KEY}' / '{SUBTYPES_KEY}'"
        )

    if defined[0] == EFFECT_CARD_TYPE_KEY:
        return {EFFECT_CARD_TYPE_KEY: definition[EFFECT_CARD_TYPE_KEY]}
    return {EFFECT_SUBTYPE_KEY: list(definition[SUBTYPES_KEY])}


# ─── Validate ──────────────────────────────────────────
def _validate_required_fields(card, label, errors):
    is_log = card.get("card_type") == CARD_TYPE_LOG
    if is_log:
        required = ["card_no", "card_id", "card_name", "const_name", "card_type", "restriction", "is_active"]
    else:
        required = ["card_no", "card_id", "card_name", "const_name", "card_type", "resizable", "elastic", "restriction", "is_active", "stats"]
    for field in required:
        if field not in card:
            errors.append(f"{label}: missing required field '{field}'")
    # subtype は Compute/Data カテゴリで必須、それ以外は禁止
    card_type = card.get("card_type")
    has_subtype = "subtype" in card
    if card_type in SUBTYPES_BY_CARD_TYPE and not has_subtype:
        errors.append(f"{label}: card_type '{card_type}' requires 'subtype' field")
    if card_type not in SUBTYPES_BY_CARD_TYPE and has_subtype:
        errors.append(f"{label}: card_type '{card_type}' must not have 'subtype' field")


def _validate_uniqueness(card, label, seen_nos, seen_consts, seen_card_ids, errors):
    card_no = card.get("card_no")
    card_name = card.get("card_name", "???")

    if card_no in seen_nos:
        errors.append(f"{label}: duplicate card_no (also used by {seen_nos[card_no]})")
    else:
        seen_nos[card_no] = card_name

    card_id = card.get("card_id", "")
    if card_id:
        if not re.match(r"^(SH|TK|SL|TN|NT)-\d{4}$", card_id):
            errors.append(f"{label}: card_id '{card_id}' must match XX-NNNN format (SH/TK/SL/TN/NT)")
        if card_id in seen_card_ids:
            errors.append(f"{label}: duplicate card_id '{card_id}' (also used by #{seen_card_ids[card_id]})")
        else:
            seen_card_ids[card_id] = card_no

    const_name = card.get("const_name", "")
    if const_name:
        if const_name in seen_consts:
            errors.append(f"{label}: duplicate const_name '{const_name}' (also used by #{seen_consts[const_name]})")
        else:
            seen_consts[const_name] = card_no
        if not re.match(r"^[A-Z][a-zA-Z0-9]*$", const_name):
            errors.append(f"{label}: const_name '{const_name}' is not a valid Go identifier (must be ASCII PascalCase)")


def _validate_types_and_values(card, label, errors):
    card_type = card.get("card_type", "")
    if card_type and card_type not in ALL_CARD_TYPES:
        errors.append(f"{label}: invalid card_type '{card_type}'")
    subtype = card.get("subtype")
    if subtype is not None:
        allowed = SUBTYPES_BY_CARD_TYPE.get(card_type, set())
        if subtype not in allowed:
            errors.append(f"{label}: invalid subtype '{subtype}' for card_type '{card_type}'")

    for bf in ("resizable", "elastic"):
        val = card.get(bf)
        if val is not None and not isinstance(val, bool):
            errors.append(f"{label}: '{bf}' must be a boolean, got {type(val).__name__}")

    restriction = card.get("restriction", "")
    if restriction and restriction not in VALID_RESTRICTION:
        errors.append(f"{label}: invalid restriction '{restriction}'")

    faction = card.get("faction", "")
    if faction and faction not in VALID_FACTIONS:
        errors.append(f"{label}: invalid faction '{faction}'")

    is_active = card.get("is_active")
    if is_active is not None and not isinstance(is_active, bool):
        errors.append(f"{label}: 'is_active' must be a boolean, got {type(is_active).__name__}")

    card_no = card.get("card_no")
    if card_no is not None:
        if not isinstance(card_no, int) or card_no <= 0:
            errors.append(f"{label}: card_no must be a positive integer, got {card_no}")

    deploy_turns = card.get("deploy_turns")
    if deploy_turns is not None:
        if not isinstance(deploy_turns, int) or deploy_turns not in (0, 1, 2):
            errors.append(f"{label}: deploy_turns must be 0, 1, or 2, got {deploy_turns}")


def _validate_stats(card, label, errors):
    card_type = card.get("card_type", "")
    stats = card.get("stats", {})
    if not (card_type and stats):
        return

    if card_type == CARD_TYPE_COMPUTE:
        missing = COMPUTE_STAT_KEYS - set(stats.keys())
        if missing:
            errors.append(f"{label}: compute card missing stats: {missing}")
    elif card_type == CARD_TYPE_DATA:
        missing = DATA_STAT_KEYS - set(stats.keys())
        if missing:
            errors.append(f"{label}: data card missing stats: {missing}")

    for stat_key, stat_val in stats.items():
        if not isinstance(stat_val, (int, float)):
            errors.append(f"{label}: stat '{stat_key}' must be a number, got {type(stat_val).__name__}")
        elif stat_val < 0:
            errors.append(f"{label}: stat '{stat_key}' must be non-negative, got {stat_val}")


def _validate_elastic(card, label, errors):
    if not card.get("elastic"):
        return

    for ef in ("free_tier", "cost_per_request"):
        if ef not in card:
            errors.append(f"{label}: elastic card missing required field '{ef}'")

    for ef in ("elastic_increment", "free_tier", "cost_per_request"):
        val = card.get(ef)
        if val is not None and (not isinstance(val, (int, float)) or val < 0):
            errors.append(f"{label}: '{ef}' must be a non-negative number, got {val}")


def validate(cards):
    """Run all validation checks. Returns list of error strings."""
    errors = []
    seen_nos = {}
    seen_consts = {}
    seen_card_ids = {}

    for card in cards:
        card_no = card.get("card_no")
        card_name = card.get("card_name", "???")
        label = f"#{card_no} {card_name}"

        _validate_required_fields(card, label, errors)
        if card_no is None:
            continue
        _validate_uniqueness(card, label, seen_nos, seen_consts, seen_card_ids, errors)
        _validate_types_and_values(card, label, errors)
        _validate_stats(card, label, errors)
        _validate_elastic(card, label, errors)

    return errors


def validate_effect_taxonomy(cards, taxonomy):
    """Check that effect definitions only name canonical card_type / subtype values.

    Args:
        cards: Cards whose resource groups are already expanded.
        taxonomy: Canonical values owned by common.

    Returns:
        List of error strings, each naming the card and the offending value.
    """
    errors = []
    allowed = {EFFECT_CARD_TYPE_KEY: taxonomy.card_types, EFFECT_SUBTYPE_KEY: taxonomy.subtypes}
    for card in cards:
        if not card.get("effects"):
            continue
        label = f"{card.get('card_id', '???')} {card.get('card_name', '???')}"
        _collect_effect_taxonomy_errors(card["effects"], allowed, label, errors)
    return errors


def _collect_effect_taxonomy_errors(node, allowed, label, errors):
    if isinstance(node, list):
        for item in node:
            _collect_effect_taxonomy_errors(item, allowed, label, errors)
        return
    if not isinstance(node, dict):
        return

    for key, value in node.items():
        if key in allowed:
            # 空で書くと engine が絞り込みなしとして扱い、効果が黙って全てのカードに及ぶ
            if isinstance(value, list) and not value:
                errors.append(f"{label}: effect '{key}' is empty")
            for name in value if isinstance(value, list) else [value]:
                if name not in allowed[key]:
                    errors.append(f"{label}: effect '{key}' has invalid value '{name}'")
        _collect_effect_taxonomy_errors(value, allowed, label, errors)


# ─── Generate JSON ─────────────────────────────────────
def generate_json(cards, *, out_path):
    """Generate cards_gen.json."""
    json_out = Path(out_path)
    output = []
    for card in sorted(cards, key=lambda c: c["card_id"]):
        stats = dict(card["stats"].items()) if card.get("stats") else {}
        deploy_turns = card.get("deploy_turns", 0)
        entry = {
            "card_id": card["card_id"],
            "card_name": card["card_name"],
            "resource_label": card.get("resource_label", ""),
            "faction": card["faction"],
            "card_type": card["card_type"],
            "deploy_turns": deploy_turns,
            "resizable": card.get("resizable", False),
            "elastic": card.get("elastic", False),
            "elastic_increment": card.get("elastic_increment", 0),
            "free_tier": card.get("free_tier", 0),
            "cost_per_request": card.get("cost_per_request", 0),
            "stats": stats,
            "restriction": card["restriction"],
            "is_active": card["is_active"],
        }
        if card.get("subtype") is not None:
            entry["subtype"] = card["subtype"]
        if card.get("effect_text"):
            entry["effect_text"] = card["effect_text"]
        if card.get("effects"):
            entry["effects"] = card["effects"]
        output.append(entry)

    json_out.parent.mkdir(parents=True, exist_ok=True)
    with open(json_out, "w", encoding="utf-8") as f:
        json.dump(output, f, ensure_ascii=False, indent=2)
        f.write("\n")

    return len(output)


# ─── Generate Seed SQL ─────────────────────────────────
def _sql_escape(s):
    """Escape a string for PostgreSQL single-quoted literal."""
    if s is None:
        return "NULL"
    return "'" + s.replace("'", "''") + "'"


def generate_seed_sql(cards, *, out_path):
    """Generate db/seed/cards_seed.sql with UPSERT statements."""
    sql_out = Path(out_path)
    sorted_cards = sorted(cards, key=lambda c: c["card_id"])

    lines = []
    lines.append("-- Auto-generated by scripts/generate_cards.py. DO NOT EDIT.")
    lines.append("")
    lines.append("INSERT INTO card.card_definitions")
    lines.append("  (card_id, card_name, resource_label, faction, card_type, subtype,")
    lines.append("   resizable, elastic, stats, effect_text, effects, restriction, is_active)")
    lines.append("VALUES")

    value_rows = []
    for card in sorted_cards:
        card_id = _sql_escape(card["card_id"])
        card_name = _sql_escape(card["card_name"])
        resource_label = _sql_escape(card.get("resource_label", ""))
        faction = _sql_escape(card["faction"])
        card_type = _sql_escape(card["card_type"])
        subtype = _sql_escape(card["subtype"]) if card.get("subtype") is not None else "NULL"
        resizable = "true" if card.get("resizable", False) else "false"
        elastic = "true" if card.get("elastic", False) else "false"

        stats = card.get("stats")
        stats_json = _sql_escape(json.dumps(stats, ensure_ascii=False)) + "::jsonb" if stats else "'{}'::jsonb"

        effect_text = _sql_escape(card.get("effect_text")) if card.get("effect_text") and card["effect_text"] != "—" else "NULL"

        effects = card.get("effects")
        effects_json = _sql_escape(json.dumps(effects, ensure_ascii=False)) + "::jsonb" if effects else "NULL"

        restriction = _sql_escape(card["restriction"])
        is_active = "true" if card["is_active"] else "false"

        row = (
            f"  ({card_id}, {card_name}, {resource_label}, {faction}, {card_type}, {subtype},\n"
            f"   {resizable}, {elastic}, {stats_json},\n"
            f"   {effect_text},\n"
            f"   {effects_json},\n"
            f"   {restriction}, {is_active})"
        )
        value_rows.append(row)

    lines.append(",\n\n".join(value_rows))
    lines.append("")
    lines.append("ON CONFLICT (card_id) DO UPDATE SET")
    lines.append("  card_name      = EXCLUDED.card_name,")
    lines.append("  resource_label = EXCLUDED.resource_label,")
    lines.append("  faction        = EXCLUDED.faction,")
    lines.append("  card_type      = EXCLUDED.card_type,")
    lines.append("  subtype        = EXCLUDED.subtype,")
    lines.append("  resizable      = EXCLUDED.resizable,")
    lines.append("  elastic        = EXCLUDED.elastic,")
    lines.append("  stats          = EXCLUDED.stats,")
    lines.append("  effect_text    = EXCLUDED.effect_text,")
    lines.append("  effects        = EXCLUDED.effects,")
    lines.append("  restriction    = EXCLUDED.restriction,")
    lines.append("  is_active      = EXCLUDED.is_active,")
    lines.append("  updated_at     = now();")
    lines.append("")

    sql_out.parent.mkdir(parents=True, exist_ok=True)
    with open(sql_out, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
        f.write("\n")

    return len(sorted_cards)


# ─── Generate CARDS.md ─────────────────────────────────
def _scalability_display(card):
    # Subtype がある場合（Compute/Data 配下）は subtype を、なければ card_type を表示
    label = card.get("subtype") or card["card_type"]
    display_type = {"ObjectStorage": "Object Storage", "CacheDB": "Cache DB"}.get(label, label)
    r = card.get("resizable", False)
    e = card.get("elastic", False)
    if r and e:
        return f"{display_type} (R+E)"
    elif r:
        return f"{display_type} (R)"
    elif e:
        return f"{display_type} (E)"
    return display_type


def _effect_display(card):
    text = card.get("effect_text", "")
    if not text:
        return "—"
    return text.replace("\n", "<br>")


def _md_resource_table(cards, primary_stat_label, primary_stat_key):
    """Generate a Markdown table for compute or data resource cards."""
    lines = []
    lines.append(f"| ID | カード名 | タイプ | {primary_stat_label} | 可用性 | 維持コスト | デプロイT | SLAペナルティ | 効果 |")
    lines.append("|------|---------|-------|-----|-----|-----|-----|-----|------|")
    for c in cards:
        dt = c.get("deploy_turns", 0)
        lines.append(
            f"| {c['card_id']} | {c['card_name']} | {_scalability_display(c)} "
            f"| {c['stats'].get(primary_stat_key, 0)} | {c['stats']['availability']} "
            f"| {c['stats']['maintenance_cost']} | {dt} "
            f"| {c['stats']['sla_penalty']} | {_effect_display(c)} |"
        )
    return lines


def _md_support_table(cards):
    """Generate a Markdown table for support/log cards."""
    lines = []
    lines.append("| ID | カード名 | 効果 |")
    lines.append("|------|---------|------|")
    for c in cards:
        lines.append(f"| {c['card_id']} | {c['card_name']} | {_effect_display(c)} |")
    return lines


def _md_restriction_section(faction_data, lines):
    """Append restriction card tables (limited / semi-limited)."""
    limited = []
    semi_limited = []
    for f in FACTION_ORDER:
        if f not in faction_data:
            continue
        fd = faction_data[f]
        for c in fd["cards"]:
            if c.get("restriction") == "limited":
                limited.append(c)
            elif c.get("restriction") == "semi_limited":
                semi_limited.append(c)

    if limited:
        lines.append("### 制限カード（1枚制限）")
        lines.append("")
        lines.append("| カード名 | 理由 |")
        lines.append("|---------|------|")
        for card in limited:
            lines.append(f"| {card['card_name']} | {card.get('restriction_reason', '')} |")
        lines.append("")

    if semi_limited:
        lines.append("### 準制限カード（2枚制限）")
        lines.append("")
        lines.append("| カード名 | 理由 |")
        lines.append("|---------|------|")
        for card in semi_limited:
            lines.append(f"| {card['card_name']} | {card.get('restriction_reason', '')} |")
        lines.append("")

    lines.append("**上記以外のカードはすべて 3枚まで投入可能。**")
    lines.append("")
    lines.append("---")
    lines.append("")


def _md_faction_sections(faction_data, lines):
    """Append per-faction card tables. Returns summary dict for the summary table."""
    summary = {}

    for faction in FACTION_ORDER:
        if faction not in faction_data:
            continue
        fd = faction_data[faction]
        faction_cards = fd["cards"]
        display = fd["display_name"]

        lines.append(f"## {display}— {len(faction_cards)}枚")
        lines.append("")

        summary[faction] = {}

        for cat_name, by_field, allowed_values, table_kind in CATEGORY_ORDER:
            cat_cards = sorted(
                [c for c in faction_cards if c.get(by_field) in allowed_values],
                key=lambda c: c["card_id"]
            )
            summary[faction][cat_name] = len(cat_cards)
            if not cat_cards:
                continue

            lines.append(f"### {cat_name}（{len(cat_cards)}枚）")
            lines.append("")

            if table_kind == "compute":
                lines.extend(_md_resource_table(cat_cards, "スループット", "throughput"))
            elif table_kind == "data":
                lines.extend(_md_resource_table(cat_cards, "Yield", "yield"))
            else:
                lines.extend(_md_support_table(cat_cards))

            lines.append("")

        lines.append("---")
        lines.append("")

    return summary


def _md_summary_table(faction_data, summary, lines):
    """Append the card count summary table."""
    lines.append("## カード総数サマリ")
    lines.append("")
    header_names = [faction_data[f]["display_name"] if f in faction_data else f for f in FACTION_ORDER]
    lines.append("| カテゴリ | " + " | ".join(header_names) + " | 合計 |")
    lines.append("|---------|" + "|".join(["------" for _ in FACTION_ORDER]) + "|------|")

    grand_total = {f: 0 for f in FACTION_ORDER}
    for cat_name, *_ in CATEGORY_ORDER:
        row = f"| {cat_name} "
        cat_total = 0
        for faction in FACTION_ORDER:
            count = summary.get(faction, {}).get(cat_name, 0)
            grand_total[faction] += count
            cat_total += count
            row += f"| {count} "
        row += f"| {cat_total} |"
        lines.append(row)

    row = "| **合計** "
    all_total = 0
    for faction in FACTION_ORDER:
        row += f"| **{grand_total[faction]}** "
        all_total += grand_total[faction]
    row += f"| **{all_total}** |"
    lines.append(row)
    lines.append("")


def generate_md(cards, faction_data, *, out_path):
    """Generate docs/CARDS.md."""
    md_out = Path(out_path)
    total = len(cards)
    lines = []

    lines.append("<!-- This file is auto-generated by scripts/generate_cards.py. DO NOT EDIT. -->")
    lines.append("")
    lines.append("# Overload Party — Card List")
    lines.append("")
    lines.append(f"**全 {total} 枚**")
    lines.append("")
    lines.append("---")
    lines.append("")

    _md_restriction_section(faction_data, lines)
    summary = _md_faction_sections(faction_data, lines)
    _md_summary_table(faction_data, summary, lines)

    md_out.parent.mkdir(parents=True, exist_ok=True)
    with open(md_out, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
        f.write("\n")

    return total


# ─── Starter Decks ────────────────────────────────────
def load_starter_decks():
    if not STARTER_DECKS_YAML.exists():
        return None
    with open(STARTER_DECKS_YAML, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f)
    return data.get("starter_decks", [])


def validate_starter_decks(decks, valid_card_ids):
    errors = []
    for deck in decks:
        name = deck.get("deck_name", "???")
        cards = deck.get("cards", [])
        total = sum(c.get("copies", 1) for c in cards)
        if total != 30:
            errors.append(f"deck '{name}': has {total} cards, expected 30")
        for entry in cards:
            cid = entry.get("card_id", "")
            if cid not in valid_card_ids:
                errors.append(f"deck '{name}': unknown card_id '{cid}'")
    return errors


def generate_starter_decks_json(decks, *, out_path):
    output = []
    for deck in decks:
        cards = []
        for entry in deck["cards"]:
            for _ in range(entry.get("copies", 1)):
                cards.append(entry["card_id"])
        output.append({"deck_name": deck["deck_name"], "cards": cards})

    out = Path(out_path)
    out.parent.mkdir(parents=True, exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        json.dump(output, f, ensure_ascii=False, indent=2)
        f.write("\n")
    return len(output)


# ─── Main ──────────────────────────────────────────────
def main():
    if not YAML_DIR.exists():
        print(f"ERROR: YAML directory not found: {YAML_DIR}", file=sys.stderr)
        sys.exit(1)

    cards, faction_data = load_all_cards()
    if not cards:
        print("ERROR: No cards loaded from YAML files", file=sys.stderr)
        sys.exit(1)

    try:
        taxonomy = load_effect_taxonomy(resolve_common_constants_path())
        expand_card_resource_groups(cards, taxonomy.resource_groups)
    except CardGenerationError as err:
        print(f"ERROR: {err}", file=sys.stderr)
        sys.exit(1)

    errors = validate(cards) + validate_effect_taxonomy(cards, taxonomy)
    if errors:
        print(f"Validation failed with {len(errors)} error(s):", file=sys.stderr)
        for err in errors:
            print(f"  - {err}", file=sys.stderr)
        sys.exit(1)

    # cards_gen.json (Go embedded)
    go_count = generate_json(cards, out_path=GO_JSON_OUT)
    print(f"Generated {go_count} cards → {GO_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    # CARDS.md
    md_count = generate_md(cards, faction_data, out_path=MD_OUT)
    print(f"Generated {md_count} cards → {MD_OUT.relative_to(ROOT)}", file=sys.stderr)

    # Seed SQL
    seed_count = generate_seed_sql(cards, out_path=SEED_OUT)
    print(f"Generated {seed_count} cards → {SEED_OUT.relative_to(ROOT)}", file=sys.stderr)

    # Starter Decks
    starter_decks = load_starter_decks()
    if starter_decks:
        valid_card_ids = {c["card_id"] for c in cards}
        deck_errors = validate_starter_decks(starter_decks, valid_card_ids)
        if deck_errors:
            print(f"Starter deck validation failed with {len(deck_errors)} error(s):", file=sys.stderr)
            for err in deck_errors:
                print(f"  - {err}", file=sys.stderr)
            sys.exit(1)

        deck_count = generate_starter_decks_json(starter_decks, out_path=STARTER_DECKS_JSON_OUT)
        print(f"Generated {deck_count} starter decks → {STARTER_DECKS_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)


if __name__ == "__main__":
    main()
