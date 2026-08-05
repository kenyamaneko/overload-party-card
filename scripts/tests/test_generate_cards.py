"""generate_cards.py のカード定義の検証と、リソースの括りの展開のユニットテスト."""

from __future__ import annotations

import pytest
import yaml

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


_RESOURCE_GROUPS = {
    "compute": {"card_type": "Compute"},
    "db": {"subtypes": ["Database", "CacheDB"]},
}

_TAXONOMY = gen.EffectTaxonomy(
    card_types=frozenset({"Compute", "DataResource", "Platform", "Attachment", "Strategy", "Reactive", "Incident"}),
    subtypes=frozenset({"VM", "Container", "Orchestrator", "Serverless", "AI/ML", "Database", "ObjectStorage", "CacheDB"}),
    resource_groups=_RESOURCE_GROUPS,
)

_EFFECT_CARD_ID = "NT-9901"
_EFFECT_CARD_NAME = "テストカード"
_CARD_LABEL = f"{_EFFECT_CARD_ID} {_EFFECT_CARD_NAME}"


def _card_with_effects(effects: list) -> dict:
    """効果定義を持つカード dict を組み立てる。

    Args:
        effects: カードに持たせる効果定義。

    Returns:
        展開・検証にそのまま渡せるカード dict。
    """
    return {"card_id": _EFFECT_CARD_ID, "card_name": _EFFECT_CARD_NAME, "effects": effects}


class Testリソースの括りの展開:
    @pytest.mark.parametrize(
        ("effects", "want"),
        [
            pytest.param(
                [{"trigger": "ignition", "ops": [{"apply_buff": {"selector": {"owner": "myself", "group": "compute"}}}]}],
                [{"trigger": "ignition", "ops": [{"apply_buff": {"selector": {"owner": "myself", "card_type": "Compute"}}}]}],
                id="compute の括りを書いた効果定義は、card_type が Compute になる",
            ),
            pytest.param(
                [{"trigger": "ignition", "ops": [{"deploy_from_repo": {"filter": {"group": "db"}}}]}],
                [{"trigger": "ignition", "ops": [{"deploy_from_repo": {"filter": {"subtype": ["Database", "CacheDB"]}}}]}],
                id="db の括りを書いた効果定義は、subtype が Database と CacheDB になる",
            ),
            pytest.param(
                [{"trigger": "deploy", "guard": [{"match": {"group": "db"}}]}],
                [{"trigger": "deploy", "guard": [{"match": {"subtype": ["Database", "CacheDB"]}}]}],
                id="guard の中に括りを書いた効果定義も、展開される",
            ),
            pytest.param(
                [{"custom": "cloud_shift", "meta": {"faction": "Tuners", "group": "db", "deploy_discount": 500}}],
                [{"custom": "cloud_shift", "meta": {"faction": "Tuners", "subtype": ["Database", "CacheDB"], "deploy_discount": 500}}],
                id="meta の中に括りを書いた効果定義も、展開される",
            ),
            pytest.param(
                [{"ops": [{"apply_buff": {"amount": {"per": {"count": {"group": "compute"}}}}}]}],
                [{"ops": [{"apply_buff": {"amount": {"per": {"count": {"card_type": "Compute"}}}}}]}],
                id="ops の入れ子の中に括りを書いた効果定義も、展開される",
            ),
            pytest.param(
                [{"trigger": "deploy", "guard": [{"match": {"subtype": "Database"}}], "ops": [{"search_repo": {"card_type": "Strategy"}}]}],
                [{"trigger": "deploy", "guard": [{"match": {"subtype": "Database"}}], "ops": [{"search_repo": {"card_type": "Strategy"}}]}],
                id="括りを書かない効果定義は、そのまま変わらない",
            ),
        ],
    )
    def test_カード定義の括りを展開する(self, effects, want):
        cards = [_card_with_effects(effects)]

        gen.expand_card_resource_groups(cards, _RESOURCE_GROUPS)

        assert cards[0]["effects"] == want

    @pytest.mark.parametrize(
        ("resource_groups", "effects", "want_message"),
        [
            pytest.param(
                _RESOURCE_GROUPS,
                [{"ops": [{"deploy_from_repo": {"filter": {"group": "storage"}}}]}],
                f"{_CARD_LABEL}: unknown resource group 'storage'",
                id="定義に無い括り名を書くとき、カード ID を添えて失敗する",
            ),
            pytest.param(
                _RESOURCE_GROUPS,
                [{"ops": [{"deploy_from_repo": {"filter": {"group": "db", "card_type": "DataResource"}}}]}],
                f"{_CARD_LABEL}: resource group 'db' cannot be combined with card_type",
                id="括りと card_type を併記するとき、失敗する",
            ),
            pytest.param(
                _RESOURCE_GROUPS,
                [{"ops": [{"deploy_from_repo": {"filter": {"group": "db", "subtype": "Database"}}}]}],
                f"{_CARD_LABEL}: resource group 'db' cannot be combined with subtype",
                id="括りと subtype を併記するとき、失敗する",
            ),
            pytest.param(
                {"db": {"card_type": "DataResource", "subtypes": ["Database", "CacheDB"]}},
                [{"ops": [{"deploy_from_repo": {"filter": {"group": "db"}}}]}],
                f"{_CARD_LABEL}: resource group 'db' must define exactly one of 'card_type' / 'subtypes'",
                id="括りの定義が card_type と subtypes の両方を持つとき、失敗する",
            ),
            pytest.param(
                {"db": {}},
                [{"ops": [{"deploy_from_repo": {"filter": {"group": "db"}}}]}],
                f"{_CARD_LABEL}: resource group 'db' must define exactly one of 'card_type' / 'subtypes'",
                id="括りの定義が card_type も subtypes も持たないとき、失敗する",
            ),
        ],
    )
    def test_展開できない効果定義を渡すと失敗する(self, resource_groups, effects, want_message):
        cards = [_card_with_effects(effects)]

        with pytest.raises(gen.CardGenerationError) as excinfo:
            gen.expand_card_resource_groups(cards, resource_groups)

        assert str(excinfo.value) == want_message


class Test効果定義のカード種別とサブタイプ:
    @pytest.mark.parametrize(
        ("effects", "want_message"),
        [
            pytest.param(
                [{"ops": [{"search_repo": {"card_type": ["Database", "CacheDB"]}}]}],
                f"{_CARD_LABEL}: effect 'card_type' has invalid value 'Database'",
                id="card_type に subtype の Database を書くとき、エラーになる",
            ),
            pytest.param(
                [{"ops": [{"deploy_from_repo": {"filter": {"subtype": "DataResource"}}}]}],
                f"{_CARD_LABEL}: effect 'subtype' has invalid value 'DataResource'",
                id="subtype に card_type の DataResource を書くとき、エラーになる",
            ),
            pytest.param(
                [{"guard": [{"count": {"selector": {"subtype": "Blob"}}}]}],
                f"{_CARD_LABEL}: effect 'subtype' has invalid value 'Blob'",
                id="guard の中の subtype が正規値でないとき、エラーになる",
            ),
            pytest.param(
                [{"custom": "cloud_shift", "meta": {"card_type": ["Database", "CacheDB"]}}],
                f"{_CARD_LABEL}: effect 'card_type' has invalid value 'CacheDB'",
                id="meta の中の card_type が正規値でないとき、エラーになる",
            ),
            pytest.param(
                [{"ops": [{"apply_buff": {"selector": {"subtype": []}}}]}],
                f"{_CARD_LABEL}: effect 'subtype' is empty",
                id="subtype を空の並びで書くとき、エラーになる",
            ),
            pytest.param(
                [{"custom": "cloud_shift", "meta": {"card_type": []}}],
                f"{_CARD_LABEL}: effect 'card_type' is empty",
                id="meta の中の card_type を空の並びで書くとき、エラーになる",
            ),
        ],
    )
    def test_正規値でない値を書くとエラーになる(self, effects, want_message):
        errors = gen.validate_effect_taxonomy([_card_with_effects(effects)], _TAXONOMY)

        assert want_message in errors

    @pytest.mark.parametrize(
        "effects",
        [
            pytest.param(
                [{"ops": [{"apply_buff": {"selector": {"card_type": "Compute"}}}]}],
                id="card_type に Compute を書くとき、エラーにならない",
            ),
            pytest.param(
                [{"custom": "cloud_shift", "meta": {"card_type": ["Compute"]}}],
                id="card_type に Compute だけの並びを書くとき、エラーにならない",
            ),
            pytest.param(
                [{"ops": [{"deploy_from_repo": {"filter": {"subtype": ["Database", "CacheDB"]}}}]}],
                id="subtype に Database と CacheDB の並びを書くとき、エラーにならない",
            ),
            pytest.param(
                [{"guard": [{"match": {"subtype": "ObjectStorage"}}]}],
                id="guard の中の subtype に ObjectStorage を書くとき、エラーにならない",
            ),
        ],
    )
    def test_正規値を書くとエラーにならない(self, effects):
        assert gen.validate_effect_taxonomy([_card_with_effects(effects)], _TAXONOMY) == []


_CONSTANTS = {
    "card_types": ["Compute", "DataResource"],
    "subtypes": {"Compute": ["VM", "Container"], "DataResource": ["Database", "CacheDB"]},
    "resource_groups": {"db": {"subtypes": ["Database", "CacheDB"]}},
}


def _write_constants(base_dir, constants: dict):
    """ゲーム定数ファイルを common と同じ配置で書き出す。

    Args:
        base_dir: common のチェックアウトに見立てたディレクトリ。
        constants: 書き出すゲーム定数。

    Returns:
        書き出したファイルの Path。
    """
    path = base_dir / "data" / "game_design_constants.yaml"
    path.parent.mkdir(parents=True)
    path.write_text(yaml.safe_dump(constants, allow_unicode=True), encoding="utf-8")
    return path


class Testゲーム定数の場所:
    def test_環境変数が設定されているとき_その配下のファイルを指す(self, tmp_path, monkeypatch):
        monkeypatch.setenv("COMMON_REPO", str(tmp_path / "elsewhere"))

        assert gen.resolve_common_constants_path() == tmp_path / "elsewhere" / "data" / "game_design_constants.yaml"

    def test_環境変数が設定されていないとき_隣に置かれた_common_を指す(self, monkeypatch):
        monkeypatch.delenv("COMMON_REPO", raising=False)

        got = gen.resolve_common_constants_path()

        assert got.parent.parent.name == "overload-party-common"


class Testゲーム定数の読み込み:
    def test_サブタイプはカード種別ごとの並びをまとめた一つの集合になる(self, tmp_path):
        path = _write_constants(tmp_path, _CONSTANTS)

        taxonomy = gen.load_effect_taxonomy(path)

        assert taxonomy.card_types == frozenset({"Compute", "DataResource"})
        assert taxonomy.subtypes == frozenset({"VM", "Container", "Database", "CacheDB"})
        assert taxonomy.resource_groups == {"db": {"subtypes": ["Database", "CacheDB"]}}

    def test_ファイルが無いとき_置き場所の指定方法を添えて失敗する(self, tmp_path):
        missing = tmp_path / "data" / "game_design_constants.yaml"

        with pytest.raises(gen.CardGenerationError) as excinfo:
            gen.load_effect_taxonomy(missing)

        assert "not found" in str(excinfo.value)
        assert "COMMON_REPO" in str(excinfo.value)

    @pytest.mark.parametrize(
        "missing_key",
        [
            pytest.param("card_types", id="カード種別の一覧が無いとき、失敗する"),
            pytest.param("subtypes", id="サブタイプの一覧が無いとき、失敗する"),
            pytest.param("resource_groups", id="リソースの括りの定義が無いとき、失敗する"),
        ],
    )
    def test_必要な項目が欠けているとき_欠けている項目を挙げて失敗する(self, tmp_path, missing_key):
        constants = {k: v for k, v in _CONSTANTS.items() if k != missing_key}
        path = _write_constants(tmp_path, constants)

        with pytest.raises(gen.CardGenerationError) as excinfo:
            gen.load_effect_taxonomy(path)

        assert f"missing '{missing_key}'" in str(excinfo.value)
