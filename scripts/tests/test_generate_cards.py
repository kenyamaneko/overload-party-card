"""generate_cards.py の validate() のユニットテスト."""

from __future__ import annotations

import pytest

import generate_cards as gen

_COMPUTE_STATS = {"throughput": 10, "availability": 1, "maintenance_cost": 1, "sla_penalty": 1}
_DATA_STATS = {"yield": 10, "availability": 1, "maintenance_cost": 1, "sla_penalty": 1}

_CARD_TYPE_EXTRA_FIELDS = {
    "Compute": {"subtype": "VM", "resizable": True, "elastic": False, "stats": dict(_COMPUTE_STATS)},
    "DataResource": {"subtype": "Database", "resizable": True, "elastic": False, "stats": dict(_DATA_STATS)},
    "Log": {},
    "Platform": {"resizable": False, "elastic": False, "stats": {}},
    "Attachment": {"resizable": False, "elastic": False, "stats": {}},
    "Strategy": {"resizable": False, "elastic": False, "stats": {}},
    "Reactive": {"resizable": False, "elastic": False, "stats": {}},
    "Incident": {"resizable": False, "elastic": False, "stats": {}},
}


def _valid_card_for_type(card_type: str, **overrides) -> dict:
    """指定 card_type の必須フィールドを満たすカード dict を組み立てる。

    Args:
        card_type: 組み立てるカードの card_type。
        **overrides: デフォルト値を上書きするフィールド。

    Returns:
        validate() にそのまま渡せるカード dict。
    """
    card = {
        "card_no": 1,
        "card_id": "NT-9901",
        "card_name": "テストカード",
        "const_name": "TestCard",
        "card_type": card_type,
        "restriction": "unlimited",
        "is_active": True,
        "faction": "SHE",
    }
    card.update(_CARD_TYPE_EXTRA_FIELDS[card_type])
    card.update(overrides)
    return card


def _valid_card(**overrides) -> dict:
    """Compute カードのデフォルト値でカード dict を組み立てる。

    Args:
        **overrides: デフォルト値を上書きするフィールド。

    Returns:
        validate() にそのまま渡せるカード dict。
    """
    return _valid_card_for_type("Compute", **overrides)


def _valid_data_card(**overrides) -> dict:
    """DataResource カードのデフォルト値でカード dict を組み立てる。

    Args:
        **overrides: デフォルト値を上書きするフィールド。

    Returns:
        validate() にそのまま渡せるカード dict。
    """
    return _valid_card_for_type("DataResource", **overrides)


def _valid_log_card(**overrides) -> dict:
    """Log カードのデフォルト値でカード dict を組み立てる。

    Args:
        **overrides: デフォルト値を上書きするフィールド。

    Returns:
        validate() にそのまま渡せるカード dict。
    """
    return _valid_card_for_type("Log", **overrides)


def _without_fields(card: dict, *keys: str) -> dict:
    """指定フィールドを取り除いたカード dict のコピーを返す。

    Args:
        card: 元になるカード dict。
        *keys: 取り除くフィールド名。

    Returns:
        keys を持たないカード dict のコピー。
    """
    card = dict(card)
    for key in keys:
        card.pop(key, None)
    return card


class Testカード定義の検証エラー:
    @pytest.mark.parametrize(
        ("cards", "want_message"),
        [
            pytest.param([_valid_card(card_id="NT-99")], "must match XX-NNNN format", id="card_id が XX-NNNN 形式でないとき、エラーになる"),
            pytest.param(
                [_valid_card(), _valid_card(card_id="NT-9902", const_name="OtherCard")],
                "duplicate card_no",
                id="card_no が重複するとき、エラーになる",
            ),
            pytest.param([_valid_card(), _valid_card(card_no=2, const_name="OtherCard")], "duplicate card_id", id="card_id が重複するとき、エラーになる"),
            pytest.param([_valid_card(), _valid_card(card_no=2, card_id="NT-9902")], "duplicate const_name", id="const_name が重複するとき、エラーになる"),
            pytest.param(
                [{**_without_fields(_valid_card(), "subtype"), "card_type": "Quantum", "stats": {}}],
                "invalid card_type",
                id="未知の card_type のとき、エラーになる",
            ),
            pytest.param([_valid_card(subtype="Mainframe")], "invalid subtype", id="card_type に許されない subtype のとき、エラーになる"),
            pytest.param([_valid_card(restriction="banned")], "invalid restriction", id="未知の restriction のとき、エラーになる"),
            pytest.param([_valid_card(faction="Atlantis")], "invalid faction", id="未知の faction のとき、エラーになる"),
            pytest.param([_without_fields(_valid_card(), "restriction")], "missing required field 'restriction'", id="必須フィールドが欠けるとき、エラーになる"),
            pytest.param(
                [_valid_card(stats={"availability": 1, "maintenance_cost": 1, "sla_penalty": 1})],
                "compute card missing stats",
                id="Compute カードの stats にキーが欠けるとき、エラーになる",
            ),
            pytest.param(
                [_valid_data_card(stats={"availability": 1, "maintenance_cost": 1, "sla_penalty": 1})],
                "data card missing stats",
                id="DataResource カードの stats にキーが欠けるとき、エラーになる",
            ),
            pytest.param(
                [_valid_card(elastic=True, cost_per_request=1)],
                "elastic card missing required field 'free_tier'",
                id="elastic カードに free_tier が無いとき、エラーになる",
            ),
            pytest.param([_valid_card(deploy_turns=3)], "deploy_turns must be 0, 1, or 2", id="deploy_turns が 3 のとき、エラーになる"),
            pytest.param([_without_fields(_valid_card(), "subtype")], "requires 'subtype' field", id="Compute カードに subtype が無いとき、エラーになる"),
            pytest.param([_valid_card_for_type("Platform", subtype="VM")], "must not have 'subtype' field", id="Platform カードに subtype があるとき、エラーになる"),
            pytest.param([_valid_card(card_no=0)], "card_no must be a positive integer", id="card_no が 0 のとき、エラーになる"),
            pytest.param([_valid_card(resizable="yes")], "'resizable' must be a boolean", id="resizable が文字列のとき、エラーになる"),
            pytest.param([_valid_card(is_active=1)], "'is_active' must be a boolean", id="is_active が数値のとき、エラーになる"),
            pytest.param([_valid_card(stats={**_COMPUTE_STATS, "throughput": -1})], "must be non-negative", id="stats の値が負のとき、エラーになる"),
            pytest.param([_valid_card(stats={**_COMPUTE_STATS, "throughput": "high"})], "must be a number", id="stats の値が文字列のとき、エラーになる"),
            pytest.param([_valid_card(const_name="invalid_name")], "not a valid Go identifier", id="const_name が PascalCase でないとき、エラーになる"),
            pytest.param(
                [_valid_card(elastic=True, free_tier=-1, cost_per_request=1)],
                "'free_tier' must be a non-negative number",
                id="elastic の数値フィールドが負のとき、エラーになる",
            ),
        ],
    )
    def test_不正な入力を渡すとエラーになる(self, cards, want_message):
        errors = gen.validate(cards)
        assert any(want_message in e for e in errors)


class Testカード定義の境界値:
    @pytest.mark.parametrize(
        "cards",
        [
            pytest.param([_valid_card(deploy_turns=2)], id="deploy_turns が 2 のとき、エラーにならない"),
            pytest.param([_valid_log_card()], id="Log カードは stats が無くても、エラーにならない"),
            pytest.param([_valid_card()], id="Compute カードに subtype があるとき、エラーにならない"),
            pytest.param([_valid_card_for_type("Platform")], id="Platform カードに subtype が無いとき、エラーにならない"),
            pytest.param([_valid_card(card_no=1)], id="card_no が 1 のとき、エラーにならない"),
        ],
    )
    def test_有効な入力を渡すとエラーにならない(self, cards):
        assert gen.validate(cards) == []


class Testカード定義の列挙値網羅:
    @pytest.mark.parametrize(
        "cards",
        [pytest.param([_valid_card(faction=v)], id=f"faction の {v} のカードは、検証を通る") for v in sorted(gen.VALID_FACTIONS)],
    )
    def test_陣営の全値で検証を通る(self, cards):
        assert gen.validate(cards) == []

    @pytest.mark.parametrize(
        "cards",
        [pytest.param([_valid_card(restriction=v)], id=f"restriction の {v} のカードは、検証を通る") for v in sorted(gen.VALID_RESTRICTION)],
    )
    def test_制限区分の全値で検証を通る(self, cards):
        assert gen.validate(cards) == []

    @pytest.mark.parametrize(
        "cards",
        [pytest.param([_valid_card_for_type(v)], id=f"card_type の {v} のカードは、検証を通る") for v in sorted(gen.ALL_CARD_TYPES)],
    )
    def test_カード種別の全値で検証を通る(self, cards):
        assert gen.validate(cards) == []

    @pytest.mark.parametrize(
        "cards",
        [
            pytest.param([_valid_card(subtype=v)], id=f"Compute の subtype {v} のカードは、検証を通る")
            for v in sorted(gen.SUBTYPES_BY_CARD_TYPE["Compute"])
        ],
    )
    def test_Computeのsubtypeの全値で検証を通る(self, cards):
        assert gen.validate(cards) == []

    @pytest.mark.parametrize(
        "cards",
        [
            pytest.param([_valid_data_card(subtype=v)], id=f"DataResource の subtype {v} のカードは、検証を通る")
            for v in sorted(gen.SUBTYPES_BY_CARD_TYPE["DataResource"])
        ],
    )
    def test_DataResourceのsubtypeの全値で検証を通る(self, cards):
        assert gen.validate(cards) == []
