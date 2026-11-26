#!/usr/bin/env python3
"""YAML のカード定義から、配布用 JSON・カード一覧文書・初期投入 SQL を生成する。

実行には、ゲーム定数を持つ overload-party-common の場所を環境変数 COMMON_REPO で、
ゲームロジック定数を持つ overload-party-battle の場所を環境変数 BATTLE_REPO で渡す。

    COMMON_REPO=../overload-party-common BATTLE_REPO=../overload-party-battle \\
        python3 scripts/generate_cards.py
"""

import os
import re
import sys
from dataclasses import dataclass
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
YAML_DIR = ROOT / "data" / "cards"
MD_OUT = ROOT / "docs" / "CARDS.md"
SEED_OUT = ROOT / "db" / "seed" / "cards_seed.sql"
GO_JSON_OUT = ROOT / "data" / "cache" / "cards_gen.json"

COMMON_REPO_ENV = "COMMON_REPO"
COMMON_REPO_NAME = "overload-party-common"
GAME_DESIGN_CONSTANTS_RELPATH = Path("data") / "game_design_constants.yaml"
GAME_DESIGN_CONSTANTS_NAME = "game design constants"
BATTLE_REPO_ENV = "BATTLE_REPO"
BATTLE_REPO_NAME = "overload-party-battle"
GAME_LOGIC_CONSTANTS_RELPATH = Path("data") / "game_logic_constants.yaml"
GAME_LOGIC_CONSTANTS_NAME = "game logic constants"

CARD_TYPE_COMPUTE = "Compute"
CARD_TYPE_DATA = "DataResource"
# ログカードは用語集にあるカード種別だが、共通のゲーム定数の card_types に無いため、カード定義の種別としてここで加える
CARD_TYPE_LOG = "Log"

EFFECT_GROUP_KEY = "group"
EFFECT_CARD_TYPE_KEY = "card_type"
EFFECT_SUBTYPE_KEY = "subtype"
SUBTYPES_KEY = "subtypes"
RESOURCE_GROUPS_KEY = "resource_groups"
CARD_TYPES_KEY = "card_types"
TRIGGER_TYPES_KEY = "trigger_types"
EFFECT_OPS_KEY = "effect_ops"
BUFF_TYPES_KEY = "buff_types"
EFFECT_DURATIONS_KEY = "effect_durations"
SELECTOR_KEYWORDS_KEY = "selector_keywords"
PLAYER_REFS_KEY = "player_refs"

EFFECT_TRIGGER_KEY = "trigger"
EFFECT_OP_LIST_KEY = "ops"
EFFECT_CHOICE_KEY = "choice"
CHOICE_USE_KEY = "use"
OP_ITEM = "op"
OP_BUFF_KEY = "buff"
OP_DURATION_KEY = "duration"
OP_SELECTOR_KEY = "selector"
SELECTOR_OWNER_KEY = "owner"

VALID_RESTRICTION = {"unlimited", "limited", "semi_limited", "forbidden"}
FACTION_ORDER = ["SHE", "Tenki", "Sugar", "Tuners", "Neutral"]
VALID_FACTIONS = frozenset(FACTION_ORDER)
FACTION_FILE_MAP = {"SHE": "sd"}
FACTION_FILE_REQUIRED_KEYS = ("faction", "display_name", "cards")

REQUIRED_CARD_FIELDS = [
    "card_no", "card_id", "card_name", "const_name", "card_type",
    "resizable", "elastic", "restriction", "is_active", "stats",
]
REQUIRED_LOG_CARD_FIELDS = [
    "card_no", "card_id", "card_name", "const_name", "card_type", "restriction", "is_active",
]
VALID_DEPLOY_TURNS = (0, 1, 2)
COMPUTE_STAT_KEYS = {"throughput", "availability", "maintenance_cost", "sla_penalty"}
DATA_STAT_KEYS = {"yield", "availability", "maintenance_cost", "sla_penalty"}
ALLOWED_STAT_KEYS = frozenset({
    "throughput", "throughput_max", "yield", "yield_max",
    "availability", "maintenance_cost", "sla_penalty",
})

OPTIONAL_CARD_FIELD_DEFAULTS = {
    "deploy_turns": 0,
    "resource_label": "",
    "elastic_increment": 0,
    "free_tier": 0,
    "cost_per_request": 0,
    "resizable": False,
    "elastic": False,
}
NO_EFFECT_TEXT_MARK = "—"

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
    ("ログカード",             "card_type",{CARD_TYPE_LOG},                                    "support"),
]


class CardGenerationError(Exception):
    """入力が誤っていて、カードの生成を続けられないことを表す。"""


@dataclass(frozen=True)
class EffectTaxonomy:
    """効果に書ける値と、リソースの括り。"""

    card_types: frozenset
    subtypes_by_card_type: dict
    resource_groups: dict
    triggers: frozenset
    ops: frozenset
    buffs: frozenset
    durations: frozenset
    selector_keywords: frozenset
    selector_owners: frozenset

    @property
    def subtypes(self):
        """全カードタイプの subtype をまとめた集合。"""
        return frozenset(
            subtype for subtypes in self.subtypes_by_card_type.values() for subtype in subtypes
        )


def load_all_cards():
    """陣営ごとの定義ファイルを読み込み、全カードと陣営ごとの情報を返す。

    Returns:
        全カードのリストと、陣営名から表示名とカードのリストへの辞書の組。

    Raises:
        CardGenerationError: 陣営の定義ファイルが無い、必須項目が無い、
            または項目の陣営がファイルの陣営と合わないとき。
    """
    all_cards = []
    faction_data = {}

    for faction in FACTION_ORDER:
        data = _load_faction_file(faction)
        cards = data["cards"]
        for card in cards:
            card.setdefault("faction", faction)
        faction_data[faction] = {"display_name": data["display_name"], "cards": cards}
        all_cards.extend(cards)

    return all_cards, faction_data


def _load_faction_file(faction):
    path = YAML_DIR / f"{FACTION_FILE_MAP.get(faction, faction.lower())}.yaml"
    if not path.exists():
        raise CardGenerationError(f"faction definition file not found: {path}")

    data = load_yaml_file(path)
    for key in FACTION_FILE_REQUIRED_KEYS:
        if key not in data:
            raise CardGenerationError(f"{path}: missing required field '{key}'")
    if data["faction"] != faction:
        raise CardGenerationError(
            f"{path}: faction '{data['faction']}' does not match the faction '{faction}' of this file"
        )
    return data


def resolve_common_constants_path():
    """common が持つゲーム定数のファイルの場所を返す。

    Raises:
        CardGenerationError: 環境変数 COMMON_REPO が設定されていないとき。
    """
    return _resolve_constants_path(COMMON_REPO_ENV, COMMON_REPO_NAME, GAME_DESIGN_CONSTANTS_RELPATH)


def resolve_battle_constants_path():
    """battle が持つゲームロジック定数のファイルの場所を返す。

    Raises:
        CardGenerationError: 環境変数 BATTLE_REPO が設定されていないとき。
    """
    return _resolve_constants_path(BATTLE_REPO_ENV, BATTLE_REPO_NAME, GAME_LOGIC_CONSTANTS_RELPATH)


def _resolve_constants_path(repo_env, repo_name, relpath):
    repo_root = os.environ.get(repo_env)
    if not repo_root:
        raise CardGenerationError(
            f"{repo_env} is not set. Set {repo_env} to the {repo_name} checkout."
        )
    return Path(repo_root) / relpath


def load_effect_taxonomy(design_constants_path, logic_constants_path):
    """ゲーム定数とゲームロジック定数から、効果に書ける値とリソースの括りを読み込む。

    Args:
        design_constants_path: common の game_design_constants.yaml の場所。
        logic_constants_path: battle の game_logic_constants.yaml の場所。

    Raises:
        CardGenerationError: ファイルが無い、または必須の項目が無いとき。
    """
    design = _load_constants_file(
        design_constants_path,
        GAME_DESIGN_CONSTANTS_NAME,
        COMMON_REPO_ENV,
        COMMON_REPO_NAME,
        (CARD_TYPES_KEY, SUBTYPES_KEY, RESOURCE_GROUPS_KEY),
    )
    logic = _load_constants_file(
        logic_constants_path,
        GAME_LOGIC_CONSTANTS_NAME,
        BATTLE_REPO_ENV,
        BATTLE_REPO_NAME,
        (
            TRIGGER_TYPES_KEY,
            EFFECT_OPS_KEY,
            BUFF_TYPES_KEY,
            EFFECT_DURATIONS_KEY,
            SELECTOR_KEYWORDS_KEY,
            PLAYER_REFS_KEY,
        ),
    )

    return EffectTaxonomy(
        card_types=frozenset(design[CARD_TYPES_KEY]),
        subtypes_by_card_type={
            card_type: frozenset(subtypes)
            for card_type, subtypes in design[SUBTYPES_KEY].items()
        },
        resource_groups=design[RESOURCE_GROUPS_KEY],
        triggers=frozenset(logic[TRIGGER_TYPES_KEY]),
        ops=frozenset(logic[EFFECT_OPS_KEY]),
        buffs=frozenset(logic[BUFF_TYPES_KEY]),
        durations=frozenset(logic[EFFECT_DURATIONS_KEY]),
        selector_keywords=frozenset(logic[SELECTOR_KEYWORDS_KEY]),
        selector_owners=frozenset(logic[PLAYER_REFS_KEY]),
    )


def _load_constants_file(constants_path, name, repo_env, repo_name, required_keys):
    path = Path(constants_path)
    if not path.exists():
        raise CardGenerationError(
            f"{name} not found at {path}. Set {repo_env} to the {repo_name} checkout."
        )

    constants = load_yaml_file(path)

    for key in required_keys:
        if key not in constants:
            raise CardGenerationError(f"{name} at {path} is missing '{key}'")

    return constants


def expand_card_resource_groups(cards, resource_groups):
    """カードの効果に書かれたリソースの括りを、card_type または subtype に展開する。

    Args:
        cards: 展開するカード。effects を直接書き換える。
        resource_groups: 括り名から card_type または subtypes の定義への辞書。

    Raises:
        CardGenerationError: 未定義の括り、card_type や subtype との併記、
            または括りの定義の誤りがあるとき。
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


def _validate_required_fields(card, label, taxonomy, errors):
    card_type = card.get("card_type")
    required_fields = REQUIRED_LOG_CARD_FIELDS if card_type == CARD_TYPE_LOG else REQUIRED_CARD_FIELDS
    for field_name in required_fields:
        if field_name not in card:
            errors.append(f"{label}: missing required field '{field_name}'")


    has_subtype = "subtype" in card
    takes_subtype = card_type in taxonomy.subtypes_by_card_type
    if takes_subtype and not has_subtype:
        errors.append(f"{label}: card_type '{card_type}' requires 'subtype' field")
    if not takes_subtype and has_subtype:
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


def _validate_types_and_values(card, label, taxonomy, errors):
    card_type = card.get("card_type", "")
    if card_type and card_type not in taxonomy.card_types | {CARD_TYPE_LOG}:
        errors.append(f"{label}: invalid card_type '{card_type}'")
    subtype = card.get("subtype")
    if subtype is not None:
        allowed = taxonomy.subtypes_by_card_type.get(card_type, frozenset())
        if subtype not in allowed:
            errors.append(f"{label}: invalid subtype '{subtype}' for card_type '{card_type}'")

    for field_name in ("resizable", "elastic"):
        value = card.get(field_name)
        if value is not None and not isinstance(value, bool):
            errors.append(f"{label}: '{field_name}' must be a boolean, got {type(value).__name__}")

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
        if not isinstance(deploy_turns, int) or deploy_turns not in VALID_DEPLOY_TURNS:
            errors.append(f"{label}: deploy_turns must be 0, 1, or 2, got {deploy_turns}")


def _validate_stats(card, label, errors):
    if "stats" not in card:
        return

    card_type = card.get("card_type", "")
    stats = card["stats"] or {}

    if card_type == CARD_TYPE_COMPUTE:
        missing = COMPUTE_STAT_KEYS - set(stats.keys())
        if missing:
            errors.append(f"{label}: compute card missing stats: {missing}")
    elif card_type == CARD_TYPE_DATA:
        missing = DATA_STAT_KEYS - set(stats.keys())
        if missing:
            errors.append(f"{label}: data card missing stats: {missing}")

    for stat_key, stat_value in stats.items():
        if stat_key not in ALLOWED_STAT_KEYS:
            errors.append(f"{label}: stat '{stat_key}' is unknown")
        if not isinstance(stat_value, (int, float)):
            errors.append(f"{label}: stat '{stat_key}' must be a number, got {type(stat_value).__name__}")
        elif stat_value < 0:
            errors.append(f"{label}: stat '{stat_key}' must be non-negative, got {stat_value}")


def _validate_elastic(card, label, errors):
    if not card.get("elastic"):
        return

    for field_name in ("free_tier", "cost_per_request"):
        if field_name not in card:
            errors.append(f"{label}: elastic card missing required field '{field_name}'")

    for field_name in ("elastic_increment", "free_tier", "cost_per_request"):
        value = card.get(field_name)
        if value is not None and (not isinstance(value, (int, float)) or value < 0):
            errors.append(f"{label}: '{field_name}' must be a non-negative number, got {value}")


def validate_cards(cards, taxonomy):
    """カード定義を検証し、誤りの文字列のリストを返す。

    Args:
        cards: 検証するカード定義。
        taxonomy: カードタイプと subtype の正しい値。
    """
    errors = []
    seen_nos = {}
    seen_consts = {}
    seen_card_ids = {}

    for card in cards:
        card_no = card.get("card_no")
        card_name = card.get("card_name", "???")
        label = f"#{card_no} {card_name}"

        _validate_required_fields(card, label, taxonomy, errors)
        if card_no is None:
            continue
        _validate_uniqueness(card, label, seen_nos, seen_consts, seen_card_ids, errors)
        _validate_types_and_values(card, label, taxonomy, errors)
        _validate_stats(card, label, errors)
        _validate_elastic(card, label, errors)

    return errors


def validate_effect_taxonomy(cards, taxonomy):
    """効果に書かれた card_type・subtype・trigger・op・buff・selector・owner・duration が、定義された値かを検証する。

    Args:
        cards: リソースの括りを展開済みのカード。
        taxonomy: 効果に書ける値。

    Returns:
        カードと誤った値を示す誤りの文字列のリスト。
    """
    errors = []
    allowed = {EFFECT_CARD_TYPE_KEY: taxonomy.card_types, EFFECT_SUBTYPE_KEY: taxonomy.subtypes}
    for card in cards:
        if not card.get("effects"):
            continue
        label = f"{card.get('card_id', '???')} {card.get('card_name', '???')}"
        _collect_effect_taxonomy_errors(card["effects"], allowed, label, errors)
        for effect in card["effects"]:
            _collect_effect_logic_errors(effect, taxonomy, label, errors)
    return errors


def _collect_effect_logic_errors(effect, taxonomy, label, errors):
    if EFFECT_TRIGGER_KEY in effect:
        _validate_defined_value(
            label, EFFECT_TRIGGER_KEY, effect[EFFECT_TRIGGER_KEY], taxonomy.triggers, errors
        )

    ops = list(effect.get(EFFECT_OP_LIST_KEY, []))
    ops += effect.get(EFFECT_CHOICE_KEY, {}).get(CHOICE_USE_KEY, [])
    for op in ops:
        for op_name, params in op.items():
            _validate_defined_value(label, OP_ITEM, op_name, taxonomy.ops, errors)
            if isinstance(params, dict):
                _collect_op_parameter_errors(params, taxonomy, label, errors)


def _collect_op_parameter_errors(params, taxonomy, label, errors):
    if OP_BUFF_KEY in params:
        _validate_defined_value(label, OP_BUFF_KEY, params[OP_BUFF_KEY], taxonomy.buffs, errors)
    if OP_DURATION_KEY in params:
        _validate_defined_value(
            label, OP_DURATION_KEY, params[OP_DURATION_KEY], taxonomy.durations, errors
        )

    selector = params.get(OP_SELECTOR_KEY)
    if isinstance(selector, str):
        _validate_defined_value(label, OP_SELECTOR_KEY, selector, taxonomy.selector_keywords, errors)
    elif isinstance(selector, dict) and SELECTOR_OWNER_KEY in selector:
        _validate_defined_value(
            label, SELECTOR_OWNER_KEY, selector[SELECTOR_OWNER_KEY], taxonomy.selector_owners, errors
        )


def _collect_effect_taxonomy_errors(node, allowed, label, errors):
    if isinstance(node, list):
        for item in node:
            _collect_effect_taxonomy_errors(item, allowed, label, errors)
        return
    if not isinstance(node, dict):
        return

    for key, value in node.items():
        if key in allowed:
            # 空で書くと絞り込みなしとして扱われ、効果が黙って全てのカードに及ぶため、誤りにする
            if isinstance(value, list) and not value:
                errors.append(f"{label}: effect '{key}' is empty")
            for name in value if isinstance(value, list) else [value]:
                _validate_defined_value(label, key, name, allowed[key], errors)
        _collect_effect_taxonomy_errors(value, allowed, label, errors)


def _validate_defined_value(label, key, value, defined_values, errors):
    if value not in defined_values:
        errors.append(
            f"{label}: effect '{key}' has value '{value}' "
            "that is not defined in the game constants"
        )


def _read_optional_card_field(card, field_name):
    return card.get(field_name, OPTIONAL_CARD_FIELD_DEFAULTS[field_name])


def _sort_cards_by_id(cards):
    return sorted(cards, key=lambda card: card["card_id"])


def generate_cards_json(cards, *, out_path):
    """カードの配布用 JSON を card_id の順に書き出す。

    Args:
        cards: 検証済みのカード定義。
        out_path: 書き出す場所。

    Returns:
        書き出したカードの枚数。
    """
    output = []
    for card in _sort_cards_by_id(cards):
        entry = {
            "card_id": card["card_id"],
            "card_name": card["card_name"],
            "resource_label": _read_optional_card_field(card, "resource_label"),
            "faction": card["faction"],
            "card_type": card["card_type"],
            "deploy_turns": _read_optional_card_field(card, "deploy_turns"),
            "resizable": _read_optional_card_field(card, "resizable"),
            "elastic": _read_optional_card_field(card, "elastic"),
            "elastic_increment": _read_optional_card_field(card, "elastic_increment"),
            "free_tier": _read_optional_card_field(card, "free_tier"),
            "cost_per_request": _read_optional_card_field(card, "cost_per_request"),
            "stats": card.get("stats") or {},
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

    write_json_file(out_path, output)
    return len(output)


def generate_cards_seed_sql(cards, *, out_path):
    """カードの初期投入 SQL を card_id の順に書き出す。

    Args:
        cards: 検証済みのカード定義。
        out_path: 書き出す場所。

    Returns:
        書き出したカードの枚数。
    """
    sorted_cards = _sort_cards_by_id(cards)

    lines = []
    lines.append("-- Auto-generated by scripts/generate_cards.py. DO NOT EDIT.")
    lines.append("")
    lines.append("INSERT INTO card.card_definitions")
    lines.append("  (card_id, card_name, resource_label, faction, card_type, subtype,")
    lines.append("   resizable, elastic, stats, effect_text, effects, restriction, is_active)")
    lines.append("VALUES")

    value_rows = []
    for card in sorted_cards:
        card_id = quote_sql_string(card["card_id"])
        card_name = quote_sql_string(card["card_name"])
        resource_label = quote_sql_string(_read_optional_card_field(card, "resource_label"))
        faction = quote_sql_string(card["faction"])
        card_type = quote_sql_string(card["card_type"])
        subtype = quote_sql_string(card["subtype"]) if card.get("subtype") is not None else "NULL"
        resizable = format_sql_boolean(_read_optional_card_field(card, "resizable"))
        elastic = format_sql_boolean(_read_optional_card_field(card, "elastic"))
        stats = quote_sql_jsonb(card.get("stats") or {})

        effect_text = card.get("effect_text")
        has_effect_text = effect_text and effect_text != NO_EFFECT_TEXT_MARK
        effect_text_sql = quote_sql_string(effect_text) if has_effect_text else "NULL"

        effects = card.get("effects")
        effects_sql = quote_sql_jsonb(effects) if effects else "NULL"

        restriction = quote_sql_string(card["restriction"])
        is_active = format_sql_boolean(card["is_active"])

        value_rows.append(
            f"  ({card_id}, {card_name}, {resource_label}, {faction}, {card_type}, {subtype},\n"
            f"   {resizable}, {elastic}, {stats},\n"
            f"   {effect_text_sql},\n"
            f"   {effects_sql},\n"
            f"   {restriction}, {is_active})"
        )

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

    write_lines_file(out_path, lines)
    return len(sorted_cards)


def _format_type_cell(card):
    label = card.get("subtype") or card["card_type"]
    display_type = {"ObjectStorage": "Object Storage", "CacheDB": "Cache DB"}.get(label, label)
    if card["resizable"] and card["elastic"]:
        return f"{display_type} (R+E)"
    if card["resizable"]:
        return f"{display_type} (R)"
    if card["elastic"]:
        return f"{display_type} (E)"
    return display_type


def _format_effect_cell(card):
    text = card.get("effect_text", "")
    if not text:
        return NO_EFFECT_TEXT_MARK
    return text.replace("\n", "<br>")


def _build_resource_table_lines(cards, primary_stat_label, primary_stat_key):
    lines = []
    lines.append(f"| ID | カード名 | タイプ | {primary_stat_label} | 可用性 | 維持コスト | デプロイターン | SLAペナルティ | 効果 |")
    lines.append("|------|---------|-------|-----|-----|-----|-----|-----|------|")
    for card in cards:
        lines.append(
            f"| {card['card_id']} | {card['card_name']} | {_format_type_cell(card)} "
            f"| {card['stats'][primary_stat_key]} | {card['stats']['availability']} "
            f"| {card['stats']['maintenance_cost']} | {_read_optional_card_field(card, 'deploy_turns')} "
            f"| {card['stats']['sla_penalty']} | {_format_effect_cell(card)} |"
        )
    return lines


def _build_support_table_lines(cards):
    lines = []
    lines.append("| ID | カード名 | 効果 |")
    lines.append("|------|---------|------|")
    for card in cards:
        lines.append(f"| {card['card_id']} | {card['card_name']} | {_format_effect_cell(card)} |")
    return lines


def _append_restriction_section(faction_data, lines):
    limited = []
    semi_limited = []
    for faction in FACTION_ORDER:
        if faction not in faction_data:
            continue
        for card in faction_data[faction]["cards"]:
            if card.get("restriction") == "limited":
                limited.append(card)
            elif card.get("restriction") == "semi_limited":
                semi_limited.append(card)

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


def _append_faction_sections(faction_data, lines):
    """陣営ごとのカード表を lines に追加し、陣営とカテゴリごとの枚数を返す。"""
    summary = {}

    for faction in FACTION_ORDER:
        if faction not in faction_data:
            continue
        faction_cards = faction_data[faction]["cards"]
        display_name = faction_data[faction]["display_name"]

        lines.append(f"## {display_name}（{len(faction_cards)}枚）")
        lines.append("")

        summary[faction] = {}

        for category_name, by_field, allowed_values, table_kind in CATEGORY_ORDER:
            category_cards = _sort_cards_by_id(
                [card for card in faction_cards if card.get(by_field) in allowed_values]
            )
            summary[faction][category_name] = len(category_cards)
            if not category_cards:
                continue

            lines.append(f"### {category_name}（{len(category_cards)}枚）")
            lines.append("")

            if table_kind == "compute":
                lines.extend(_build_resource_table_lines(category_cards, "スループット", "throughput"))
            elif table_kind == "data":
                lines.extend(_build_resource_table_lines(category_cards, "イールド", "yield"))
            else:
                lines.extend(_build_support_table_lines(category_cards))

            lines.append("")

        lines.append("---")
        lines.append("")

    return summary


def _append_summary_table(faction_data, summary, lines):
    lines.append("## カード総数サマリ")
    lines.append("")
    header_names = [
        faction_data[faction]["display_name"] if faction in faction_data else faction
        for faction in FACTION_ORDER
    ]
    lines.append("| カテゴリ | " + " | ".join(header_names) + " | 合計 |")
    lines.append("|---------|" + "|".join(["------" for _ in FACTION_ORDER]) + "|------|")

    grand_total = {faction: 0 for faction in FACTION_ORDER}
    for category_name, *_ in CATEGORY_ORDER:
        row = f"| {category_name} "
        category_total = 0
        for faction in FACTION_ORDER:
            count = summary.get(faction, {}).get(category_name, 0)
            grand_total[faction] += count
            category_total += count
            row += f"| {count} "
        row += f"| {category_total} |"
        lines.append(row)

    row = "| **合計** "
    all_total = 0
    for faction in FACTION_ORDER:
        row += f"| **{grand_total[faction]}** "
        all_total += grand_total[faction]
    row += f"| **{all_total}** |"
    lines.append(row)
    lines.append("")


def generate_cards_md(cards, faction_data, *, out_path):
    """カード一覧文書を書き出す。

    Args:
        cards: 検証済みのカード定義。
        faction_data: 陣営名から表示名とカードのリストへの辞書。
        out_path: 書き出す場所。

    Returns:
        文書の冒頭に書いたカードの総数。
    """
    total = len(cards)
    lines = []

    lines.append("<!-- This file is auto-generated by scripts/generate_cards.py. DO NOT EDIT. -->")
    lines.append("")
    lines.append("# Overload Party Card List")
    lines.append("")
    lines.append(f"**全 {total} 枚**")
    lines.append("")
    lines.append("---")
    lines.append("")

    _append_restriction_section(faction_data, lines)
    summary = _append_faction_sections(faction_data, lines)
    _append_summary_table(faction_data, summary, lines)

    write_lines_file(out_path, lines)
    return total


def main():
    """カード定義を検証し、配布用 JSON・カード一覧文書・初期投入 SQL を生成する。"""
    if not YAML_DIR.exists():
        exit_with_error(f"YAML directory not found: {YAML_DIR}")

    try:
        cards, faction_data = load_all_cards()
    except CardGenerationError as err:
        exit_with_error(err)
    if not cards:
        exit_with_error("No cards loaded from YAML files")

    try:
        taxonomy = load_effect_taxonomy(
            resolve_common_constants_path(), resolve_battle_constants_path()
        )
        expand_card_resource_groups(cards, taxonomy.resource_groups)
    except CardGenerationError as err:
        exit_with_error(err)

    errors = validate_cards(cards, taxonomy) + validate_effect_taxonomy(cards, taxonomy)
    if errors:
        exit_with_validation_errors(errors)

    json_count = generate_cards_json(cards, out_path=GO_JSON_OUT)
    print(f"Generated {json_count} cards → {GO_JSON_OUT.relative_to(ROOT)}", file=sys.stderr)

    md_count = generate_cards_md(cards, faction_data, out_path=MD_OUT)
    print(f"Generated {md_count} cards → {MD_OUT.relative_to(ROOT)}", file=sys.stderr)

    seed_count = generate_cards_seed_sql(cards, out_path=SEED_OUT)
    print(f"Generated {seed_count} cards → {SEED_OUT.relative_to(ROOT)}", file=sys.stderr)


if __name__ == "__main__":
    main()
