import re
import shutil
from types import SimpleNamespace

import generate_cards
import pytest
import yaml

COMPUTE_STAT_NAMES = ["throughput", "availability", "maintenance_cost", "sla_penalty"]
DATA_STAT_NAMES = ["yield", "availability", "maintenance_cost", "sla_penalty"]
CARD_LABEL = "#1 TSTカードA"
UNDEFINED_VALUE_PHRASE = "that is not defined in the game constants"
EFFECT_LABEL = "SH-9001 TSTカードA"
EXPECTED_CARD_IDS = ["SH-9001", "TK-9002"]
FIRST_CARD_NUMBER = 9001
GENERATED_CARD_OUTPUT_KEYS = ["cards_json", "cards_document", "cards_seed"]
NO_RESOURCE_GROUPS = {}
RUN_WITHOUT_COMMON_REPO = {"sets_common_repo": False}
RUN_WITHOUT_BATTLE_REPO = {"sets_battle_repo": False}


def drop_stat_from_stats(stats, name):
    return {key: value for key, value in stats.items() if key != name}


class Testカード定義__カード定義の検証:
    class Test正常系:
        class Test必須項目と_stats:
            @pytest.mark.parametrize(
                ("card_type", "fields", "omit"),
                [
                    pytest.param(
                        "Platform",
                        {},
                        (),
                        id="card_type が Platform、card_no が 1、card_id が SH-9001、card_name が TSTカードA、const_name が TstCardA、resizable が false、elastic が false、restriction が unlimited、is_active が true、stats が空のカードのとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Log",
                        {},
                        ("resizable", "elastic", "stats"),
                        id="Log カードが card_no, card_id, card_name, const_name, card_type, restriction, is_active だけを持つとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Platform",
                        {"stats": None},
                        (),
                        id="Platform カードの stats に値が書かれていないとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "DataResource",
                        {"subtype": "Database"},
                        (),
                        id="DataResource カードが yield, availability, maintenance_cost, sla_penalty を持つとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Compute",
                        {
                            "subtype": "VM",
                            "stats": {
                                "throughput": 0,
                                "availability": 0,
                                "maintenance_cost": 0,
                                "sla_penalty": 0,
                            },
                        },
                        (),
                        id="Compute カードの throughput, availability, maintenance_cost, sla_penalty がすべて 0 のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card_of_type, card_type, fields, omit):
                card = build_card_of_type(card_type, omit=omit, **fields)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testcard_type_が_Compute_のときの_subtype_の値:
            @pytest.mark.parametrize(
                "subtype",
                [
                    pytest.param(
                        "VM", id="VM のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "Container",
                        id="Container のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Orchestrator",
                        id="Orchestrator のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Serverless",
                        id="Serverless のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "AI/ML", id="AI/ML のとき、バリデーションエラーの一覧が空になる"
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_compute_card, subtype):
                card = build_compute_card(subtype=subtype)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testcard_type_が_DataResource_のときの_subtype_の値:
            @pytest.mark.parametrize(
                "subtype",
                [
                    pytest.param(
                        "Database",
                        id="Database のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "ObjectStorage",
                        id="ObjectStorage のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "CacheDB",
                        id="CacheDB のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_data_resource_card, subtype):
                card = build_data_resource_card(subtype=subtype)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testsubtype_が無いときの_card_type_の値:
            @pytest.mark.parametrize(
                "card_type",
                [
                    pytest.param(
                        "Platform",
                        id="Platform のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Attachment",
                        id="Attachment のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Strategy",
                        id="Strategy のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Reactive",
                        id="Reactive のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "Incident",
                        id="Incident のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, card_type):
                card = build_card(card_type=card_type)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Test数字_4_桁の_card_id_の接頭辞:
            @pytest.mark.parametrize(
                "prefix",
                [
                    pytest.param(
                        "SH", id="SH のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "TK", id="TK のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "SL", id="SL のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "TN", id="TN のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "NT", id="NT のとき、バリデーションエラーの一覧が空になる"
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, prefix):
                card = build_card(card_id=f"{prefix}-9001")

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testconst_name:
            @pytest.mark.parametrize(
                "const_name",
                [
                    pytest.param(
                        "TstCardA",
                        id="const_name が TstCardA のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "TstCard2",
                        id="const_name が TstCard2 のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, const_name):
                card = build_card(const_name=const_name)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testrestriction_の値:
            @pytest.mark.parametrize(
                "restriction",
                [
                    pytest.param(
                        "unlimited",
                        id="unlimited のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "limited",
                        id="limited のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "semi_limited",
                        id="semi_limited のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "forbidden",
                        id="forbidden のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, restriction):
                card = build_card(restriction=restriction)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testfaction_の値:
            @pytest.mark.parametrize(
                "faction",
                [
                    pytest.param(
                        "SHE", id="SHE のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "Tenki", id="Tenki のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "Sugar", id="Sugar のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "Tuners", id="Tuners のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        "Neutral",
                        id="Neutral のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, faction):
                card = build_card(faction=faction)

                assert generate_cards.validate_cards([card], taxonomy) == []

        def test_card_no_が_1_のとき_バリデーションエラーの一覧が空になる(
            self, taxonomy, build_card
        ):
            card = build_card(card_no=1)

            assert generate_cards.validate_cards([card], taxonomy) == []

        class Testdeploy_turns_の値:
            @pytest.mark.parametrize(
                "deploy_turns",
                [
                    pytest.param(
                        0, id="0 のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        1, id="1 のとき、バリデーションエラーの一覧が空になる"
                    ),
                    pytest.param(
                        2, id="2 のとき、バリデーションエラーの一覧が空になる"
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, deploy_turns):
                card = build_card(deploy_turns=deploy_turns)

                assert generate_cards.validate_cards([card], taxonomy) == []

        class Testelastic_が_true_のカードの弾力課金の項目:
            @pytest.mark.parametrize(
                "zero_field",
                [
                    pytest.param(
                        "free_tier",
                        id="free_tier が 0 のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "elastic_increment",
                        id="elastic_increment が 0 のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "cost_per_request",
                        id="cost_per_request が 0 のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_compute_card, zero_field):
                billing = {
                    "elastic_increment": 100,
                    "free_tier": 500,
                    "cost_per_request": 10,
                    zero_field: 0,
                }
                card = build_compute_card(elastic=True, **billing)

                assert generate_cards.validate_cards([card], taxonomy) == []

        def test_elastic_が_false_のカードで_free_tier_と_cost_per_request_が無いとき_バリデーションエラーの一覧が空になる(
            self, taxonomy, build_compute_card
        ):
            card = build_compute_card(elastic=False)

            assert generate_cards.validate_cards([card], taxonomy) == []

    class Test異常系:
        class Testリソースカードの必須項目:
            @pytest.mark.parametrize(
                ("field", "label"),
                [
                    pytest.param(
                        "card_no",
                        "",
                        id="card_no が無いとき、バリデーションエラーの一覧に card_no の欠落が含まれる",
                    ),
                    pytest.param(
                        "card_id",
                        CARD_LABEL,
                        id="card_id が無いとき、バリデーションエラーの一覧に card_id の欠落が含まれる",
                    ),
                    pytest.param(
                        "card_name",
                        "",
                        id="card_name が無いとき、バリデーションエラーの一覧に card_name の欠落が含まれる",
                    ),
                    pytest.param(
                        "const_name",
                        CARD_LABEL,
                        id="const_name が無いとき、バリデーションエラーの一覧に const_name の欠落が含まれる",
                    ),
                    pytest.param(
                        "card_type",
                        CARD_LABEL,
                        id="card_type が無いとき、バリデーションエラーの一覧に card_type の欠落が含まれる",
                    ),
                    pytest.param(
                        "resizable",
                        CARD_LABEL,
                        id="resizable が無いとき、バリデーションエラーの一覧に resizable の欠落が含まれる",
                    ),
                    pytest.param(
                        "elastic",
                        CARD_LABEL,
                        id="elastic が無いとき、バリデーションエラーの一覧に elastic の欠落が含まれる",
                    ),
                    pytest.param(
                        "restriction",
                        CARD_LABEL,
                        id="restriction が無いとき、バリデーションエラーの一覧に restriction の欠落が含まれる",
                    ),
                    pytest.param(
                        "is_active",
                        CARD_LABEL,
                        id="is_active が無いとき、バリデーションエラーの一覧に is_active の欠落が含まれる",
                    ),
                    pytest.param(
                        "stats",
                        CARD_LABEL,
                        id="stats が無いとき、バリデーションエラーの一覧に stats の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, field, label):
                card = build_card(omit=(field,))

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, f"missing required field '{field}'", label=label
                )

        class TestLog_カードの必須項目:
            @pytest.mark.parametrize(
                "field",
                [
                    pytest.param(
                        "restriction",
                        id="restriction が無いとき、バリデーションエラーの一覧に restriction の欠落が含まれる",
                    ),
                    pytest.param(
                        "is_active",
                        id="is_active が無いとき、バリデーションエラーの一覧に is_active の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, field):
                card = build_card(
                    card_type="Log", omit=("resizable", "elastic", "stats", field)
                )

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, f"missing required field '{field}'", label=CARD_LABEL
                )

        class Testsubtype_の有無が_card_type_と合わないとき:
            class Testsubtype_が無いとき:
                @pytest.mark.parametrize(
                    "card_type",
                    [
                        pytest.param(
                            "Compute",
                            id="card_type が Compute のとき、バリデーションエラーの一覧に subtype が必須であることを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "DataResource",
                            id="card_type が DataResource のとき、バリデーションエラーの一覧に subtype が必須であることを示す誤りが含まれる",
                        ),
                    ],
                )
                def test_検証する(self, taxonomy, build_card_of_type, has_error, card_type):
                    card = build_card_of_type(card_type)

                    errors = generate_cards.validate_cards([card], taxonomy)

                    assert has_error(
                        errors,
                        f"card_type '{card_type}' requires 'subtype' field",
                        label=CARD_LABEL,
                    )

            class Testsubtype_が書かれているとき:
                @pytest.mark.parametrize(
                    "card_type",
                    [
                        pytest.param(
                            "Platform",
                            id="card_type が Platform のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "Log",
                            id="card_type が Log のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "Attachment",
                            id="card_type が Attachment のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "Strategy",
                            id="card_type が Strategy のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "Reactive",
                            id="card_type が Reactive のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                        pytest.param(
                            "Incident",
                            id="card_type が Incident のとき、バリデーションエラーの一覧に subtype を持てないことを示す誤りが含まれる",
                        ),
                    ],
                )
                def test_検証する(self, taxonomy, build_card_of_type, has_error, card_type):
                    card = build_card_of_type(card_type, subtype="VM")

                    errors = generate_cards.validate_cards([card], taxonomy)

                    assert has_error(
                        errors,
                        f"card_type '{card_type}' must not have 'subtype' field",
                        label=CARD_LABEL,
                    )

        class Testsubtype_の値が_card_type_と合わないとき:
            @pytest.mark.parametrize(
                ("card_type", "subtype"),
                [
                    pytest.param(
                        "Compute",
                        "Database",
                        id="card_type が Compute で subtype が Database のとき、バリデーションエラーの一覧に subtype の値の誤りが含まれる",
                    ),
                    pytest.param(
                        "DataResource",
                        "VM",
                        id="card_type が DataResource で subtype が VM のとき、バリデーションエラーの一覧に subtype の値の誤りが含まれる",
                    ),
                    pytest.param(
                        "Compute",
                        "TST_UNKNOWN",
                        id="card_type が Compute で subtype が TST_UNKNOWN のとき、バリデーションエラーの一覧に subtype の値 TST_UNKNOWN の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(
                self, taxonomy, build_card_of_type, has_error, card_type, subtype
            ):
                card = build_card_of_type(card_type, subtype=subtype)

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors,
                    f"invalid subtype '{subtype}' for card_type '{card_type}'",
                    label=CARD_LABEL,
                )

        def test_card_type_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_card_type_の値_TST_UNKNOWN_の誤りが含まれる(
            self, taxonomy, build_card, has_error
        ):
            card = build_card(card_type="TST_UNKNOWN")

            errors = generate_cards.validate_cards([card], taxonomy)

            assert has_error(errors, "invalid card_type 'TST_UNKNOWN'", label=CARD_LABEL)

        class Test別のカードと値が重複するとき:
            @pytest.mark.parametrize(
                ("second_card_fields", "phrase", "label"),
                [
                    pytest.param(
                        {
                            "card_id": "SH-9002",
                            "card_name": "TSTカードB",
                            "const_name": "TstCardB",
                        },
                        "duplicate card_no",
                        "#1 TSTカードB",
                        id="2 枚のカードの card_no がどちらも 1 のとき、バリデーションエラーの一覧に card_no の重複が含まれる",
                    ),
                    pytest.param(
                        {
                            "card_no": 2,
                            "card_name": "TSTカードB",
                            "const_name": "TstCardB",
                        },
                        "duplicate card_id 'SH-9001'",
                        "#2 TSTカードB",
                        id="2 枚のカードの card_id がどちらも SH-9001 のとき、バリデーションエラーの一覧に card_id SH-9001 の重複が含まれる",
                    ),
                    pytest.param(
                        {
                            "card_no": 2,
                            "card_id": "SH-9002",
                            "card_name": "TSTカードB",
                        },
                        "duplicate const_name 'TstCardA'",
                        "#2 TSTカードB",
                        id="2 枚のカードの const_name がどちらも TstCardA のとき、バリデーションエラーの一覧に const_name TstCardA の重複が含まれる",
                    ),
                ],
            )
            def test_検証する(
                self, taxonomy, build_card, has_error, second_card_fields, phrase, label
            ):
                cards = [build_card(), build_card(**second_card_fields)]

                errors = generate_cards.validate_cards(cards, taxonomy)

                assert has_error(errors, phrase, label=label)

        class Testcard_id_の形式が誤っているとき:
            @pytest.mark.parametrize(
                "card_id",
                [
                    pytest.param(
                        "SH-901",
                        id="card_id が SH-901 のとき、バリデーションエラーの一覧に card_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "SH-90001",
                        id="card_id が SH-90001 のとき、バリデーションエラーの一覧に card_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "XX-9001",
                        id="card_id が XX-9001 のとき、バリデーションエラーの一覧に card_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "sh-9001",
                        id="card_id が sh-9001 のとき、バリデーションエラーの一覧に card_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "SH9001",
                        id="card_id が SH9001 のとき、バリデーションエラーの一覧に card_id の形式の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, card_id):
                card = build_card(card_id=card_id)

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(errors, "must match XX-NNNN format", label=CARD_LABEL)

        class Testconst_name_が英字始まりの英数字でないとき:
            @pytest.mark.parametrize(
                "const_name",
                [
                    pytest.param(
                        "tstCardA",
                        id="const_name が tstCardA のとき、バリデーションエラーの一覧に const_name の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "1TstCard",
                        id="const_name が 1TstCard のとき、バリデーションエラーの一覧に const_name の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "Tst-Card",
                        id="const_name が Tst-Card のとき、バリデーションエラーの一覧に const_name の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "Tstカード",
                        id="const_name が Tstカード のとき、バリデーションエラーの一覧に const_name の形式の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, const_name):
                card = build_card(const_name=const_name)

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "is not a valid Go identifier", label=CARD_LABEL
                )

        class Test真偽値の項目が真偽値でないとき:
            @pytest.mark.parametrize(
                ("field", "value"),
                [
                    pytest.param(
                        "resizable",
                        "true",
                        id='resizable が文字列の "true" のとき、バリデーションエラーの一覧に resizable が真偽値でない誤りが含まれる',
                    ),
                    pytest.param(
                        "elastic",
                        1,
                        id="elastic が数値の 1 のとき、バリデーションエラーの一覧に elastic が真偽値でない誤りが含まれる",
                    ),
                    pytest.param(
                        "is_active",
                        "yes",
                        id='is_active が文字列の "yes" のとき、バリデーションエラーの一覧に is_active が真偽値でない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, field, value):
                card = build_card(**{field: value})

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, f"'{field}' must be a boolean", label=CARD_LABEL
                )

        def test_restriction_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_restriction_の値_TST_UNKNOWN_の誤りが含まれる(
            self, taxonomy, build_card, has_error
        ):
            card = build_card(restriction="TST_UNKNOWN")

            errors = generate_cards.validate_cards([card], taxonomy)

            assert has_error(
                errors, "invalid restriction 'TST_UNKNOWN'", label=CARD_LABEL
            )

        def test_faction_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_faction_の値_TST_UNKNOWN_の誤りが含まれる(
            self, taxonomy, build_card, has_error
        ):
            card = build_card(faction="TST_UNKNOWN")

            errors = generate_cards.validate_cards([card], taxonomy)

            assert has_error(errors, "invalid faction 'TST_UNKNOWN'", label=CARD_LABEL)

        class Testcard_no_が正の整数でないとき:
            @pytest.mark.parametrize(
                ("card_no", "label"),
                [
                    pytest.param(
                        0,
                        "#0 TSTカードA",
                        id="0 のとき、バリデーションエラーの一覧に card_no が正の整数でない誤りが含まれる",
                    ),
                    pytest.param(
                        1.5,
                        "#1.5 TSTカードA",
                        id="小数の1点5 のとき、バリデーションエラーの一覧に card_no が正の整数でない誤りが含まれる",
                    ),
                    pytest.param(
                        "1",
                        "#1 TSTカードA",
                        id='文字列の "1" のとき、バリデーションエラーの一覧に card_no が正の整数でない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, card_no, label):
                card = build_card(card_no=card_no)

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "card_no must be a positive integer", label=label
                )

        class Testdeploy_turns_が_0_1_2_のどれでもないとき:
            @pytest.mark.parametrize(
                "deploy_turns",
                [
                    pytest.param(
                        3,
                        id="3 のとき、バリデーションエラーの一覧に deploy_turns の値の誤りが含まれる",
                    ),
                    pytest.param(
                        -1,
                        id="マイナス1 のとき、バリデーションエラーの一覧に deploy_turns の値の誤りが含まれる",
                    ),
                    pytest.param(
                        "1",
                        id='文字列の "1" のとき、バリデーションエラーの一覧に deploy_turns の値の誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_card, has_error, deploy_turns):
                card = build_card(deploy_turns=deploy_turns)

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "deploy_turns must be 0, 1, or 2", label=CARD_LABEL
                )

        class TestCompute_カードの_stats_の項目:
            @pytest.mark.parametrize(
                "stat_name",
                [
                    pytest.param(
                        "throughput",
                        id="throughput が無いとき、バリデーションエラーの一覧に Compute カードの throughput の欠落が含まれる",
                    ),
                    pytest.param(
                        "availability",
                        id="availability が無いとき、バリデーションエラーの一覧に Compute カードの availability の欠落が含まれる",
                    ),
                    pytest.param(
                        "maintenance_cost",
                        id="maintenance_cost が無いとき、バリデーションエラーの一覧に Compute カードの maintenance_cost の欠落が含まれる",
                    ),
                    pytest.param(
                        "sla_penalty",
                        id="sla_penalty が無いとき、バリデーションエラーの一覧に Compute カードの sla_penalty の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(self, taxonomy, build_compute_card, has_error, stat_name):
                default_stats = build_compute_card()["stats"]
                card = build_compute_card(
                    stats=drop_stat_from_stats(default_stats, stat_name)
                )

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "compute card missing stats", stat_name, label=CARD_LABEL
                )

        class TestDataResource_カードの_stats_の項目:
            @pytest.mark.parametrize(
                "stat_name",
                [
                    pytest.param(
                        "yield",
                        id="yield が無いとき、バリデーションエラーの一覧に DataResource カードの yield の欠落が含まれる",
                    ),
                    pytest.param(
                        "availability",
                        id="availability が無いとき、バリデーションエラーの一覧に DataResource カードの availability の欠落が含まれる",
                    ),
                    pytest.param(
                        "maintenance_cost",
                        id="maintenance_cost が無いとき、バリデーションエラーの一覧に DataResource カードの maintenance_cost の欠落が含まれる",
                    ),
                    pytest.param(
                        "sla_penalty",
                        id="sla_penalty が無いとき、バリデーションエラーの一覧に DataResource カードの sla_penalty の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(
                self, taxonomy, build_data_resource_card, has_error, stat_name
            ):
                default_stats = build_data_resource_card()["stats"]
                card = build_data_resource_card(
                    stats=drop_stat_from_stats(default_stats, stat_name)
                )

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "data card missing stats", stat_name, label=CARD_LABEL
                )

        class Teststats_が空のとき:
            def test_Compute_カードの_stats_が空のとき_バリデーションエラーの一覧に_Compute_カードの_throughputavailabilitymaintenance_costsla_penalty_の欠落が含まれる(
                self, taxonomy, build_compute_card, has_error
            ):
                card = build_compute_card(stats={})

                errors = generate_cards.validate_cards([card], taxonomy)

                assert all(
                    has_error(
                        errors, "compute card missing stats", name, label=CARD_LABEL
                    )
                    for name in COMPUTE_STAT_NAMES
                )

            def test_DataResource_カードの_stats_が空のとき_バリデーションエラーの一覧に_DataResource_カードの_yieldavailabilitymaintenance_costsla_penalty_の欠落が含まれる(
                self, taxonomy, build_data_resource_card, has_error
            ):
                card = build_data_resource_card(stats={})

                errors = generate_cards.validate_cards([card], taxonomy)

                assert all(
                    has_error(errors, "data card missing stats", name, label=CARD_LABEL)
                    for name in DATA_STAT_NAMES
                )

        class Teststats_に未知の項目があるとき:
            def test_Compute_カードの_stats_に_throughput_と_availability_と_maintenance_cost_と_sla_penalty_のほかに_TST_UNKNOWN_STAT_があるとき_バリデーションエラーの一覧に_stats_の未知の項目_TST_UNKNOWN_STAT_を示す誤りが含まれる(
                self, taxonomy, build_compute_card, has_error
            ):
                default_stats = build_compute_card()["stats"]
                card = build_compute_card(
                    stats={**default_stats, "TST_UNKNOWN_STAT": 1}
                )

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(
                    errors, "stat 'TST_UNKNOWN_STAT' is unknown", label=CARD_LABEL
                )

        class TestCompute_カードの_throughput_が不正なとき:
            @pytest.mark.parametrize(
                ("value", "phrase"),
                [
                    pytest.param(
                        -1,
                        "stat 'throughput' must be non-negative",
                        id="マイナス1 のとき、バリデーションエラーの一覧に stats の値が負である誤りが含まれる",
                    ),
                    pytest.param(
                        "10",
                        "stat 'throughput' must be a number",
                        id='文字列の "10" のとき、バリデーションエラーの一覧に stats の値が数値でない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(
                self, taxonomy, build_compute_card, has_error, value, phrase
            ):
                default_stats = build_compute_card()["stats"]
                card = build_compute_card(stats={**default_stats, "throughput": value})

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(errors, phrase, label=CARD_LABEL)

        class Testelastic_が_true_のカードで弾力課金の項目が不正なとき:
            @pytest.mark.parametrize(
                ("fields", "omit", "phrase"),
                [
                    pytest.param(
                        {},
                        ("free_tier",),
                        "elastic card missing required field 'free_tier'",
                        id="free_tier が無いとき、バリデーションエラーの一覧に free_tier の欠落が含まれる",
                    ),
                    pytest.param(
                        {},
                        ("cost_per_request",),
                        "elastic card missing required field 'cost_per_request'",
                        id="cost_per_request が無いとき、バリデーションエラーの一覧に cost_per_request の欠落が含まれる",
                    ),
                    pytest.param(
                        {"elastic_increment": -1},
                        (),
                        "'elastic_increment' must be a non-negative number",
                        id="elastic_increment がマイナス1 のとき、バリデーションエラーの一覧に elastic_increment が負である誤りが含まれる",
                    ),
                    pytest.param(
                        {"free_tier": -1},
                        (),
                        "'free_tier' must be a non-negative number",
                        id="free_tier がマイナス1 のとき、バリデーションエラーの一覧に free_tier が負である誤りが含まれる",
                    ),
                    pytest.param(
                        {"cost_per_request": -1},
                        (),
                        "'cost_per_request' must be a non-negative number",
                        id="cost_per_request がマイナス1 のとき、バリデーションエラーの一覧に cost_per_request が負である誤りが含まれる",
                    ),
                    pytest.param(
                        {"cost_per_request": "5"},
                        (),
                        "'cost_per_request' must be a non-negative number",
                        id='cost_per_request が文字列の "5" のとき、バリデーションエラーの一覧に cost_per_request が数値でない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(
                self, taxonomy, build_compute_card, has_error, fields, omit, phrase
            ):
                card = build_compute_card(
                    omit=omit,
                    **{
                        "elastic": True,
                        "elastic_increment": 100,
                        "free_tier": 500,
                        "cost_per_request": 10,
                        **fields,
                    },
                )

                errors = generate_cards.validate_cards([card], taxonomy)

                assert has_error(errors, phrase, label=CARD_LABEL)

        def test_card_no_が_1_で_card_name_が_TSTカードA_のカードに_restriction_が無く_card_no_が_2_で_card_name_が_TSTカードB_のカードに_is_active_が無いとき_バリデーションエラーの一覧に1_TSTカードAの_restriction_の欠落と2_TSTカードBの_is_active_の欠落の_2_件が含まれる(
            self, taxonomy, build_card, has_error
        ):
            cards = [
                build_card(omit=("restriction",)),
                build_card(
                    omit=("is_active",),
                    card_no=2,
                    card_id="SH-9002",
                    card_name="TSTカードB",
                    const_name="TstCardB",
                ),
            ]

            errors = generate_cards.validate_cards(cards, taxonomy)

            assert has_error(
                errors, "missing required field 'restriction'", label="#1 TSTカードA"
            )
            assert has_error(
                errors, "missing required field 'is_active'", label="#2 TSTカードB"
            )


class Testカード定義__効果定義のリソースの括りの展開:
    class Test異常系:
        class Testcard_id_が_SH9001_のカードの効果の括り名が_TST_GROUP_X_で_TST_GROUP_X_の定義が無いとき_生成を続けられないエラーが発生する:
            def test_エラーのメッセージに未定義の括りであることが含まれる(self, build_card):
                card = build_card(effects=[{"group": "TST_GROUP_X"}])

                with pytest.raises(
                    generate_cards.CardGenerationError,
                    match=re.escape("unknown resource group"),
                ):
                    generate_cards.expand_card_resource_groups(
                        [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                    )

            def test_エラーのメッセージにカードの_card_id_SH9001_が含まれる(
                self, build_card
            ):
                card = build_card(effects=[{"group": "TST_GROUP_X"}])

                with pytest.raises(
                    generate_cards.CardGenerationError, match=re.escape("SH-9001")
                ):
                    generate_cards.expand_card_resource_groups(
                        [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                    )

            def test_エラーのメッセージに括り名_TST_GROUP_X_が含まれる(self, build_card):
                card = build_card(effects=[{"group": "TST_GROUP_X"}])

                with pytest.raises(
                    generate_cards.CardGenerationError, match=re.escape("TST_GROUP_X")
                ):
                    generate_cards.expand_card_resource_groups(
                        [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                    )

        def test_括り名が数値の_1_のとき_生成を続けられないエラーが発生し_エラーのメッセージに未定義の括りであることが含まれる(
            self, build_card
        ):
            card = build_card(effects=[{"group": 1}])

            with pytest.raises(
                generate_cards.CardGenerationError,
                match=re.escape("unknown resource group"),
            ):
                generate_cards.expand_card_resource_groups(
                    [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                )

        def test_効果に_card_type_が_Compute_group_が定義済みの_TST_GROUP_A_の両方が書かれているとき_生成を続けられないエラーが発生し_エラーのメッセージに_TST_GROUP_A_が_card_type_と併記できないことが含まれる(
            self, build_card
        ):
            card = build_card(
                effects=[{"card_type": "Compute", "group": "TST_GROUP_A"}]
            )

            with pytest.raises(
                generate_cards.CardGenerationError,
                match=re.escape(
                    "resource group 'TST_GROUP_A' cannot be combined with card_type"
                ),
            ):
                generate_cards.expand_card_resource_groups(
                    [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                )

        def test_効果に_subtype_が_VM_group_が定義済みの_TST_GROUP_A_の両方が書かれているとき_生成を続けられないエラーが発生し_エラーのメッセージに_TST_GROUP_A_が_subtype_と併記できないことが含まれる(
            self, build_card
        ):
            card = build_card(effects=[{"subtype": "VM", "group": "TST_GROUP_A"}])

            with pytest.raises(
                generate_cards.CardGenerationError,
                match=re.escape(
                    "resource group 'TST_GROUP_A' cannot be combined with subtype"
                ),
            ):
                generate_cards.expand_card_resource_groups(
                    [card], {"TST_GROUP_A": {"card_type": "Compute"}}
                )

        class Test括りの定義が_card_type_と_subtypes_のどちらか一方だけを持つ形でないとき:
            @pytest.mark.parametrize(
                "group_definition",
                [
                    pytest.param(
                        {},
                        id="TST_GROUP_A の定義が card_type も subtypes も持たないとき、生成を続けられないエラーが発生し、エラーのメッセージに TST_GROUP_A はどちらか一方だけを定義する必要があることが含まれる",
                    ),
                    pytest.param(
                        {"card_type": "Compute", "subtypes": ["VM"]},
                        id="TST_GROUP_A の定義が card_type の Compute と subtypes の VM の両方を持つとき、生成を続けられないエラーが発生し、エラーのメッセージに TST_GROUP_A はどちらか一方だけを定義する必要があることが含まれる",
                    ),
                ],
            )
            def test_展開する(self, build_card, group_definition):
                card = build_card(effects=[{"group": "TST_GROUP_A"}])

                with pytest.raises(
                    generate_cards.CardGenerationError,
                    match=re.escape(
                        "resource group 'TST_GROUP_A' must define exactly one of 'card_type' / 'subtypes'"
                    ),
                ):
                    generate_cards.expand_card_resource_groups(
                        [card], {"TST_GROUP_A": group_definition}
                    )


class Testカード定義__効果定義の値の検証:
    class Testゲーム定数の_trigger_に_TST_TRIGGER_A_と_TST_TRIGGER_B_が定義されているとき:
        class Test正常系:
            @pytest.mark.parametrize(
                "trigger",
                [
                    pytest.param(
                        "TST_TRIGGER_A",
                        id="効果の trigger が TST_TRIGGER_A のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        "TST_TRIGGER_B",
                        id="効果の trigger が TST_TRIGGER_B のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, build_effect_taxonomy, build_card, trigger):
                taxonomy = build_effect_taxonomy(
                    triggers=["TST_TRIGGER_A", "TST_TRIGGER_B"]
                )
                card = build_card(effects=[{"trigger": trigger}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_効果の_trigger_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_trigger_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    triggers=["TST_TRIGGER_A", "TST_TRIGGER_B"]
                )
                card = build_card(effects=[{"trigger": "TST_UNKNOWN"}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'trigger' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_op_に_TST_OP_A_と_TST_OP_B_が定義されているとき:
        class Test正常系:
            @pytest.mark.parametrize(
                "effect",
                [
                    pytest.param(
                        {"ops": [{"TST_OP_B": {}}]},
                        id="ops の op が TST_OP_B のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        {"choice": {"use": [{"TST_OP_B": {}}], "skip": []}},
                        id="choice の use の内側の op が TST_OP_B のとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, build_effect_taxonomy, build_card, effect):
                taxonomy = build_effect_taxonomy(ops=["TST_OP_A", "TST_OP_B"])
                card = build_card(effects=[effect])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            @pytest.mark.parametrize(
                "effect",
                [
                    pytest.param(
                        {"ops": [{"TST_UNKNOWN": {}}]},
                        id="ops の op が TST_UNKNOWN のとき、バリデーションエラーの一覧に op の値 TST_UNKNOWN がゲーム定数に定義されていないという誤りが含まれる",
                    ),
                    pytest.param(
                        {"choice": {"use": [{"TST_UNKNOWN": {}}], "skip": []}},
                        id="choice の use の内側の op が TST_UNKNOWN のとき、バリデーションエラーの一覧に op の値 TST_UNKNOWN がゲーム定数に定義されていないという誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_effect_taxonomy, build_card, has_error, effect):
                taxonomy = build_effect_taxonomy(ops=["TST_OP_A", "TST_OP_B"])
                card = build_card(effects=[effect])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'op' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_op_に_apply_buff_buff_に_TST_BUFF_A_と_TST_BUFF_B_が定義されているとき:
        class Test正常系:
            def test_apply_buff_の_buff_が_TST_BUFF_B_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["apply_buff"], buffs=["TST_BUFF_A", "TST_BUFF_B"]
                )
                card = build_card(
                    effects=[{"ops": [{"apply_buff": {"buff": "TST_BUFF_B"}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_apply_buff_の_buff_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_buff_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["apply_buff"], buffs=["TST_BUFF_A", "TST_BUFF_B"]
                )
                card = build_card(
                    effects=[{"ops": [{"apply_buff": {"buff": "TST_UNKNOWN"}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'buff' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_op_に_heal_damage_selector_に_TST_SELECTOR_A_と_TST_SELECTOR_B_が定義されているとき:
        class Test正常系:
            def test_ops_の_selector_が_TST_SELECTOR_B_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["heal_damage"],
                    selector_keywords=["TST_SELECTOR_A", "TST_SELECTOR_B"],
                )
                card = build_card(
                    effects=[
                        {"ops": [{"heal_damage": {"selector": "TST_SELECTOR_B"}}]}
                    ]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_ops_の_selector_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_selector_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["heal_damage"],
                    selector_keywords=["TST_SELECTOR_A", "TST_SELECTOR_B"],
                )
                card = build_card(
                    effects=[{"ops": [{"heal_damage": {"selector": "TST_UNKNOWN"}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'selector' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_op_に_heal_damage_selector_の_owner_に_TST_OWNER_A_と_TST_OWNER_B_が定義されているとき:
        class Test正常系:
            def test_ops_の_selector_がキーと値の組で_owner_が_TST_OWNER_B_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["heal_damage"], selector_owners=["TST_OWNER_A", "TST_OWNER_B"]
                )
                card = build_card(
                    effects=[
                        {
                            "ops": [
                                {"heal_damage": {"selector": {"owner": "TST_OWNER_B"}}}
                            ]
                        }
                    ]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            @pytest.mark.parametrize(
                "selector",
                [
                    pytest.param(
                        {"owner": "TST_UNKNOWN"},
                        id="ops の selector がキーと値の組で、owner が TST_UNKNOWN のとき、バリデーションエラーの一覧に owner の値 TST_UNKNOWN がゲーム定数に定義されていないという誤りが含まれる",
                    ),
                    pytest.param(
                        {"owner": "TST_UNKNOWN", "pick": "choice"},
                        id="ops の selector がキーと値の組で、pick が choice で、owner が TST_UNKNOWN のとき、バリデーションエラーの一覧に owner の値 TST_UNKNOWN がゲーム定数に定義されていないという誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_effect_taxonomy, build_card, has_error, selector):
                taxonomy = build_effect_taxonomy(
                    ops=["heal_damage"], selector_owners=["TST_OWNER_A", "TST_OWNER_B"]
                )
                card = build_card(
                    effects=[{"ops": [{"heal_damage": {"selector": selector}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'owner' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_op_に_apply_buff_duration_に_TST_DURATION_A_と_TST_DURATION_B_が定義されているとき:
        class Test正常系:
            def test_apply_buff_の_duration_が_TST_DURATION_B_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["apply_buff"], durations=["TST_DURATION_A", "TST_DURATION_B"]
                )
                card = build_card(
                    effects=[{"ops": [{"apply_buff": {"duration": "TST_DURATION_B"}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_apply_buff_の_duration_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_duration_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    ops=["apply_buff"], durations=["TST_DURATION_A", "TST_DURATION_B"]
                )
                card = build_card(
                    effects=[{"ops": [{"apply_buff": {"duration": "TST_UNKNOWN"}}]}]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'duration' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_card_types_に_TstTypeA_と_TstTypeB_が定義されているとき:
        class Test正常系:
            def test_効果の_card_type_が_TstTypeB_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA", "TstTypeB"], subtypes_by_card_type={}
                )
                card = build_card(effects=[{"card_type": "TstTypeB"}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_効果の_card_type_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_card_type_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA", "TstTypeB"], subtypes_by_card_type={}
                )
                card = build_card(effects=[{"card_type": "TST_UNKNOWN"}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'card_type' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label="SH-9001 TSTカードA",
                )

    class Testゲーム定数の_card_types_に_TstTypeA_と_TstTypeB_op_に_heal_damage_が定義されているとき:
        class Test正常系:
            def test_ops_の内側の_selector_の_card_type_が_TstTypeA_のとき_バリデーションエラーの一覧が空になる(
                self, build_effect_taxonomy, build_card
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA", "TstTypeB"], ops=["heal_damage"]
                )
                card = build_card(
                    effects=[
                        {
                            "ops": [
                                {"heal_damage": {"selector": {"card_type": "TstTypeA"}}}
                            ]
                        }
                    ]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_ops_の内側の_selector_の_card_type_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_card_type_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA", "TstTypeB"], ops=["heal_damage"]
                )
                card = build_card(
                    effects=[
                        {
                            "ops": [
                                {
                                    "heal_damage": {
                                        "selector": {"card_type": "TST_UNKNOWN"}
                                    }
                                }
                            ]
                        }
                    ]
                )

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'card_type' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label=EFFECT_LABEL,
                )

    class Testゲーム定数の_subtypes_に_card_type_TstTypeA_の_subtype_として_TstSubA_と_TstSubB_が定義されているとき:
        class Test正常系:
            @pytest.mark.parametrize(
                "subtype",
                [
                    pytest.param(
                        "TstSubA",
                        id="効果の subtype が文字列の TstSubA のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        ["TstSubA", "TstSubB"],
                        id="効果の subtype が TstSubA と TstSubB を含むリストのとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, build_effect_taxonomy, build_card, subtype):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA"],
                    subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                )
                card = build_card(effects=[{"subtype": subtype}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert errors == []

        class Test異常系:
            def test_効果の_subtype_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_subtype_の値_TST_UNKNOWN_がゲーム定数に定義されていないという誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA"],
                    subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                )
                card = build_card(effects=[{"subtype": "TST_UNKNOWN"}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors,
                    f"effect 'subtype' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label="SH-9001 TSTカードA",
                )

            def test_効果の_subtype_が空のリストのとき_バリデーションエラーの一覧に_subtype_が空である誤りが含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA"],
                    subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                )
                card = build_card(effects=[{"subtype": []}])

                errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                assert has_error(
                    errors, "effect 'subtype' is empty", label="SH-9001 TSTカードA"
                )

            class Test効果の_subtype_が_TstSubA_と_TST_UNKNOWN_を含むリストのとき:
                def test_バリデーションエラーの一覧に_TST_UNKNOWN_がゲーム定数に定義されていないという誤りとして含まれる(
                    self, build_effect_taxonomy, build_card, has_error
                ):
                    taxonomy = build_effect_taxonomy(
                        card_types=["TstTypeA"],
                        subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                    )
                    card = build_card(effects=[{"subtype": ["TstSubA", "TST_UNKNOWN"]}])

                    errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                    assert has_error(
                        errors,
                        f"effect 'subtype' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                        label="SH-9001 TSTカードA",
                    )

                def test_バリデーションエラーの一覧に_TstSubA_がゲーム定数に定義されていないという誤りとして含まれない(
                    self, build_effect_taxonomy, build_card, has_error
                ):
                    taxonomy = build_effect_taxonomy(
                        card_types=["TstTypeA"],
                        subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                    )
                    card = build_card(effects=[{"subtype": ["TstSubA", "TST_UNKNOWN"]}])

                    errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

                    assert not has_error(
                        errors,
                        f"effect 'subtype' has value 'TstSubA' {UNDEFINED_VALUE_PHRASE}",
                    )

    class Testゲーム定数の_card_types_に_TstTypeA_と_TstTypeB_subtypes_に_card_type_TstTypeA_の_subtype_として_TstSubA_と_TstSubB_が定義されているとき:
        class Test異常系:
            def test_card_id_が_SH9001_のカードの効果の_card_type_が_TST_UNKNOWN_で_card_id_が_TK9002_のカードの効果の_subtype_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_SH9001_のカードの_card_type_の値_TST_UNKNOWN_の誤りと_TK9002_のカードの_subtype_の値_TST_UNKNOWN_の誤りの_2_件が含まれる(
                self, build_effect_taxonomy, build_card, has_error
            ):
                taxonomy = build_effect_taxonomy(
                    card_types=["TstTypeA", "TstTypeB"],
                    subtypes_by_card_type={"TstTypeA": ["TstSubA", "TstSubB"]},
                )
                cards = [
                    build_card(effects=[{"card_type": "TST_UNKNOWN"}]),
                    build_card(
                        card_no=2,
                        card_id="TK-9002",
                        card_name="TSTカードB",
                        const_name="TstCardB",
                        effects=[{"subtype": "TST_UNKNOWN"}],
                    ),
                ]

                errors = generate_cards.validate_effect_taxonomy(cards, taxonomy)

                assert has_error(
                    errors,
                    f"effect 'card_type' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label="SH-9001 TSTカードA",
                )
                assert has_error(
                    errors,
                    f"effect 'subtype' has value 'TST_UNKNOWN' {UNDEFINED_VALUE_PHRASE}",
                    label="TK-9002 TSTカードB",
                )

    class Test正常系:
        def test_effects_が無いカードのとき_バリデーションエラーの一覧が空になる(
            self, taxonomy, build_card
        ):
            card = build_card()

            errors = generate_cards.validate_effect_taxonomy([card], taxonomy)

            assert errors == []


class Testカード定義__ゲーム定数の読み込み:
    class Test正常系:
        def test_ゲーム定数の_card_types_が_TstTypeA_と_TstTypeB_のとき_読み込んだカードタイプが_TstTypeA_と_TstTypeB_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(card_types=["TstTypeA", "TstTypeB"])
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.card_types) == {"TstTypeA", "TstTypeB"}

        def test_ゲーム定数の_subtypes_が_card_type_TstTypeA_に_TstSubA_と_TstSubB_card_type_TstTypeB_に_TstSubC_と書かれているとき_読み込んだ_subtype_が_TstSubA_と_TstSubB_と_TstSubC_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(
                card_types=["TstTypeA", "TstTypeB"],
                subtypes={"TstTypeA": ["TstSubA", "TstSubB"], "TstTypeB": ["TstSubC"]},
            )
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set().union(*taxonomy.subtypes_by_card_type.values()) == {
                "TstSubA",
                "TstSubB",
                "TstSubC",
            }

        def test_ゲームロジック定数の_trigger_types_が_TST_TRIGGER_A_と_TST_TRIGGER_B_のとき_読み込んだ_trigger_が_TST_TRIGGER_A_と_TST_TRIGGER_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(
                trigger_types=["TST_TRIGGER_A", "TST_TRIGGER_B"]
            )

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.triggers) == {"TST_TRIGGER_A", "TST_TRIGGER_B"}

        def test_ゲームロジック定数の_effect_ops_が_TST_OP_A_と_TST_OP_B_のとき_読み込んだ_op_が_TST_OP_A_と_TST_OP_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(effect_ops=["TST_OP_A", "TST_OP_B"])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.ops) == {"TST_OP_A", "TST_OP_B"}

        def test_ゲームロジック定数の_buff_types_が_TST_BUFF_A_と_TST_BUFF_B_のとき_読み込んだ_buff_が_TST_BUFF_A_と_TST_BUFF_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(buff_types=["TST_BUFF_A", "TST_BUFF_B"])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.buffs) == {"TST_BUFF_A", "TST_BUFF_B"}

        def test_ゲームロジック定数の_effect_durations_が_TST_DURATION_A_と_TST_DURATION_B_のとき_読み込んだ_duration_が_TST_DURATION_A_と_TST_DURATION_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(
                effect_durations=["TST_DURATION_A", "TST_DURATION_B"]
            )

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.durations) == {"TST_DURATION_A", "TST_DURATION_B"}

        def test_ゲームロジック定数の_selector_keywords_が_TST_SELECTOR_A_と_TST_SELECTOR_B_のとき_読み込んだ_selector_の文字列の値が_TST_SELECTOR_A_と_TST_SELECTOR_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(
                selector_keywords=["TST_SELECTOR_A", "TST_SELECTOR_B"]
            )

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.selector_keywords) == {
                "TST_SELECTOR_A",
                "TST_SELECTOR_B",
            }

        def test_ゲームロジック定数の_player_refs_が_TST_OWNER_A_と_TST_OWNER_B_のとき_読み込んだ_selector_の_owner_の値が_TST_OWNER_A_と_TST_OWNER_B_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(player_refs=["TST_OWNER_A", "TST_OWNER_B"])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.selector_owners) == {"TST_OWNER_A", "TST_OWNER_B"}

        def test_ゲーム定数の_resource_groups_が_括り_TST_GROUP_A_は_card_type_が_TstTypeA_括り_TST_GROUP_B_は_subtypes_が_TstSubA_と_TstSubB_と書かれているとき_読み込んだ括りの定義が_TST_GROUP_A_は_card_type_が_TstTypeA_TST_GROUP_B_は_subtypes_が_TstSubA_と_TstSubB_になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(
                card_types=["TstTypeA"],
                subtypes={"TstTypeA": ["TstSubA", "TstSubB"]},
                resource_groups={
                    "TST_GROUP_A": {"card_type": "TstTypeA"},
                    "TST_GROUP_B": {"subtypes": ["TstSubA", "TstSubB"]},
                },
            )
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert taxonomy.resource_groups == {
                "TST_GROUP_A": {"card_type": "TstTypeA"},
                "TST_GROUP_B": {"subtypes": ["TstSubA", "TstSubB"]},
            }

        def test_ゲーム定数の_card_types_が空のリストのとき_読み込んだカードタイプが空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(card_types=[])
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.card_types) == set()

        def test_ゲーム定数の_subtypes_が空のとき_読み込んだ_subtype_が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(subtypes={})
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set().union(*taxonomy.subtypes_by_card_type.values()) == set()

        def test_ゲーム定数の_resource_groups_が空のとき_読み込んだ括りの定義が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants(resource_groups={})
            write_game_logic_constants()

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert taxonomy.resource_groups == {}

        def test_ゲームロジック定数の_trigger_types_が空のリストのとき_読み込んだ_trigger_が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(trigger_types=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.triggers) == set()

        def test_ゲームロジック定数の_effect_ops_が空のリストのとき_読み込んだ_op_が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(effect_ops=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.ops) == set()

        def test_ゲームロジック定数の_buff_types_が空のリストのとき_読み込んだ_buff_が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(buff_types=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.buffs) == set()

        def test_ゲームロジック定数の_effect_durations_が空のリストのとき_読み込んだ_duration_が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(effect_durations=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.durations) == set()

        def test_ゲームロジック定数の_selector_keywords_が空のリストのとき_読み込んだ_selector_の文字列の値が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(selector_keywords=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.selector_keywords) == set()

        def test_ゲームロジック定数の_player_refs_が空のリストのとき_読み込んだ_selector_の_owner_の値が空になる(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_game_constants,
            write_game_logic_constants,
        ):
            write_game_constants()
            write_game_logic_constants(player_refs=[])

            taxonomy = generate_cards.load_effect_taxonomy(
                game_constants_path, game_logic_constants_path
            )

            assert set(taxonomy.selector_owners) == set()

    class Test異常系:
        class Testゲーム定数のファイルが無いとき_生成を続けられないエラーが発生する:
            def test_エラーのメッセージにファイルの場所が含まれる(
                self, tmp_path, game_logic_constants_path, write_game_logic_constants
            ):
                write_game_logic_constants()
                missing_path = tmp_path / "missing" / "game_design_constants.yaml"

                with pytest.raises(
                    generate_cards.CardGenerationError, match=re.escape(str(missing_path))
                ):
                    generate_cards.load_effect_taxonomy(
                        missing_path, game_logic_constants_path
                    )

            def test_エラーのメッセージに_COMMON_REPO_を設定する案内が含まれる(
                self, tmp_path, game_logic_constants_path, write_game_logic_constants
            ):
                write_game_logic_constants()
                missing_path = tmp_path / "missing" / "game_design_constants.yaml"

                with pytest.raises(
                    generate_cards.CardGenerationError,
                    match=re.escape(
                        "Set COMMON_REPO to the overload-party-common checkout."
                    ),
                ):
                    generate_cards.load_effect_taxonomy(
                        missing_path, game_logic_constants_path
                    )

        @pytest.mark.parametrize(
            "missing_key",
            [
                pytest.param(
                    "card_types",
                    id="ゲーム定数に card_types が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに card_types の欠落が含まれる",
                ),
                pytest.param(
                    "subtypes",
                    id="ゲーム定数に subtypes が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに subtypes の欠落が含まれる",
                ),
                pytest.param(
                    "resource_groups",
                    id="ゲーム定数に resource_groups が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに resource_groups の欠落が含まれる",
                ),
                pytest.param(
                    "trigger_types",
                    id="ゲームロジック定数に trigger_types が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに trigger_types の欠落が含まれる",
                ),
                pytest.param(
                    "effect_ops",
                    id="ゲームロジック定数に effect_ops が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに effect_ops の欠落が含まれる",
                ),
                pytest.param(
                    "buff_types",
                    id="ゲームロジック定数に buff_types が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに buff_types の欠落が含まれる",
                ),
                pytest.param(
                    "effect_durations",
                    id="ゲームロジック定数に effect_durations が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに effect_durations の欠落が含まれる",
                ),
                pytest.param(
                    "selector_keywords",
                    id="ゲームロジック定数に selector_keywords が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに selector_keywords の欠落が含まれる",
                ),
                pytest.param(
                    "player_refs",
                    id="ゲームロジック定数に player_refs が無いとき、生成を続けられないエラーが発生し、エラーのメッセージに player_refs の欠落が含まれる",
                ),
            ],
        )
        def test_読み込む(
            self,
            game_constants_path,
            game_logic_constants_path,
            write_yaml_file,
            missing_key,
        ):
            design_constants = {"card_types": [], "subtypes": {}, "resource_groups": {}}
            logic_constants = {
                "trigger_types": ["TST_TRIGGER_A"],
                "effect_ops": ["TST_OP_A"],
                "buff_types": ["TST_BUFF_A"],
                "effect_durations": ["TST_DURATION_A"],
                "selector_keywords": ["TST_SELECTOR_A"],
                "player_refs": ["TST_OWNER_A"],
            }
            for path, constants in (
                (game_constants_path, design_constants),
                (game_logic_constants_path, logic_constants),
            ):
                write_yaml_file(
                    path,
                    {
                        key: value
                        for key, value in constants.items()
                        if key != missing_key
                    },
                )

            with pytest.raises(
                generate_cards.CardGenerationError,
                match=re.escape(f"is missing '{missing_key}'"),
            ):
                generate_cards.load_effect_taxonomy(
                    game_constants_path, game_logic_constants_path
                )

        class Testゲームロジック定数のファイルが無いとき_生成を続けられないエラーが発生する:
            def test_エラーのメッセージにファイルの場所が含まれる(
                self, tmp_path, game_constants_path, write_game_constants
            ):
                write_game_constants()
                missing_path = tmp_path / "missing" / "game_logic_constants.yaml"

                with pytest.raises(
                    generate_cards.CardGenerationError, match=re.escape(str(missing_path))
                ):
                    generate_cards.load_effect_taxonomy(
                        game_constants_path, missing_path
                    )

            def test_エラーのメッセージに_BATTLE_REPO_を設定する案内が含まれる(
                self, tmp_path, game_constants_path, write_game_constants
            ):
                write_game_constants()
                missing_path = tmp_path / "missing" / "game_logic_constants.yaml"

                with pytest.raises(
                    generate_cards.CardGenerationError,
                    match=re.escape(
                        "Set BATTLE_REPO to the overload-party-battle checkout."
                    ),
                ):
                    generate_cards.load_effect_taxonomy(
                        game_constants_path, missing_path
                    )


@pytest.fixture
def generate_json_elements(generated_json_path, read_json_file):
    def generate_cards_json_and_read_elements(cards):
        generate_cards.generate_cards_json(cards, out_path=generated_json_path)
        return read_json_file(generated_json_path)

    return generate_cards_json_and_read_elements


class Testカード定義__配布用_JSON:
    class Test正常系:
        def test_stats_が_throughput_120_availability_99_maintenance_cost_15_sla_penalty_25_のカードを出力すると_配布用_JSON_の要素の_stats_が_throughput_120_availability_99_maintenance_cost_15_sla_penalty_25_になる(
            self, build_compute_card, generate_json_elements
        ):
            elements = generate_json_elements([build_compute_card(faction="SHE")])

            assert elements[0]["stats"] == {
                "throughput": 120,
                "availability": 99,
                "maintenance_cost": 15,
                "sla_penalty": 25,
            }

        def test_card_id_が_TK9002_SH9001_NT9003_の順に定義された_3_枚のカードを出力すると_配布用_JSON_の要素の_card_id_が_NT9003_SH9001_TK9002_の順に並ぶ(
            self, build_card, generate_json_elements
        ):
            cards = [
                build_card(card_no=2, card_id="TK-9002", faction="Tenki"),
                build_card(card_no=1, card_id="SH-9001", faction="SHE"),
                build_card(card_no=3, card_id="NT-9003", faction="Neutral"),
            ]

            elements = generate_json_elements(cards)

            assert [element["card_id"] for element in elements] == [
                "NT-9003",
                "SH-9001",
                "TK-9002",
            ]

        def test_subtype_が_VM_のカードを出力すると_配布用_JSON_の要素の_subtype_が_VM_になる(
            self, build_compute_card, generate_json_elements
        ):
            elements = generate_json_elements(
                [build_compute_card(subtype="VM", faction="SHE")]
            )

            assert elements[0]["subtype"] == "VM"

        def test_subtype_が無いカードを出力すると_配布用_JSON_の要素に_subtype_が含まれない(
            self, build_card, generate_json_elements
        ):
            elements = generate_json_elements([build_card(faction="SHE")])

            assert "subtype" not in elements[0]

        def test_effect_text_が_TST効果A_のカードを出力すると_配布用_JSON_の要素の_effect_text_が_TST効果A_になる(
            self, build_card, generate_json_elements
        ):
            elements = generate_json_elements(
                [build_card(faction="SHE", effect_text="TST効果A")]
            )

            assert elements[0]["effect_text"] == "TST効果A"

        def test_effect_text_が無いカードを出力すると_配布用_JSON_の要素に_effect_text_が含まれない(
            self, build_card, generate_json_elements
        ):
            elements = generate_json_elements([build_card(faction="SHE")])

            assert "effect_text" not in elements[0]

        def test_effects_が_trigger_on_deploy_で_ops_に_draw_の_count_1_を持つカードを出力すると_配布用_JSON_の要素の_effects_が_trigger_on_deploy_で_ops_に_draw_の_count_1_を持つものになる(
            self, build_card, generate_json_elements
        ):
            effects = [{"trigger": "on_deploy", "ops": [{"draw": {"count": 1}}]}]

            elements = generate_json_elements(
                [build_card(faction="SHE", effects=effects)]
            )

            assert elements[0]["effects"] == [
                {"trigger": "on_deploy", "ops": [{"draw": {"count": 1}}]}
            ]

        def test_effects_が無いカードを出力すると_配布用_JSON_の要素に_effects_が含まれない(
            self, build_card, generate_json_elements
        ):
            elements = generate_json_elements([build_card(faction="SHE")])

            assert "effects" not in elements[0]

        def test_stats_に値が書かれていないカードを出力すると_配布用_JSON_の要素の_stats_が空のオブジェクトになる(
            self, build_card, generate_json_elements
        ):
            elements = generate_json_elements(
                [build_card(faction="SHE", stats=None)]
            )

            assert elements[0]["stats"] == {}


@pytest.fixture
def build_seed_card(build_card_of_type):
    def build_card_for_seed(card_type="Platform", **fields):
        return build_card_of_type(
            card_type, **{"faction": "SHE", "resource_label": "TST資源A", **fields}
        )

    return build_card_for_seed


@pytest.fixture
def two_cards(build_seed_card):
    return [
        build_seed_card(
            "Compute",
            subtype="VM",
            card_no=1,
            card_id="SH-9001",
            card_name="TSTカードA",
            const_name="TstCardA",
            resource_label="TST資源A",
            faction="SHE",
            resizable=True,
            elastic=False,
            restriction="limited",
            is_active=True,
        ),
        build_seed_card(
            "Platform",
            card_no=2,
            card_id="TK-9002",
            card_name="TSTカードB",
            const_name="TstCardB",
            resource_label="TST資源B",
            faction="Tenki",
            resizable=False,
            elastic=True,
            restriction="unlimited",
            is_active=False,
        ),
    ]


@pytest.fixture
def select_card_column(select_rows):
    def select_column_by_card_id(column):
        rows = select_rows(f"SELECT card_id, {column} FROM card.card_definitions")
        return {row["card_id"]: row[column] for row in rows}

    return select_column_by_card_id


class Testカード定義__初期投入_SQL:
    class Test正常系:
        class Testcard_id_が_SH9001_のカードと_TK9002_のカードの_2_枚の初期投入_SQL_を適用すると:
            def test_読み戻した_card_definitions_の_card_id_の一覧が_SH9001_TK9002_になる(
                self, two_cards, seed_path, apply_sql_file, select_rows
            ):
                generate_cards.generate_cards_seed_sql(two_cards, out_path=seed_path)

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT card_id FROM card.card_definitions ORDER BY card_id"
                )
                assert [row["card_id"] for row in rows] == ["SH-9001", "TK-9002"]

            @pytest.mark.parametrize(
                ("column", "expected"),
                [
                    pytest.param(
                        "card_name",
                        {"SH-9001": "TSTカードA", "TK-9002": "TSTカードB"},
                        id="SH-9001 の card_name が TSTカードA、TK-9002 の card_name が TSTカードB のとき、読み戻した card_name の列が SH-9001 の行は TSTカードA、TK-9002 の行は TSTカードB になる",
                    ),
                    pytest.param(
                        "resource_label",
                        {"SH-9001": "TST資源A", "TK-9002": "TST資源B"},
                        id="SH-9001 の resource_label が TST資源A、TK-9002 の resource_label が TST資源B のとき、読み戻した resource_label の列が SH-9001 の行は TST資源A、TK-9002 の行は TST資源B になる",
                    ),
                    pytest.param(
                        "faction",
                        {"SH-9001": "SHE", "TK-9002": "Tenki"},
                        id="SH-9001 の faction が SHE、TK-9002 の faction が Tenki のとき、読み戻した faction の列が SH-9001 の行は SHE、TK-9002 の行は Tenki になる",
                    ),
                    pytest.param(
                        "card_type",
                        {"SH-9001": "Compute", "TK-9002": "Platform"},
                        id="SH-9001 の card_type が Compute、TK-9002 の card_type が Platform のとき、読み戻した card_type の列が SH-9001 の行は Compute、TK-9002 の行は Platform になる",
                    ),
                    pytest.param(
                        "resizable",
                        {"SH-9001": True, "TK-9002": False},
                        id="SH-9001 の resizable が true、TK-9002 の resizable が false のとき、読み戻した resizable の列が SH-9001 の行は true、TK-9002 の行は false になる",
                    ),
                    pytest.param(
                        "elastic",
                        {"SH-9001": False, "TK-9002": True},
                        id="SH-9001 の elastic が false、TK-9002 の elastic が true のとき、読み戻した elastic の列が SH-9001 の行は false、TK-9002 の行は true になる",
                    ),
                    pytest.param(
                        "restriction",
                        {"SH-9001": "limited", "TK-9002": "unlimited"},
                        id="SH-9001 の restriction が limited、TK-9002 の restriction が unlimited のとき、読み戻した restriction の列が SH-9001 の行は limited、TK-9002 の行は unlimited になる",
                    ),
                    pytest.param(
                        "is_active",
                        {"SH-9001": True, "TK-9002": False},
                        id="SH-9001 の is_active が true、TK-9002 の is_active が false のとき、読み戻した is_active の列が SH-9001 の行は true、TK-9002 の行は false になる",
                    ),
                ],
            )
            def test_初期投入_SQL_を適用して列を読み戻す(
                self,
                two_cards,
                seed_path,
                apply_sql_file,
                select_card_column,
                column,
                expected,
            ):
                generate_cards.generate_cards_seed_sql(two_cards, out_path=seed_path)

                apply_sql_file(seed_path)

                assert select_card_column(column) == expected

        def test_subtype_が_VM_のカードの初期投入_SQL_を適用すると_subtype_の列が_VM_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            card = build_seed_card("Compute", subtype="VM")
            generate_cards.generate_cards_seed_sql([card], out_path=seed_path)

            apply_sql_file(seed_path)

            assert select_card_column("subtype") == {"SH-9001": "VM"}

        def test_subtype_が無いカードの初期投入_SQL_を適用すると_subtype_の列が_NULL_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card()], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("subtype") == {"SH-9001": None}

        def test_stats_が_throughput_120_availability_99_maintenance_cost_15_sla_penalty_25_のカードの初期投入_SQL_を適用すると_stats_の列の_JSON_が_throughput_120_availability_99_maintenance_cost_15_sla_penalty_25_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            card = build_seed_card("Compute", subtype="VM")
            generate_cards.generate_cards_seed_sql([card], out_path=seed_path)

            apply_sql_file(seed_path)

            assert select_card_column("stats") == {
                "SH-9001": {
                    "throughput": 120,
                    "availability": 99,
                    "maintenance_cost": 15,
                    "sla_penalty": 25,
                }
            }

        def test_stats_に値が書かれていないカードの初期投入_SQL_を適用すると_stats_の列が空のオブジェクトになる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(stats=None)], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("stats") == {"SH-9001": {}}

        def test_effect_text_が_TST効果A_のカードの初期投入_SQL_を適用すると_effect_text_の列が_TST効果A_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(effect_text="TST効果A")], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("effect_text") == {"SH-9001": "TST効果A"}

        def test_effect_text_が無いカードの初期投入_SQL_を適用すると_effect_text_の列が_NULL_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card()], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("effect_text") == {"SH-9001": None}

        def test_effects_が_trigger_on_deploy_で_ops_に_draw_の_count_1_を持つカードの初期投入_SQL_を適用すると_effects_の列の_JSON_が_trigger_on_deploy_で_ops_に_draw_の_count_1_を持つものになる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            effects = [{"trigger": "on_deploy", "ops": [{"draw": {"count": 1}}]}]
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(effects=effects)], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("effects") == {
                "SH-9001": [{"trigger": "on_deploy", "ops": [{"draw": {"count": 1}}]}]
            }

        def test_effects_が無いカードの初期投入_SQL_を適用すると_effects_の列が_NULL_になる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card()], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("effects") == {"SH-9001": None}

        def test_card_name_が_TST_と_カードA_の間に単引用符を_1_つ挟んだ名前TST単引用符カードAのカードの初期投入_SQL_を適用すると_card_name_の列が同じ名前TST単引用符カードAになる(
            self, build_seed_card, seed_path, apply_sql_file, select_card_column
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(card_name="TST'カードA")], out_path=seed_path
            )

            apply_sql_file(seed_path)

            assert select_card_column("card_name") == {"SH-9001": "TST'カードA"}

        class TestSH9001_の_card_name_が_TSTカードA_TK9002_の_card_name_が_TSTカードB_の_2_枚のカードの初期投入_SQL_を_2_回続けて適用すると:
            def test_読み戻した_card_name_の列が_SH9001_の行は_TSTカードA_TK9002_の行は_TSTカードB_のままになる(
                self, two_cards, seed_path, apply_sql_file, select_card_column
            ):
                generate_cards.generate_cards_seed_sql(two_cards, out_path=seed_path)

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                assert select_card_column("card_name") == {
                    "SH-9001": "TSTカードA",
                    "TK-9002": "TSTカードB",
                }

            def test_読み戻した_card_definitions_の行数が_2_になる(
                self, two_cards, seed_path, apply_sql_file, select_rows
            ):
                generate_cards.generate_cards_seed_sql(two_cards, out_path=seed_path)

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT count(*) AS row_count FROM card.card_definitions"
                )
                assert rows[0]["row_count"] == 2

        def test_card_id_が_SH9001_で_card_name_が_TSTカードA_のカードを適用済みのとき_card_name_を_TSTカードB_に変えた定義から出力し直した_SQL_を適用すると_SH9001_の行の_card_name_の列が_TSTカードB_になる(
            self,
            build_seed_card,
            seed_path,
            updated_seed_path,
            apply_sql_file,
            select_card_column,
        ):
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(card_name="TSTカードA")], out_path=seed_path
            )
            generate_cards.generate_cards_seed_sql(
                [build_seed_card(card_name="TSTカードB")], out_path=updated_seed_path
            )
            apply_sql_file(seed_path)

            apply_sql_file(updated_seed_path)

            assert select_card_column("card_name") == {"SH-9001": "TSTカードB"}


def select_summary_rows(document):
    table = next(
        table
        for block in document
        for table in block["tables"]
        if table["header"][-1] == "合計"
    )
    return {
        row[0]: dict(zip(table["header"], row, strict=True)) for row in table["rows"]
    }


@pytest.fixture
def build_listed_card(build_card_of_type):
    def build_card_for_listing(card_type="Platform", **fields):
        return build_card_of_type(card_type, **{"deploy_turns": 0, **fields})

    return build_card_for_listing


@pytest.fixture
def build_numbered_cards(build_listed_card):
    def build_cards_with_numbered_ids(prefix, count, card_type, **fields):
        return [
            build_listed_card(
                card_type, card_id=f"{prefix}-{FIRST_CARD_NUMBER + index}", **fields
            )
            for index in range(count)
        ]

    return build_cards_with_numbered_ids


@pytest.fixture
def select_card_ids_of_tables():
    def select_ids_from_every_table(document):
        return [
            row[0]
            for block in document
            for table in block["tables"]
            if table["header"][0] == "ID"
            for row in table["rows"]
        ]

    return select_ids_from_every_table


class Testカード定義__カード一覧文書:
    class Test正常系:
        def test_3_枚のカードを出力すると_文書の冒頭に全_3_枚と表示される(
            self, build_numbered_cards, render_card_list, select_opening_lines
        ):
            cards = build_numbered_cards("SH", 3, "Platform")

            document = render_card_list({"SHE": cards})

            assert "全 3 枚" in select_opening_lines(document)

        def test_カード一覧文書の題名の見出しは_Overload_Party_Card_List_になる(
            self, build_numbered_cards, render_card_list, select_title_headings
        ):
            document = render_card_list(
                {"SHE": build_numbered_cards("SH", 1, "Platform")}
            )

            assert select_title_headings(document) == ["Overload Party Card List"]

        class Test制限カードの節:
            class Testrestriction_が_limited_の_TSTカードA_semi_limited_の_TSTカードB_unlimited_の_TSTカードC_があるとき:
                @pytest.fixture
                def document(self, build_listed_card, render_card_list):
                    cards = [
                        build_listed_card(
                            card_id="SH-9001",
                            card_name="TSTカードA",
                            restriction="limited",
                        ),
                        build_listed_card(
                            card_id="SH-9002",
                            card_name="TSTカードB",
                            restriction="semi_limited",
                        ),
                        build_listed_card(
                            card_id="SH-9003",
                            card_name="TSTカードC",
                            restriction="unlimited",
                        ),
                    ]
                    return render_card_list({"SHE": cards})

                def test_制限カードの表のカード名の列が_TSTカードA_だけになる(
                    self, document, select_first_table_of_titled_block, select_table_rows
                ):
                    table = select_first_table_of_titled_block(document, "制限カード")

                    names = [row["カード名"] for row in select_table_rows(table)]

                    assert names == ["TSTカードA"]

                def test_準制限カードの表のカード名の列が_TSTカードB_だけになる(
                    self, document, select_first_table_of_titled_block, select_table_rows
                ):
                    table = select_first_table_of_titled_block(document, "準制限カード")

                    names = [row["カード名"] for row in select_table_rows(table)]

                    assert names == ["TSTカードB"]

            class Testcard_name_が_TSTカードA_で_restriction_が_limited_のカードがあるとき:
                def test_restriction_reason_が_TST理由A_のとき_制限カードの表の_TSTカードA_の理由の列が_TST理由A_になる(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_first_table_of_titled_block,
                    select_table_rows,
                ):
                    card = build_listed_card(
                        card_name="TSTカードA",
                        restriction="limited",
                        restriction_reason="TST理由A",
                    )

                    document = render_card_list({"SHE": [card]})

                    table = select_first_table_of_titled_block(document, "制限カード")
                    assert {
                        row["カード名"]: row["理由"] for row in select_table_rows(table)
                    } == {"TSTカードA": "TST理由A"}

                def test_restriction_reason_が無いとき_制限カードの表の_TSTカードA_の理由の列が空になる(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_first_table_of_titled_block,
                    select_table_rows,
                ):
                    card = build_listed_card(
                        card_name="TSTカードA", restriction="limited"
                    )

                    document = render_card_list({"SHE": [card]})

                    table = select_first_table_of_titled_block(document, "制限カード")
                    assert {
                        row["カード名"]: row["理由"] for row in select_table_rows(table)
                    } == {"TSTカードA": ""}

            class Testcard_name_が_TSTカードB_で_restriction_が_semi_limited_のカードがあるとき:
                def test_restriction_reason_が_TST理由B_のとき_準制限カードの表の_TSTカードB_の理由の列が_TST理由B_になる(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_first_table_of_titled_block,
                    select_table_rows,
                ):
                    card = build_listed_card(
                        card_name="TSTカードB",
                        restriction="semi_limited",
                        restriction_reason="TST理由B",
                    )

                    document = render_card_list({"SHE": [card]})

                    table = select_first_table_of_titled_block(document, "準制限カード")
                    assert {
                        row["カード名"]: row["理由"] for row in select_table_rows(table)
                    } == {"TSTカードB": "TST理由B"}

            def test_restriction_が_semi_limited_の_TSTカードB_と_unlimited_の_TSTカードC_だけがあるとき_制限カードの節が文書に現れず_準制限カードの節が現れる(
                self, build_listed_card, render_card_list
            ):
                cards = [
                    build_listed_card(
                        card_id="SH-9002",
                        card_name="TSTカードB",
                        restriction="semi_limited",
                    ),
                    build_listed_card(
                        card_id="SH-9003",
                        card_name="TSTカードC",
                        restriction="unlimited",
                    ),
                ]

                document = render_card_list({"SHE": cards})

                titles = [block["title"] for block in document]
                assert [
                    any(title.startswith("制限カード") for title in titles),
                    any(title.startswith("準制限カード") for title in titles),
                ] == [False, True]

            def test_restriction_が_limited_の_TSTカードA_と_unlimited_の_TSTカードC_だけがあるとき_準制限カードの節が文書に現れず_制限カードの節が現れる(
                self, build_listed_card, render_card_list
            ):
                cards = [
                    build_listed_card(
                        card_id="SH-9001",
                        card_name="TSTカードA",
                        restriction="limited",
                    ),
                    build_listed_card(
                        card_id="SH-9003",
                        card_name="TSTカードC",
                        restriction="unlimited",
                    ),
                ]

                document = render_card_list({"SHE": cards})

                titles = [block["title"] for block in document]
                assert [
                    any(title.startswith("準制限カード") for title in titles),
                    any(title.startswith("制限カード") for title in titles),
                ] == [False, True]

        class Test陣営の節:
            def test_表示名が_SHE_で_カードが_3_枚ある陣営があるとき_その陣営の節の見出しは_SHE3枚_になる(
                self, build_numbered_cards, render_card_list, select_section_titles
            ):
                document = render_card_list(
                    {"SHE": build_numbered_cards("SH", 3, "Platform")}
                )

                assert "SHE（3枚）" in select_section_titles(document)

            def test_全陣営にカードがあるとき_陣営の節が_SHE_Tenki_Sugar_Tuners_Neutral_の順に並ぶ(
                self,
                build_numbered_cards,
                render_card_list,
                select_section_titles,
                faction_names,
            ):
                prefixes = {
                    "SHE": "SH",
                    "Tenki": "TK",
                    "Sugar": "SL",
                    "Tuners": "TN",
                    "Neutral": "NT",
                }
                cards_by_faction = {
                    faction: build_numbered_cards(prefix, 1, "Platform")
                    for faction, prefix in prefixes.items()
                }

                document = render_card_list(cards_by_faction)

                faction_titles = [
                    title
                    for title in select_section_titles(document)
                    if title.startswith(tuple(faction_names))
                ]
                assert [title.split("（")[0] for title in faction_titles] == [
                    "SHE",
                    "Tenki",
                    "Sugar",
                    "Tuners",
                    "Neutral",
                ]

        class Testsubtype_による振り分け:
            class Testcard_id_が_SH9001_のカードが_SHE_にあるとき:
                @pytest.mark.parametrize(
                    ("card_type", "subtype", "category"),
                    [
                        pytest.param(
                            "Compute",
                            "VM",
                            "コンピュート系リソース",
                            id="subtype が VM のとき、「コンピュート系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Compute",
                            "Container",
                            "コンピュート系リソース",
                            id="subtype が Container のとき、「コンピュート系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Compute",
                            "Orchestrator",
                            "コンピュート系リソース",
                            id="subtype が Orchestrator のとき、「コンピュート系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Compute",
                            "Serverless",
                            "コンピュート系リソース",
                            id="subtype が Serverless のとき、「コンピュート系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Compute",
                            "AI/ML",
                            "AI/ML系リソース",
                            id="subtype が AI/ML のとき、「AI/ML系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "DataResource",
                            "Database",
                            "DB系リソース",
                            id="subtype が Database のとき、「DB系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "DataResource",
                            "CacheDB",
                            "DB系リソース",
                            id="subtype が CacheDB のとき、「DB系リソース」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "DataResource",
                            "ObjectStorage",
                            "オブジェクトストレージ",
                            id="subtype が ObjectStorage のとき、「オブジェクトストレージ」の表の ID 列が SH-9001 になる",
                        ),
                    ],
                )
                def test_カード一覧文書を出力する(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_table,
                    select_table_rows,
                    card_type,
                    subtype,
                    category,
                ):
                    card = build_listed_card(
                        card_type, subtype=subtype, card_id="SH-9001"
                    )

                    document = render_card_list({"SHE": [card]})

                    section = select_section_blocks(document, "SHE（")
                    table = select_category_table(section, category)
                    assert [row["ID"] for row in select_table_rows(table)] == [
                        "SH-9001"
                    ]

            class Testcard_id_が_SH9001_のカードだけが_SHE_にあるとき:
                @pytest.mark.parametrize(
                    ("card_type", "subtype", "category"),
                    [
                        pytest.param(
                            "Compute",
                            "VM",
                            "コンピュート系リソース",
                            id="subtype が VM のとき、SHE の節に「コンピュート系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Compute",
                            "Container",
                            "コンピュート系リソース",
                            id="subtype が Container のとき、SHE の節に「コンピュート系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Compute",
                            "Orchestrator",
                            "コンピュート系リソース",
                            id="subtype が Orchestrator のとき、SHE の節に「コンピュート系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Compute",
                            "Serverless",
                            "コンピュート系リソース",
                            id="subtype が Serverless のとき、SHE の節に「コンピュート系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Compute",
                            "AI/ML",
                            "AI/ML系リソース",
                            id="subtype が AI/ML のとき、SHE の節に「AI/ML系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "DataResource",
                            "Database",
                            "DB系リソース",
                            id="subtype が Database のとき、SHE の節に「DB系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "DataResource",
                            "CacheDB",
                            "DB系リソース",
                            id="subtype が CacheDB のとき、SHE の節に「DB系リソース」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "DataResource",
                            "ObjectStorage",
                            "オブジェクトストレージ",
                            id="subtype が ObjectStorage のとき、SHE の節に「オブジェクトストレージ」以外のカテゴリの見出しも表も現れない",
                        ),
                    ],
                )
                def test_カード一覧文書を出力する(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_blocks,
                    card_type,
                    subtype,
                    category,
                ):
                    card = build_listed_card(
                        card_type, subtype=subtype, card_id="SH-9001"
                    )

                    document = render_card_list({"SHE": [card]})

                    section = select_section_blocks(document, "SHE（")
                    category_blocks = select_category_blocks(section)
                    assert [
                        block["title"].startswith(category) for block in category_blocks
                    ] == [True]
                    assert sum(len(block["tables"]) for block in section) == 1

        class Testcard_type_による振り分け:
            class Testcard_id_が_SH9001_のカードが_SHE_にあるとき:
                @pytest.mark.parametrize(
                    ("card_type", "category"),
                    [
                        pytest.param(
                            "Platform",
                            "プラットフォーム",
                            id="card_type が Platform のとき、「プラットフォーム」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Attachment",
                            "アタッチメント",
                            id="card_type が Attachment のとき、「アタッチメント」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Strategy",
                            "ストラテジー",
                            id="card_type が Strategy のとき、「ストラテジー」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Incident",
                            "インシデント",
                            id="card_type が Incident のとき、「インシデント」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Reactive",
                            "リアクティブ",
                            id="card_type が Reactive のとき、「リアクティブ」の表の ID 列が SH-9001 になる",
                        ),
                        pytest.param(
                            "Log",
                            "ログカード",
                            id="card_type が Log のとき、「ログカード」の表の ID 列が SH-9001 になる",
                        ),
                    ],
                )
                def test_カード一覧文書を出力する(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_table,
                    select_table_rows,
                    card_type,
                    category,
                ):
                    card = build_listed_card(card_type, card_id="SH-9001")

                    document = render_card_list({"SHE": [card]})

                    section = select_section_blocks(document, "SHE（")
                    table = select_category_table(section, category)
                    assert [row["ID"] for row in select_table_rows(table)] == [
                        "SH-9001"
                    ]

            class Testcard_id_が_SH9001_のカードだけが_SHE_にあるとき:
                @pytest.mark.parametrize(
                    ("card_type", "category"),
                    [
                        pytest.param(
                            "Platform",
                            "プラットフォーム",
                            id="card_type が Platform のとき、SHE の節に「プラットフォーム」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Attachment",
                            "アタッチメント",
                            id="card_type が Attachment のとき、SHE の節に「アタッチメント」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Strategy",
                            "ストラテジー",
                            id="card_type が Strategy のとき、SHE の節に「ストラテジー」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Incident",
                            "インシデント",
                            id="card_type が Incident のとき、SHE の節に「インシデント」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Reactive",
                            "リアクティブ",
                            id="card_type が Reactive のとき、SHE の節に「リアクティブ」以外のカテゴリの見出しも表も現れない",
                        ),
                        pytest.param(
                            "Log",
                            "ログカード",
                            id="card_type が Log のとき、SHE の節に「ログカード」以外のカテゴリの見出しも表も現れない",
                        ),
                    ],
                )
                def test_カード一覧文書を出力する(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_blocks,
                    card_type,
                    category,
                ):
                    card = build_listed_card(card_type, card_id="SH-9001")

                    document = render_card_list({"SHE": [card]})

                    section = select_section_blocks(document, "SHE（")
                    category_blocks = select_category_blocks(section)
                    assert [
                        block["title"].startswith(category) for block in category_blocks
                    ] == [True]
                    assert sum(len(block["tables"]) for block in section) == 1

        def test_card_id_が_SH9002_SH9001_の順に定義された_subtype_が_VM_の_2_枚のカードがあるとき_コンピュート系リソースの表の_ID_列が_SH9001_SH9002_の順に並ぶ(
            self,
            build_listed_card,
            render_card_list,
            select_section_blocks,
            select_category_table,
            select_table_rows,
        ):
            cards = [
                build_listed_card("Compute", subtype="VM", card_id="SH-9002"),
                build_listed_card("Compute", subtype="VM", card_id="SH-9001"),
            ]

            document = render_card_list({"SHE": cards})

            section = select_section_blocks(document, "SHE（")
            table = select_category_table(section, "コンピュート系リソース")
            assert [row["ID"] for row in select_table_rows(table)] == [
                "SH-9001",
                "SH-9002",
            ]

        class Testリソースの表のタイプ列:
            class Testsubtype_が_VM_のとき:
                @pytest.mark.parametrize(
                    ("resizable", "elastic", "expected"),
                    [
                        pytest.param(
                            True,
                            False,
                            "VM (R)",
                            id="resizable が true で elastic が false のとき、タイプ列が「VM (R)」になる",
                        ),
                        pytest.param(
                            False,
                            True,
                            "VM (E)",
                            id="resizable が false で elastic が true のとき、タイプ列が「VM (E)」になる",
                        ),
                        pytest.param(
                            True,
                            True,
                            "VM (R+E)",
                            id="resizable と elastic が両方 true のとき、タイプ列が「VM (R+E)」になる",
                        ),
                        pytest.param(
                            False,
                            False,
                            "VM",
                            id="resizable と elastic が両方 false のとき、タイプ列が「VM」になる",
                        ),
                    ],
                )
                def test_カード一覧文書を出力する(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_table,
                    select_table_rows,
                    resizable,
                    elastic,
                    expected,
                ):
                    card = build_listed_card(
                        "Compute", subtype="VM", resizable=resizable, elastic=elastic
                    )

                    document = render_card_list({"SHE": [card]})

                    section = select_section_blocks(document, "SHE（")
                    table = select_category_table(section, "コンピュート系リソース")
                    assert [row["タイプ"] for row in select_table_rows(table)] == [
                        expected
                    ]

            def test_subtype_が_ObjectStorage_のとき_タイプ列がObject_Storageから始まる(
                self,
                build_listed_card,
                render_card_list,
                select_section_blocks,
                select_category_table,
                select_table_rows,
            ):
                card = build_listed_card("DataResource", subtype="ObjectStorage")

                document = render_card_list({"SHE": [card]})

                section = select_section_blocks(document, "SHE（")
                table = select_category_table(section, "オブジェクトストレージ")
                type_cells = [row["タイプ"] for row in select_table_rows(table)]
                assert [cell.startswith("Object Storage") for cell in type_cells] == [
                    True
                ]

            def test_subtype_が_CacheDB_のとき_タイプ列がCache_DBから始まる(
                self,
                build_listed_card,
                render_card_list,
                select_section_blocks,
                select_category_table,
                select_table_rows,
            ):
                card = build_listed_card("DataResource", subtype="CacheDB")

                document = render_card_list({"SHE": [card]})

                section = select_section_blocks(document, "SHE（")
                table = select_category_table(section, "DB系リソース")
                type_cells = [row["タイプ"] for row in select_table_rows(table)]
                assert [cell.startswith("Cache DB") for cell in type_cells] == [True]

        class Testリソースの表の数値列:
            class Testthroughput_が_120_availability_が_99_maintenance_cost_が_15_sla_penalty_が_25_deploy_turns_が_2_の_Compute_のカードを載せると:
                @pytest.fixture
                def rows(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_table,
                    select_table_rows,
                ):
                    card = build_listed_card(
                        "Compute",
                        subtype="VM",
                        deploy_turns=2,
                        stats={
                            "throughput": 120,
                            "availability": 99,
                            "maintenance_cost": 15,
                            "sla_penalty": 25,
                        },
                    )
                    document = render_card_list({"SHE": [card]})
                    section = select_section_blocks(document, "SHE（")
                    table = select_category_table(section, "コンピュート系リソース")
                    return select_table_rows(table)

                def test_スループット列が_120_になる(self, rows):
                    assert [row["スループット"] for row in rows] == ["120"]

                def test_可用性列が_99_になる(self, rows):
                    assert [row["可用性"] for row in rows] == ["99"]

                def test_維持コスト列が_15_になる(self, rows):
                    assert [row["維持コスト"] for row in rows] == ["15"]

                def test_デプロイターン列が_2_になる(self, rows):
                    assert [row["デプロイターン"] for row in rows] == ["2"]

                def test_SLAペナルティ列が_25_になる(self, rows):
                    assert [row["SLAペナルティ"] for row in rows] == ["25"]

            class Testyield_が_80_availability_が_99_の_DataResource_のカードを載せると:
                @pytest.fixture
                def rows(
                    self,
                    build_listed_card,
                    render_card_list,
                    select_section_blocks,
                    select_category_table,
                    select_table_rows,
                ):
                    card = build_listed_card(
                        "DataResource",
                        subtype="Database",
                        stats={
                            "yield": 80,
                            "availability": 99,
                            "maintenance_cost": 15,
                            "sla_penalty": 25,
                        },
                    )
                    document = render_card_list({"SHE": [card]})
                    section = select_section_blocks(document, "SHE（")
                    table = select_category_table(section, "DB系リソース")
                    return select_table_rows(table)

                def test_イールド列が_80_になる(self, rows):
                    assert [row["イールド"] for row in rows] == ["80"]

                def test_可用性列が_99_になる(self, rows):
                    assert [row["可用性"] for row in rows] == ["99"]

        class Test効果列:
            def test_effect_text_が無いカードのとき_効果列がダッシュになる(
                self,
                build_listed_card,
                render_card_list,
                select_section_blocks,
                select_category_table,
                select_table_rows,
            ):
                document = render_card_list({"SHE": [build_listed_card()]})

                section = select_section_blocks(document, "SHE（")
                table = select_category_table(section, "プラットフォーム")
                assert [row["効果"] for row in select_table_rows(table)] == ["—"]

            def test_effect_text_がTST効果1行目とTST効果2行目を改行でつないだ文字列のカードのとき_効果列がTST効果1行目とTST効果2行目を表の_1_行の中の改行でつないだものになる(
                self,
                build_listed_card,
                render_card_list,
                select_section_blocks,
                select_category_table,
                select_table_rows,
            ):
                card = build_listed_card(effect_text="TST効果1行目\nTST効果2行目")

                document = render_card_list({"SHE": [card]})

                section = select_section_blocks(document, "SHE（")
                table = select_category_table(section, "プラットフォーム")
                assert [row["効果"].split("<br>") for row in select_table_rows(table)] == [
                    ["TST効果1行目", "TST効果2行目"]
                ]

        def test_card_type_が_Platform_のカードを載せると_プラットフォームの表の列が_ID_カード名_効果の_3_列になる(
            self,
            build_listed_card,
            render_card_list,
            select_section_blocks,
            select_category_table,
        ):
            document = render_card_list({"SHE": [build_listed_card()]})

            section = select_section_blocks(document, "SHE（")
            table = select_category_table(section, "プラットフォーム")
            assert table["header"] == ["ID", "カード名", "効果"]

        class Testカード総数サマリ:
            class TestSHE_に_subtype_が_VM_のカードが_2_枚と_card_type_が_Platform_のカードが_1_枚_Tenki_に_subtype_が_VM_のカードが_4_枚と_card_type_が_Platform_のカードが_7_枚あるとき:
                @pytest.fixture
                def summary_rows(self, build_numbered_cards, render_card_list):
                    cards_by_faction = {
                        "SHE": [
                            *build_numbered_cards("SH", 2, "Compute", subtype="VM"),
                            *build_numbered_cards("SH", 1, "Platform"),
                        ],
                        "Tenki": [
                            *build_numbered_cards("TK", 4, "Compute", subtype="VM"),
                            *build_numbered_cards("TK", 7, "Platform"),
                        ],
                    }
                    return select_summary_rows(render_card_list(cards_by_faction))

                def test_コンピュート系リソースの行の_SHE_の列が_2_Tenki_の列が_4_になり_プラットフォームの行の_SHE_の列が_1_Tenki_の列が_7_になる(
                    self, summary_rows
                ):
                    assert [
                        summary_rows["コンピュート系リソース"]["SHE"],
                        summary_rows["コンピュート系リソース"]["Tenki"],
                        summary_rows["プラットフォーム"]["SHE"],
                        summary_rows["プラットフォーム"]["Tenki"],
                    ] == ["2", "4", "1", "7"]

                def test_コンピュート系リソースの行の合計列が_6_プラットフォームの行の合計列が_8_になる(
                    self, summary_rows
                ):
                    assert [
                        summary_rows["コンピュート系リソース"]["合計"],
                        summary_rows["プラットフォーム"]["合計"],
                    ] == ["6", "8"]

                def test_合計行の_SHE_の列が_3_Tenki_の列が_11_になる(self, summary_rows):
                    assert [summary_rows["合計"]["SHE"], summary_rows["合計"]["Tenki"]] == [
                        "3",
                        "11",
                    ]

                def test_合計行の合計列が_14_になる(self, summary_rows):
                    assert summary_rows["合計"]["合計"] == "14"


def rewrite_yaml_file(path, edit):
    content = yaml.safe_load(path.read_text(encoding="utf-8"))
    path.write_text(
        yaml.safe_dump(edit(content), allow_unicode=True, sort_keys=False),
        encoding="utf-8",
    )


def remove_card_directory(locations):
    shutil.rmtree(locations.card_directory)


def empty_every_faction_file(locations):
    for faction in locations.faction_names:
        rewrite_yaml_file(
            locations.faction_file(faction),
            lambda content: {**content, "cards": []},
        )


def remove_game_constants_file(locations):
    locations.game_constants.unlink()


def remove_game_logic_constants_file(locations):
    locations.game_logic_constants.unlink()


def remove_tenki_file(locations):
    locations.faction_file("Tenki").unlink()


def remove_faction_from_sd_file(locations):
    rewrite_yaml_file(
        locations.faction_file("SHE"),
        lambda content: {
            name: value for name, value in content.items() if name != "faction"
        },
    )


def set_tenki_file_faction_to_she(locations):
    rewrite_yaml_file(
        locations.faction_file("Tenki"),
        lambda content: {**content, "faction": "SHE"},
    )


def set_undefined_resource_group(locations):
    rewrite_yaml_file(
        locations.faction_file("SHE"),
        lambda content: {
            **content,
            "cards": [{**content["cards"][0], "effects": [{"group": "TST_GROUP_X"}]}],
        },
    )


def remove_restriction_from_first_card(locations):
    rewrite_yaml_file(
        locations.faction_file("SHE"),
        lambda content: {
            **content,
            "cards": [
                {
                    name: value
                    for name, value in content["cards"][0].items()
                    if name != "restriction"
                }
            ],
        },
    )


@pytest.fixture
def definition_locations(
    locate_faction_file, game_constants_path, game_logic_constants_path, faction_names
):
    return SimpleNamespace(
        card_directory=locate_faction_file("SHE").parent,
        faction_file=locate_faction_file,
        faction_names=faction_names,
        game_constants=game_constants_path,
        game_logic_constants=game_logic_constants_path,
        sd_file=locate_faction_file("SHE"),
        tenki_file=locate_faction_file("Tenki"),
    )


@pytest.fixture
def prepare_card_definitions(
    build_card,
    write_card_definitions,
    write_game_constants,
    write_game_logic_constants,
):
    def prepare_definitions_and_game_constants(
        cards_by_faction=None, *, resource_groups=NO_RESOURCE_GROUPS
    ):
        write_card_definitions(cards_by_faction or {"SHE": [build_card()]})
        write_game_constants(resource_groups=resource_groups)
        write_game_logic_constants()

    return prepare_definitions_and_game_constants


@pytest.fixture
def two_card_definitions(build_card):
    return {
        "SHE": [build_card()],
        "Tenki": [
            build_card(
                card_no=2,
                card_id="TK-9002",
                card_name="TSTカードB",
                const_name="TstCardB",
            )
        ],
    }


class Testカード定義__生成コマンドの終了コード:
    class Test正常系:
        def test_card_id_が_SH9001_のカード_1_枚とゲーム定数があるとき_実行すると_終了コードが_0_になる(
            self, prepare_card_definitions, run_generator
        ):
            prepare_card_definitions()

            result = run_generator("generate_cards.py")

            assert result.returncode == 0

    class Test異常系:
        class Test実行すると:
            @pytest.mark.parametrize(
                ("steps", "run_options"),
                [
                    pytest.param(
                        [remove_card_directory],
                        {},
                        id="カード定義のディレクトリが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [empty_every_faction_file],
                        {},
                        id="カードが 1 枚も無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [remove_game_constants_file],
                        {},
                        id="ゲーム定数のファイルが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [],
                        RUN_WITHOUT_COMMON_REPO,
                        id="環境変数 COMMON_REPO が設定されていないとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [remove_game_logic_constants_file],
                        {},
                        id="ゲームロジック定数のファイルが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [],
                        RUN_WITHOUT_BATTLE_REPO,
                        id="環境変数 BATTLE_REPO が設定されていないとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [remove_tenki_file],
                        {},
                        id="5 つの陣営の定義ファイルのうち tenki.yaml が無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [remove_faction_from_sd_file],
                        {},
                        id="sd.yaml に faction が書かれていないとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [set_tenki_file_faction_to_she],
                        {},
                        id="tenki.yaml の faction が SHE のとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [set_undefined_resource_group],
                        {},
                        id="効果の group が定義の無い TST_GROUP_X のとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [remove_restriction_from_first_card],
                        {},
                        id="card_id が SH-9001 のカードの restriction が無いとき、終了コードが 1 になる",
                    ),
                ],
            )
            def test_実行する(
                self,
                definition_locations,
                prepare_card_definitions,
                run_generator,
                steps,
                run_options,
            ):
                prepare_card_definitions()
                for step in steps:
                    step(definition_locations)

                result = run_generator("generate_cards.py", **run_options)

                assert result.returncode == 1


class Testカード定義__生成コマンドの標準エラー出力:
    class Test正常系:
        def test_3_枚のカードの有効な定義とゲーム定数があるとき_実行すると_標準エラー出力のGeneratedの行のうち_配布用_JSONカード一覧文書初期投入_SQL_を書いた_3_行が_どれも_3_枚を示す(
            self,
            build_card,
            prepare_card_definitions,
            run_generator,
            read_generated_counts,
        ):
            prepare_card_definitions(
                {
                    "SHE": [build_card()],
                    "Tenki": [
                        build_card(
                            card_no=2,
                            card_id="TK-9002",
                            card_name="TSTカードB",
                            const_name="TstCardB",
                        )
                    ],
                    "Sugar": [
                        build_card(
                            card_no=3,
                            card_id="SL-9003",
                            card_name="TSTカードC",
                            const_name="TstCardC",
                        )
                    ],
                }
            )

            result = run_generator("generate_cards.py")

            assert read_generated_counts(result.stderr, "cards") == {
                "cards_gen.json": "3",
                "CARDS.md": "3",
                "cards_seed.sql": "3",
            }

    class Test異常系:
        @pytest.mark.parametrize(
            ("steps", "run_options", "expected_fragments"),
            [
                pytest.param(
                    [remove_card_directory],
                    {},
                    ("ERROR: YAML directory not found: {card_directory}",),
                    id="カード定義のディレクトリが無いとき、標準エラー出力に「ERROR: YAML directory not found: 」に続けてそのディレクトリの場所が表示される",
                ),
                pytest.param(
                    [empty_every_faction_file],
                    {},
                    ("ERROR: No cards loaded from YAML files",),
                    id="カードが 1 枚も無いとき、標準エラー出力に「ERROR: No cards loaded from YAML files」が表示される",
                ),
                pytest.param(
                    [remove_game_constants_file],
                    {},
                    ("ERROR: game design constants not found at {game_constants}",),
                    id="ゲーム定数のファイルが無いとき、標準エラー出力に「ERROR: game design constants not found at 」に続けてそのファイルの場所が表示される",
                ),
                pytest.param(
                    [],
                    RUN_WITHOUT_COMMON_REPO,
                    (
                        "ERROR: COMMON_REPO is not set. Set COMMON_REPO to the overload-party-common checkout.",
                    ),
                    id="環境変数 COMMON_REPO が設定されていないとき、標準エラー出力に「ERROR: COMMON_REPO is not set. Set COMMON_REPO to the overload-party-common checkout.」が表示される",
                ),
                pytest.param(
                    [remove_game_logic_constants_file],
                    {},
                    ("ERROR: game logic constants not found at {game_logic_constants}",),
                    id="ゲームロジック定数のファイルが無いとき、標準エラー出力に「ERROR: game logic constants not found at 」に続けてそのファイルの場所が表示される",
                ),
                pytest.param(
                    [],
                    RUN_WITHOUT_BATTLE_REPO,
                    (
                        "ERROR: BATTLE_REPO is not set. Set BATTLE_REPO to the overload-party-battle checkout.",
                    ),
                    id="環境変数 BATTLE_REPO が設定されていないとき、標準エラー出力に「ERROR: BATTLE_REPO is not set. Set BATTLE_REPO to the overload-party-battle checkout.」が表示される",
                ),
                pytest.param(
                    [remove_tenki_file],
                    {},
                    ("ERROR: faction definition file not found: {tenki_file}",),
                    id="5 つの陣営の定義ファイルのうち tenki.yaml が無いとき、標準エラー出力に「ERROR: faction definition file not found: 」に続けて tenki.yaml の場所が表示される",
                ),
                pytest.param(
                    [remove_faction_from_sd_file],
                    {},
                    ("{sd_file}", "missing required field 'faction'"),
                    id="sd.yaml に faction が書かれていないとき、標準エラー出力に sd.yaml の場所に続けて「missing required field 'faction'」が表示される",
                ),
                pytest.param(
                    [set_tenki_file_faction_to_she],
                    {},
                    (
                        "{tenki_file}",
                        "faction 'SHE' does not match the faction 'Tenki' of this file",
                    ),
                    id="tenki.yaml の faction が SHE のとき、標準エラー出力に tenki.yaml の場所に続けて「faction 'SHE' does not match the faction 'Tenki' of this file」が表示される",
                ),
                pytest.param(
                    [set_undefined_resource_group],
                    {},
                    (
                        "ERROR: SH-9001 TSTカードA: unknown resource group 'TST_GROUP_X'",
                    ),
                    id="card_id が SH-9001 で card_name が TSTカードA のカードの効果の group が定義の無い TST_GROUP_X のとき、標準エラー出力に「ERROR: SH-9001 TSTカードA: unknown resource group 'TST_GROUP_X'」が表示される",
                ),
            ],
        )
        def test_実行する(
            self,
            definition_locations,
            prepare_card_definitions,
            run_generator,
            steps,
            run_options,
            expected_fragments,
        ):
            prepare_card_definitions()
            for step in steps:
                step(definition_locations)

            result = run_generator("generate_cards.py", **run_options)

            expected_pattern = ".*".join(
                re.escape(fragment.format(**vars(definition_locations)))
                for fragment in expected_fragments
            )
            assert re.search(expected_pattern, result.stderr)

        class Testcard_no_が_1_で_card_name_が_TSTカードA_のカードに_restriction_が無く_card_no_が_2_で_card_name_が_TSTカードB_のカードに_is_active_が無いとき:
            @pytest.fixture
            def stderr(self, build_card, prepare_card_definitions, run_generator):
                prepare_card_definitions(
                    {
                        "SHE": [
                            build_card(omit=("restriction",)),
                            build_card(
                                omit=("is_active",),
                                card_no=2,
                                card_id="SH-9002",
                                card_name="TSTカードB",
                                const_name="TstCardB",
                            ),
                        ]
                    }
                )
                return run_generator("generate_cards.py").stderr

            def test_標準エラー出力にValidation_failed_with_2_errorsが表示される(
                self, stderr
            ):
                assert "Validation failed with 2 error(s)" in stderr

            def test_標準エラー出力に1_TSTカードA_missing_required_field_restrictionと2_TSTカードB_missing_required_field_is_activeの_2_件が表示される(
                self, stderr
            ):
                assert "#1 TSTカードA: missing required field 'restriction'" in stderr
                assert "#2 TSTカードB: missing required field 'is_active'" in stderr

        class Testcard_no_が_1_で_card_id_が_SH9001_で_card_name_が_TSTカードA_のカードの_restriction_が無く_その効果の_subtype_が_TST_UNKNOWN_のとき:
            @pytest.fixture
            def stderr(self, build_card, prepare_card_definitions, run_generator):
                prepare_card_definitions(
                    {
                        "SHE": [
                            build_card(
                                omit=("restriction",),
                                effects=[{"subtype": "TST_UNKNOWN"}],
                            )
                        ]
                    }
                )
                return run_generator("generate_cards.py").stderr

            def test_標準エラー出力に1_TSTカードA_missing_required_field_restrictionが表示される(
                self, stderr
            ):
                assert "#1 TSTカードA: missing required field 'restriction'" in stderr

            def test_標準エラー出力にSH9001_TSTカードA_effect_subtype_has_value_TST_UNKNOWN_that_is_not_defined_in_the_game_constantsが表示される(
                self, stderr
            ):
                assert (
                    "SH-9001 TSTカードA: effect 'subtype' has value 'TST_UNKNOWN' "
                    "that is not defined in the game constants"
                ) in stderr

        def test_card_id_が_SH9001_で_card_name_が_TSTカードA_のカードの効果の_trigger_が_TST_UNKNOWN_のとき_標準エラー出力にSH9001_TSTカードA_effect_trigger_has_value_TST_UNKNOWN_that_is_not_defined_in_the_game_constantsが表示される(
            self, build_card, prepare_card_definitions, run_generator
        ):
            prepare_card_definitions(
                {"SHE": [build_card(effects=[{"trigger": "TST_UNKNOWN"}])]}
            )

            result = run_generator("generate_cards.py")

            assert (
                "SH-9001 TSTカードA: effect 'trigger' has value 'TST_UNKNOWN' "
                "that is not defined in the game constants"
            ) in result.stderr

        def test_括り_TST_GROUP_A_の定義が_subtypes_の_VM_と_TST_UNKNOWN_で_card_id_が_SH9001_で_card_name_が_TSTカードA_のカードの効果が_group_に_TST_GROUP_A_を指定しているとき_標準エラー出力にSH9001_TSTカードA_effect_subtype_has_value_TST_UNKNOWN_that_is_not_defined_in_the_game_constantsが表示される(
            self, build_card, prepare_card_definitions, run_generator
        ):
            prepare_card_definitions(
                {"SHE": [build_card(effects=[{"group": "TST_GROUP_A"}])]},
                resource_groups={"TST_GROUP_A": {"subtypes": ["VM", "TST_UNKNOWN"]}},
            )

            result = run_generator("generate_cards.py")

            assert (
                "SH-9001 TSTカードA: effect 'subtype' has value 'TST_UNKNOWN' "
                "that is not defined in the game constants"
            ) in result.stderr


@pytest.fixture
def read_generated_card_ids(locate_output_file, read_json_file):
    def read_card_ids_of_generated_json():
        return [
            element["card_id"]
            for element in read_json_file(locate_output_file("cards_json"))
        ]

    return read_card_ids_of_generated_json


@pytest.fixture
def read_document_card_ids(
    locate_output_file, parse_markdown_file, select_card_ids_of_tables
):
    def read_card_ids_of_generated_document():
        document = parse_markdown_file(locate_output_file("cards_document"))
        return select_card_ids_of_tables(document)

    return read_card_ids_of_generated_document


@pytest.fixture
def read_applied_card_ids(locate_output_file, apply_sql_file, select_rows):
    def read_card_ids_after_applying_seed_sql():
        apply_sql_file(locate_output_file("cards_seed"))
        rows = select_rows("SELECT card_id FROM card.card_definitions ORDER BY card_id")
        return [row["card_id"] for row in rows]

    return read_card_ids_after_applying_seed_sql


@pytest.fixture
def read_generated_effects(locate_output_file, read_json_file):
    def read_effects_of_first_generated_card():
        elements = read_json_file(locate_output_file("cards_json"))
        return elements[0]["effects"]

    return read_effects_of_first_generated_card


class Testカード定義__生成コマンドの出力:
    class Test正常系:
        class Testcard_id_が_SH9001_のカードと_TK9002_のカードの_2_枚の有効な定義とゲーム定数があるとき:
            class Test実行すると:
                def test_配布用_JSON_の_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)

                    run_generator("generate_cards.py")

                    assert read_generated_card_ids() == EXPECTED_CARD_IDS

                def test_カード一覧文書の表の_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_document_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)

                    run_generator("generate_cards.py")

                    assert read_document_card_ids() == EXPECTED_CARD_IDS

                def test_作られた初期投入_SQL_を適用して読み戻した_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_applied_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)

                    run_generator("generate_cards.py")

                    assert read_applied_card_ids() == EXPECTED_CARD_IDS

            class Test出力先のディレクトリが無いときに実行すると:
                def test_出力先のディレクトリ内に配布用_JSONカード一覧文書初期投入_SQL_の_3_つが作られる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    remove_output_directories,
                    run_generator,
                    locate_output_file,
                ):
                    prepare_card_definitions(two_card_definitions)
                    remove_output_directories()

                    run_generator("generate_cards.py")

                    assert [
                        locate_output_file(output_key).exists()
                        for output_key in GENERATED_CARD_OUTPUT_KEYS
                    ] == [True, True, True]

                def test_作られた配布用_JSON_の_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    remove_output_directories,
                    run_generator,
                    read_generated_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)
                    remove_output_directories()

                    run_generator("generate_cards.py")

                    assert read_generated_card_ids() == EXPECTED_CARD_IDS

            class Test続けて_2_回実行すると:
                def test_2_回目の実行で作られた配布用_JSON_の_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)
                    run_generator("generate_cards.py")

                    second_result = run_generator("generate_cards.py")

                    assert second_result.returncode == 0
                    assert read_generated_card_ids() == EXPECTED_CARD_IDS

                def test_2_回目の実行で作られたカード一覧文書の表の_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_document_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)
                    run_generator("generate_cards.py")

                    second_result = run_generator("generate_cards.py")

                    assert second_result.returncode == 0
                    assert read_document_card_ids() == EXPECTED_CARD_IDS

                def test_2_回目の実行で作られた初期投入_SQL_を適用して読み戻した_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                    read_applied_card_ids,
                ):
                    prepare_card_definitions(two_card_definitions)
                    run_generator("generate_cards.py")

                    second_result = run_generator("generate_cards.py")

                    assert second_result.returncode == 0
                    assert read_applied_card_ids() == EXPECTED_CARD_IDS

        class Test効果が_group_に_COMMON_REPO_が指すゲーム定数で定義された括りを指定しているとき:
            class Test括り_TST_GROUP_A_が_card_type_の_Compute_で定義され_効果の_group_が_TST_GROUP_A_のとき:
                def test_実行すると_配布用_JSON_のその効果の_card_type_が_Compute_になる(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_effects,
                ):
                    prepare_card_definitions(
                        {"SHE": [build_card(effects=[{"group": "TST_GROUP_A"}])]},
                        resource_groups={"TST_GROUP_A": {"card_type": "Compute"}},
                    )

                    run_generator("generate_cards.py")

                    assert read_generated_effects()[0]["card_type"] == "Compute"

                def test_実行すると_配布用_JSON_のその効果に_group_が含まれない(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_effects,
                ):
                    prepare_card_definitions(
                        {"SHE": [build_card(effects=[{"group": "TST_GROUP_A"}])]},
                        resource_groups={"TST_GROUP_A": {"card_type": "Compute"}},
                    )

                    run_generator("generate_cards.py")

                    assert "group" not in read_generated_effects()[0]

                def test_効果に_owner_が_opponent_も書かれているとき_実行すると_配布用_JSON_のその効果の_owner_が_opponent_になる(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_effects,
                ):
                    effects = [{"group": "TST_GROUP_A", "owner": "opponent"}]
                    prepare_card_definitions(
                        {"SHE": [build_card(effects=effects)]},
                        resource_groups={"TST_GROUP_A": {"card_type": "Compute"}},
                    )

                    run_generator("generate_cards.py")

                    assert read_generated_effects()[0]["owner"] == "opponent"

            def test_括り_TST_GROUP_B_が_subtypes_の_VM_と_Container_で定義され_効果の_group_が_TST_GROUP_B_のとき_実行すると_配布用_JSON_のその効果の_subtype_が_VM_Container_のリストになる(
                self,
                build_card,
                prepare_card_definitions,
                run_generator,
                read_generated_effects,
            ):
                prepare_card_definitions(
                    {"SHE": [build_card(effects=[{"group": "TST_GROUP_B"}])]},
                    resource_groups={"TST_GROUP_B": {"subtypes": ["VM", "Container"]}},
                )

                run_generator("generate_cards.py")

                assert read_generated_effects()[0]["subtype"] == ["VM", "Container"]

            class Test効果の位置:
                def test_括り_TST_GROUP_A_が_card_type_の_Compute_で定義されているとき_ops_の内側の_selector_の_group_に_TST_GROUP_A_を指定して実行すると_配布用_JSON_のその_selector_の_card_type_が_Compute_になる(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_effects,
                ):
                    effects = [
                        {
                            "ops": [
                                {"heal_damage": {"selector": {"group": "TST_GROUP_A"}}}
                            ]
                        }
                    ]
                    prepare_card_definitions(
                        {"SHE": [build_card(effects=effects)]},
                        resource_groups={"TST_GROUP_A": {"card_type": "Compute"}},
                    )

                    run_generator("generate_cards.py")

                    selector = read_generated_effects()[0]["ops"][0]["heal_damage"][
                        "selector"
                    ]
                    assert selector["card_type"] == "Compute"

                def test_括り_TST_GROUP_A_が_card_type_の_Compute_括り_TST_GROUP_C_が_card_type_の_DataResource_で定義されているとき_カードの_1_つ目の効果の_group_が_TST_GROUP_A_2_つ目の効果の_group_が_TST_GROUP_C_のとき_実行すると_配布用_JSON_の_1_つ目の効果の_card_type_が_Compute_2_つ目の効果の_card_type_が_DataResource_になる(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    read_generated_effects,
                ):
                    effects = [{"group": "TST_GROUP_A"}, {"group": "TST_GROUP_C"}]
                    prepare_card_definitions(
                        {"SHE": [build_card(effects=effects)]},
                        resource_groups={
                            "TST_GROUP_A": {"card_type": "Compute"},
                            "TST_GROUP_C": {"card_type": "DataResource"},
                        },
                    )

                    run_generator("generate_cards.py")

                    assert [
                        effect["card_type"] for effect in read_generated_effects()
                    ] == ["Compute", "DataResource"]

        def test_効果が_trigger_on_deploy_で_ops_に_gain_budget_の_target_myself_amount_400_を持ち_group_を使わないカードを定義して実行すると_配布用_JSON_のその効果が_trigger_on_deploy_で_ops_に_gain_budget_の_target_myself_amount_400_だけを持つものになる(
            self,
            build_card,
            prepare_card_definitions,
            run_generator,
            read_generated_effects,
        ):
            effects = [
                {
                    "trigger": "on_deploy",
                    "ops": [{"gain_budget": {"target": "myself", "amount": 400}}],
                }
            ]
            prepare_card_definitions({"SHE": [build_card(effects=effects)]})

            run_generator("generate_cards.py")

            assert read_generated_effects() == [
                {
                    "trigger": "on_deploy",
                    "ops": [{"gain_budget": {"target": "myself", "amount": 400}}],
                }
            ]

        class Test陣営ファイルの取り込み:
            def test_tenkiyaml_に_カードの_faction_を書かない_card_id_が_TK9002_のカードを書いたとき_配布用_JSON_の_TK9002_の_faction_が_Tenki_になる(
                self,
                build_card,
                prepare_card_definitions,
                run_generator,
                locate_output_file,
                read_json_file,
            ):
                prepare_card_definitions({"Tenki": [build_card(card_id="TK-9002")]})

                run_generator("generate_cards.py")

                elements = read_json_file(locate_output_file("cards_json"))
                assert {element["card_id"]: element["faction"] for element in elements} == {
                    "TK-9002": "Tenki"
                }

            def test_sdyaml_に_card_id_が_SH9001_のカードを書いたとき_配布用_JSON_の_SH9001_の_faction_が_SHE_になる(
                self,
                build_card,
                prepare_card_definitions,
                run_generator,
                locate_output_file,
                read_json_file,
            ):
                prepare_card_definitions({"SHE": [build_card(card_id="SH-9001")]})

                run_generator("generate_cards.py")

                elements = read_json_file(locate_output_file("cards_json"))
                assert {element["card_id"]: element["faction"] for element in elements} == {
                    "SH-9001": "SHE"
                }

    class Test異常系:
        class Testcard_id_が_SH9001_のカードの_restriction_が無い定義のとき:
            class Test出力先に何も無い状態で実行すると:
                @pytest.mark.parametrize(
                    "output_key",
                    [
                        pytest.param("cards_json", id="配布用 JSON が作られない"),
                        pytest.param("cards_document", id="カード一覧文書が作られない"),
                        pytest.param("cards_seed", id="初期投入 SQL が作られない"),
                    ],
                )
                def test_実行する(
                    self,
                    build_card,
                    prepare_card_definitions,
                    run_generator,
                    locate_output_file,
                    output_key,
                ):
                    prepare_card_definitions(
                        {"SHE": [build_card(omit=("restriction",))]}
                    )

                    run_generator("generate_cards.py")

                    assert not locate_output_file(output_key).exists()

        class Testcard_id_が_SH9001_と_TK9002_の定義で出力を作ったあと_card_id_が_SH9003_のカードの_restriction_が無い定義に変えたとき:
            class Test既存の出力があるときに実行すると:
                @pytest.fixture
                def rerun_generator_after_invalidating_third_card(
                    self,
                    build_card,
                    two_card_definitions,
                    prepare_card_definitions,
                    run_generator,
                ):
                    prepare_card_definitions(two_card_definitions)
                    run_generator("generate_cards.py")
                    prepare_card_definitions(
                        {
                            "SHE": [
                                *two_card_definitions["SHE"],
                                build_card(
                                    omit=("restriction",),
                                    card_no=3,
                                    card_id="SH-9003",
                                    card_name="TSTカードC",
                                    const_name="TstCardC",
                                ),
                            ],
                            "Tenki": two_card_definitions["Tenki"],
                        }
                    )
                    run_generator("generate_cards.py")

                def test_配布用_JSON_の_card_id_の一覧が_SH9001_TK9002_のままになる(
                    self,
                    rerun_generator_after_invalidating_third_card,
                    read_generated_card_ids,
                ):
                    assert read_generated_card_ids() == EXPECTED_CARD_IDS

                def test_カード一覧文書の表の_card_id_の一覧が_SH9001_TK9002_のままになる(
                    self,
                    rerun_generator_after_invalidating_third_card,
                    read_document_card_ids,
                ):
                    assert read_document_card_ids() == EXPECTED_CARD_IDS

                def test_既存の初期投入_SQL_を適用して読み戻した_card_id_の一覧が_SH9001_TK9002_になる(
                    self,
                    rerun_generator_after_invalidating_third_card,
                    read_applied_card_ids,
                ):
                    assert read_applied_card_ids() == EXPECTED_CARD_IDS
