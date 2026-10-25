import re

import generate_products
import pytest

PRODUCT_ID_PATTERN = re.compile(r"PD-\d{4}")
EXPECTED_PRODUCT_IDS = ["PD-9001", "PD-9002", "PD-9003", "PD-9004"]
EXPECTED_INITIATIVE_IDS = [
    "IN-9001",
    "IN-9002",
    "IN-9003",
    "IN-9004",
    "IN-9005",
    "IN-9006",
    "IN-9007",
    "IN-9008",
]
GENERATED_OUTPUT_KEYS = [
    "products_json",
    "initiatives_json",
    "products_seed",
    "initiatives_seed",
    "products_document",
]
PRODUCT_FACTIONS_IN_ORDER = ["SHE", "Tenki", "Sugar", "Tuners"]


def override_first_item_fields(items, **fields):
    return [{**items[0], **fields}, *items[1:]]


def override_last_item_fields(items, **fields):
    return [*items[:-1], {**items[-1], **fields}]


def drop_field_of_first_item(items, field):
    return [
        {name: value for name, value in items[0].items() if name != field},
        *items[1:],
    ]


@pytest.fixture
def two_products(build_product):
    return [
        build_product(
            product_id="PD-9001",
            faction="SHE",
            product_name="TSTプロダクトA",
            is_active=True,
        ),
        build_product(
            product_id="PD-9002",
            faction="Tenki",
            product_name="TSTプロダクトB",
            is_active=False,
        ),
    ]


@pytest.fixture
def two_initiatives(build_initiative, insert_product_row):
    insert_product_row("PD-9001", "SHE")
    insert_product_row("PD-9002", "Tenki")
    return [
        build_initiative(
            initiative_id="IN-9001",
            product_id="PD-9001",
            kind="routine",
            name="TST施策A",
            insight_cost=100,
            effect_text="TST効果A",
            is_active=True,
            effect={"ops": [{"draw": {"count": 1}}]},
        ),
        build_initiative(
            initiative_id="IN-9002",
            product_id="PD-9002",
            kind="special",
            name="TST施策B",
            insight_cost=250,
            effect_text="TST効果B",
            is_active=False,
            effect={"ops": [{"gain_budget": {"target": "myself", "amount": 400}}]},
        ),
    ]


class Testプロダクト__プロダクトと施策の定義の検証:
    class Test正常系:
        class Testほかの項目がすべて有効なとき:
            @pytest.mark.parametrize(
                "edit",
                [
                    pytest.param(
                        lambda products, initiatives: (products, initiatives),
                        id="SHE, Tenki, Sugar, Tuners にプロダクトが 1 件ずつあり、各プロダクトに routine の施策と special の施策が 1 件ずつあるとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            [
                                *products,
                                {
                                    **products[0],
                                    "product_id": "PD-9005",
                                    "product_name": "TSTプロダクトE",
                                },
                            ],
                            [
                                *initiatives,
                                {
                                    **initiatives[0],
                                    "initiative_id": "IN-9009",
                                    "product_id": "PD-9005",
                                    "name": "TST施策I",
                                },
                                {
                                    **initiatives[1],
                                    "initiative_id": "IN-9010",
                                    "product_id": "PD-9005",
                                    "name": "TST施策J",
                                },
                            ],
                        ),
                        id="同じ陣営に 2 件のプロダクトがあるとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            [
                                *initiatives,
                                {
                                    **initiatives[0],
                                    "initiative_id": "IN-9009",
                                    "name": "TST施策I",
                                },
                            ],
                        ),
                        id="1 つのプロダクトにルーチンの施策が 2 件あるとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            [
                                *initiatives,
                                {
                                    **initiatives[1],
                                    "initiative_id": "IN-9009",
                                    "name": "TST施策I",
                                },
                            ],
                        ),
                        id="1 つのプロダクトにスペシャルの施策が 2 件あるとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            override_first_item_fields(products, is_active=True),
                            initiatives,
                        ),
                        id="プロダクトの is_active が true のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            override_first_item_fields(products, is_active=False),
                            initiatives,
                        ),
                        id="プロダクトの is_active が false のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            override_first_item_fields(initiatives, is_active=True),
                        ),
                        id="施策の is_active が true のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            override_first_item_fields(initiatives, is_active=False),
                        ),
                        id="施策の is_active が false のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            override_first_item_fields(initiatives, insight_cost=0),
                        ),
                        id="施策の insight_cost が 0 のとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            override_first_item_fields(
                                initiatives,
                                effect={"ops": [{"draw": {"count": 1}}]},
                            ),
                        ),
                        id="施策の effect が ops だけを持つとき、バリデーションエラーの一覧が空になる",
                    ),
                    pytest.param(
                        lambda products, initiatives: (
                            products,
                            override_first_item_fields(
                                initiatives, effect={"custom": "TST_CUSTOM"}
                            ),
                        ),
                        id="施策の effect が custom だけを持つとき、バリデーションエラーの一覧が空になる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, edit):
                products, initiatives = edit(*build_catalog())

                errors = generate_products.validate_products_and_initiatives(
                    products, initiatives
                )

                assert errors == []

    class Test異常系:
        class Testプロダクトの必須項目:
            @pytest.mark.parametrize(
                ("field", "label"),
                [
                    pytest.param(
                        "product_id",
                        "",
                        id="product_id が無いとき、バリデーションエラーの一覧に product_id の欠落が含まれる",
                    ),
                    pytest.param(
                        "faction",
                        "PD-9001 TSTプロダクトA",
                        id="faction が無いとき、バリデーションエラーの一覧に faction の欠落が含まれる",
                    ),
                    pytest.param(
                        "product_name",
                        "",
                        id="product_name が無いとき、バリデーションエラーの一覧に product_name の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, field, label):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    drop_field_of_first_item(products, field), initiatives
                )

                assert has_error(
                    errors, f"missing required field '{field}'", label=label
                )

        class Testproduct_id_の形式が誤っているとき:
            @pytest.mark.parametrize(
                "product_id",
                [
                    pytest.param(
                        "PD-901",
                        id="product_id が PD-901 のとき、バリデーションエラーの一覧に product_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "PD-90001",
                        id="product_id が PD-90001 のとき、バリデーションエラーの一覧に product_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "XX-9001",
                        id="product_id が XX-9001 のとき、バリデーションエラーの一覧に product_id の形式の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, product_id):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    override_first_item_fields(products, product_id=product_id),
                    initiatives,
                )

                assert has_error(
                    errors,
                    "must match PD-NNNN format",
                    label=f"{product_id} TSTプロダクトA",
                )

        def test_2_件のプロダクトの_product_id_がどちらも_PD9001_のとき_バリデーションエラーの一覧に_product_id_の重複が含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()
            duplicated = [products[0], {**products[1], "product_id": "PD-9001"}]

            errors = generate_products.validate_products_and_initiatives(
                [*duplicated, *products[2:]], initiatives
            )

            assert has_error(
                errors, "duplicate product_id", label="PD-9001 TSTプロダクトB"
            )

        class Testプロダクトの_faction_が不正なとき:
            @pytest.mark.parametrize(
                ("faction", "phrase"),
                [
                    pytest.param(
                        "TST_UNKNOWN",
                        "invalid faction 'TST_UNKNOWN'",
                        id="TST_UNKNOWN のとき、バリデーションエラーの一覧に faction の値 TST_UNKNOWN の誤りが含まれる",
                    ),
                    pytest.param(
                        "Neutral",
                        "invalid faction 'Neutral'",
                        id="Neutral のとき、バリデーションエラーの一覧に faction の値の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, faction, phrase):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    override_first_item_fields(products, faction=faction), initiatives
                )

                assert has_error(errors, phrase, label="PD-9001 TSTプロダクトA")

        def test_プロダクトの_is_active_が文字列の_yes_のとき_バリデーションエラーの一覧に_is_active_が真偽値でない誤りが含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()

            errors = generate_products.validate_products_and_initiatives(
                override_first_item_fields(products, is_active="yes"), initiatives
            )

            assert has_error(
                errors,
                "is_active must be a boolean",
                label="PD-9001 TSTプロダクトA",
            )

        def test_SHE_にプロダクトが無いとき_バリデーションエラーの一覧にプロダクトを持たない陣営として_SHE_が含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()
            without_she_initiatives = [
                initiative
                for initiative in initiatives
                if initiative["product_id"] != "PD-9001"
            ]

            errors = generate_products.validate_products_and_initiatives(
                products[1:], without_she_initiatives
            )

            assert has_error(errors, "factions without a product", "SHE")

        def test_ルーチンの施策が_1_件も無いとき_バリデーションエラーの一覧にプロダクトのルーチンの施策の欠落が含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()

            errors = generate_products.validate_products_and_initiatives(
                products, initiatives[1:]
            )

            assert has_error(
                errors,
                "must have at least one 'routine' initiative",
                label="PD-9001",
            )

        class Test施策の必須項目のうち_1_つだけ:
            @pytest.mark.parametrize(
                ("field", "label"),
                [
                    pytest.param(
                        "initiative_id",
                        "PD-9001 / TST施策A",
                        id="initiative_id が無いとき、バリデーションエラーの一覧に initiative_id の欠落が含まれる",
                    ),
                    pytest.param(
                        "product_id",
                        "",
                        id="product_id が無いとき、バリデーションエラーの一覧に product_id の欠落が含まれる",
                    ),
                    pytest.param(
                        "kind",
                        "PD-9001 / TST施策A",
                        id="kind が無いとき、バリデーションエラーの一覧に kind の欠落が含まれる",
                    ),
                    pytest.param(
                        "name",
                        "",
                        id="name が無いとき、バリデーションエラーの一覧に name の欠落が含まれる",
                    ),
                    pytest.param(
                        "insight_cost",
                        "PD-9001 / TST施策A",
                        id="insight_cost が無いとき、バリデーションエラーの一覧に insight_cost の欠落が含まれる",
                    ),
                    pytest.param(
                        "effect_text",
                        "PD-9001 / TST施策A",
                        id="effect_text が無いとき、バリデーションエラーの一覧に effect_text の欠落が含まれる",
                    ),
                    pytest.param(
                        "effect",
                        "PD-9001 / TST施策A",
                        id="effect が無いとき、バリデーションエラーの一覧に effect の欠落が含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, field, label):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    products, drop_field_of_first_item(initiatives, field)
                )

                assert has_error(
                    errors, f"missing required field '{field}'", label=label
                )

        class Testinitiative_id_の形式が誤っているとき:
            @pytest.mark.parametrize(
                "initiative_id",
                [
                    pytest.param(
                        "IN-901",
                        id="initiative_id が IN-901 のとき、バリデーションエラーの一覧に initiative_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "IN-90001",
                        id="initiative_id が IN-90001 のとき、バリデーションエラーの一覧に initiative_id の形式の誤りが含まれる",
                    ),
                    pytest.param(
                        "XX-9001",
                        id="initiative_id が XX-9001 のとき、バリデーションエラーの一覧に initiative_id の形式の誤りが含まれる",
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, initiative_id):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    products,
                    override_first_item_fields(initiatives, initiative_id=initiative_id),
                )

                assert has_error(
                    errors,
                    "must match IN-NNNN format",
                    label="PD-9001 / TST施策A",
                )

        def test_2_件の施策の_initiative_id_がどちらも_IN9001_のとき_バリデーションエラーの一覧に_initiative_id_IN9001_の重複が含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()
            duplicated = [initiatives[0], {**initiatives[1], "initiative_id": "IN-9001"}]

            errors = generate_products.validate_products_and_initiatives(
                products, [*duplicated, *initiatives[2:]]
            )

            assert has_error(
                errors,
                "duplicate initiative_id 'IN-9001'",
                label="PD-9001 / TST施策B",
            )

        def test_施策の_product_id_が_PD9005_で_PD9005_のプロダクトが無いとき_バリデーションエラーの一覧に未知のプロダクトを参照している誤りが含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()

            errors = generate_products.validate_products_and_initiatives(
                products, override_first_item_fields(initiatives, product_id="PD-9005")
            )

            assert has_error(
                errors,
                "does not reference a known product",
                label="PD-9005 / TST施策A",
            )

        def test_施策の_kind_が_TST_UNKNOWN_のとき_バリデーションエラーの一覧に_kind_の値_TST_UNKNOWN_の誤りが含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()

            errors = generate_products.validate_products_and_initiatives(
                products, override_first_item_fields(initiatives, kind="TST_UNKNOWN")
            )

            assert has_error(
                errors,
                "invalid kind 'TST_UNKNOWN'",
                label="PD-9001 / TST施策A",
            )

        class Test施策の_insight_cost_が不正なとき:
            @pytest.mark.parametrize(
                "insight_cost",
                [
                    pytest.param(
                        -1,
                        id="マイナス1 のとき、バリデーションエラーの一覧に insight_cost が 0 以上の整数でない誤りが含まれる",
                    ),
                    pytest.param(
                        1.5,
                        id="小数の1点5 のとき、バリデーションエラーの一覧に insight_cost が 0 以上の整数でない誤りが含まれる",
                    ),
                    pytest.param(
                        "100",
                        id='文字列の "100" のとき、バリデーションエラーの一覧に insight_cost が 0 以上の整数でない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, insight_cost):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    products,
                    override_first_item_fields(initiatives, insight_cost=insight_cost),
                )

                assert has_error(
                    errors,
                    "insight_cost must be a non-negative integer",
                    label="PD-9001 / TST施策A",
                )

        class Test施策の_effect_が不正なとき:
            @pytest.mark.parametrize(
                "effect",
                [
                    pytest.param(
                        {},
                        id="ops も custom も持たないとき、バリデーションエラーの一覧に effect が ops も custom も持たない誤りが含まれる",
                    ),
                    pytest.param(
                        "TST効果",
                        id='effect が文字列の "TST効果" のとき、バリデーションエラーの一覧に effect が ops も custom も持たない誤りが含まれる',
                    ),
                ],
            )
            def test_検証する(self, build_catalog, has_error, effect):
                products, initiatives = build_catalog()

                errors = generate_products.validate_products_and_initiatives(
                    products, override_first_item_fields(initiatives, effect=effect)
                )

                assert has_error(
                    errors,
                    "effect must contain 'ops' or 'custom'",
                    label="PD-9001 / TST施策A",
                )

        def test_施策の_is_active_が文字列の_yes_のとき_バリデーションエラーの一覧に_is_active_が真偽値でない誤りが含まれる(
            self, build_catalog, has_error
        ):
            products, initiatives = build_catalog()

            errors = generate_products.validate_products_and_initiatives(
                products, override_first_item_fields(initiatives, is_active="yes")
            )

            assert has_error(
                errors,
                "is_active must be a boolean",
                label="PD-9001 / TST施策A",
            )


class Testプロダクト__配布用_JSON:
    class Test正常系:
        def test_product_id_が_PD9003_PD9001_PD9002_の順に定義された_3_件のプロダクトを出力すると_配布用_JSON_の要素の_product_id_が_PD9001_PD9002_PD9003_の順に並ぶ(
            self, generated_json_path, read_json_file, build_product
        ):
            products = [
                build_product(product_id="PD-9003", faction="Sugar"),
                build_product(product_id="PD-9001", faction="SHE"),
                build_product(product_id="PD-9002", faction="Tenki"),
            ]

            generate_products.generate_products_json(
                products, out_path=generated_json_path
            )

            assert [
                element["product_id"] for element in read_json_file(generated_json_path)
            ] == ["PD-9001", "PD-9002", "PD-9003"]


class Testプロダクト__初期投入_SQL:
    class Test正常系:
        class Testproduct_id_が_PD9001_のプロダクトと_PD9002_のプロダクトの_2_件の初期投入_SQL_を適用すると:
            def test_読み戻した_products_の_product_id_の一覧が_PD9001_PD9002_になる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT product_id FROM card.products ORDER BY product_id"
                )
                assert [row["product_id"] for row in rows] == ["PD-9001", "PD-9002"]

            def test_PD9001_の_faction_が_SHE_PD9002_の_faction_が_Tenki_のとき_読み戻した_faction_の列が_PD9001_の行は_SHE_PD9002_の行は_Tenki_になる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT product_id, faction FROM card.products")
                assert {row["product_id"]: row["faction"] for row in rows} == {
                    "PD-9001": "SHE",
                    "PD-9002": "Tenki",
                }

            def test_PD9001_の_product_name_が_TSTプロダクトA_PD9002_の_product_name_が_TSTプロダクトB_のとき_読み戻した_product_name_の列が_PD9001_の行は_TSTプロダクトA_PD9002_の行は_TSTプロダクトB_になる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT product_id, product_name FROM card.products")
                assert {row["product_id"]: row["product_name"] for row in rows} == {
                    "PD-9001": "TSTプロダクトA",
                    "PD-9002": "TSTプロダクトB",
                }

            def test_PD9001_の_is_active_が_true_PD9002_の_is_active_が_false_のとき_読み戻した_is_active_の列が_PD9001_の行は_true_PD9002_の行は_false_になる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT product_id, is_active FROM card.products")
                assert {row["product_id"]: row["is_active"] for row in rows} == {
                    "PD-9001": True,
                    "PD-9002": False,
                }

        def test_product_name_が_TST_と_プロダクトA_の間に単引用符を_1_つ挟んだ名前TST単引用符プロダクトAのプロダクトの初期投入_SQL_を適用すると_product_name_の列が同じ名前TST単引用符プロダクトAになる(
            self, build_product, seed_path, apply_sql_file, select_rows
        ):
            products = [build_product(product_name="TST'プロダクトA")]
            generate_products.generate_products_seed_sql(products, out_path=seed_path)

            apply_sql_file(seed_path)

            rows = select_rows("SELECT product_name FROM card.products")
            assert [row["product_name"] for row in rows] == ["TST'プロダクトA"]

        class TestPD9001_の_product_name_が_TSTプロダクトA_PD9002_の_product_name_が_TSTプロダクトB_の_2_件のプロダクトの初期投入_SQL_を_2_回続けて適用すると:
            def test_読み戻した_product_name_の列が_PD9001_の行は_TSTプロダクトA_PD9002_の行は_TSTプロダクトB_のままになる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                rows = select_rows("SELECT product_id, product_name FROM card.products")
                assert {row["product_id"]: row["product_name"] for row in rows} == {
                    "PD-9001": "TSTプロダクトA",
                    "PD-9002": "TSTプロダクトB",
                }

            def test_読み戻した_products_の行数が_2_になる(
                self, two_products, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_products_seed_sql(
                    two_products, out_path=seed_path
                )

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                rows = select_rows("SELECT count(*) AS row_count FROM card.products")
                assert rows[0]["row_count"] == 2

        def test_product_id_が_PD9001_で_product_name_が_TSTプロダクトA_のプロダクトを適用済みのとき_product_name_を_TSTプロダクトB_に変えた定義から出力し直した_SQL_を適用すると_PD9001_の行の_product_name_の列が_TSTプロダクトB_になる(
            self,
            build_product,
            seed_path,
            updated_seed_path,
            apply_sql_file,
            select_rows,
        ):
            generate_products.generate_products_seed_sql(
                [build_product(product_name="TSTプロダクトA")], out_path=seed_path
            )
            generate_products.generate_products_seed_sql(
                [build_product(product_name="TSTプロダクトB")], out_path=updated_seed_path
            )
            apply_sql_file(seed_path)

            apply_sql_file(updated_seed_path)

            rows = select_rows(
                "SELECT product_name FROM card.products WHERE product_id = 'PD-9001'"
            )
            assert [row["product_name"] for row in rows] == ["TSTプロダクトB"]


class Test施策__配布用_JSON:
    class Test正常系:
        def test_initiative_id_が_IN9003_IN9001_IN9002_の順に定義された_3_件の施策を出力すると_配布用_JSON_の要素の_initiative_id_が_IN9001_IN9002_IN9003_の順に並ぶ(
            self, generated_json_path, read_json_file, build_initiative
        ):
            initiatives = [
                build_initiative(initiative_id="IN-9003"),
                build_initiative(initiative_id="IN-9001"),
                build_initiative(initiative_id="IN-9002"),
            ]

            generate_products.generate_initiatives_json(
                initiatives, out_path=generated_json_path
            )

            assert [
                element["initiative_id"]
                for element in read_json_file(generated_json_path)
            ] == ["IN-9001", "IN-9002", "IN-9003"]


class Test施策__初期投入_SQL:
    class Test正常系:
        class Testinitiative_id_が_IN9001_の施策と_IN9002_の施策の_2_件の初期投入_SQL_を適用すると:
            def test_読み戻した_initiatives_の_initiative_id_の一覧が_IN9001_IN9002_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT initiative_id FROM card.initiatives ORDER BY initiative_id"
                )
                assert [row["initiative_id"] for row in rows] == ["IN-9001", "IN-9002"]

            def test_IN9001_の_product_id_が_PD9001_IN9002_の_product_id_が_PD9002_のとき_読み戻した_product_id_の列が_IN9001_の行は_PD9001_IN9002_の行は_PD9002_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT initiative_id, product_id FROM card.initiatives"
                )
                assert {row["initiative_id"]: row["product_id"] for row in rows} == {
                    "IN-9001": "PD-9001",
                    "IN-9002": "PD-9002",
                }

            def test_IN9001_の_kind_が_routine_IN9002_の_kind_が_special_のとき_読み戻した_kind_の列が_IN9001_の行は_routine_IN9002_の行は_special_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT initiative_id, kind FROM card.initiatives")
                assert {row["initiative_id"]: row["kind"] for row in rows} == {
                    "IN-9001": "routine",
                    "IN-9002": "special",
                }

            def test_IN9001_の_name_が_TST施策A_IN9002_の_name_が_TST施策B_のとき_読み戻した_name_の列が_IN9001_の行は_TST施策A_IN9002_の行は_TST施策B_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT initiative_id, name FROM card.initiatives")
                assert {row["initiative_id"]: row["name"] for row in rows} == {
                    "IN-9001": "TST施策A",
                    "IN-9002": "TST施策B",
                }

            def test_IN9001_の_insight_cost_が_100_IN9002_の_insight_cost_が_250_のとき_読み戻した_insight_cost_の列が_IN9001_の行は_100_IN9002_の行は_250_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT initiative_id, insight_cost FROM card.initiatives"
                )
                assert {row["initiative_id"]: row["insight_cost"] for row in rows} == {
                    "IN-9001": 100,
                    "IN-9002": 250,
                }

            def test_IN9001_の_effect_text_が_TST効果A_IN9002_の_effect_text_が_TST効果B_のとき_読み戻した_effect_text_の列が_IN9001_の行は_TST効果A_IN9002_の行は_TST効果B_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT initiative_id, effect_text FROM card.initiatives"
                )
                assert {row["initiative_id"]: row["effect_text"] for row in rows} == {
                    "IN-9001": "TST効果A",
                    "IN-9002": "TST効果B",
                }

            def test_IN9001_の_is_active_が_true_IN9002_の_is_active_が_false_のとき_読み戻した_is_active_の列が_IN9001_の行は_true_IN9002_の行は_false_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows(
                    "SELECT initiative_id, is_active FROM card.initiatives"
                )
                assert {row["initiative_id"]: row["is_active"] for row in rows} == {
                    "IN-9001": True,
                    "IN-9002": False,
                }

            def test_IN9001_の_effect_が_ops_に_draw_の_count_1_IN9002_の_effect_が_ops_に_gain_budget_の_target_myself_と_amount_400_のとき_読み戻した_effect_の列の_JSON_が_IN9001_の行は_ops_に_draw_の_count_1_IN9002_の行は_ops_に_gain_budget_の_target_myself_と_amount_400_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)

                rows = select_rows("SELECT initiative_id, effect FROM card.initiatives")
                assert {row["initiative_id"]: row["effect"] for row in rows} == {
                    "IN-9001": {"ops": [{"draw": {"count": 1}}]},
                    "IN-9002": {
                        "ops": [{"gain_budget": {"target": "myself", "amount": 400}}]
                    },
                }

        class TestIN9001_の_name_が_TST施策A_IN9002_の_name_が_TST施策B_の_2_件の施策の初期投入_SQL_を_2_回続けて適用すると:
            def test_読み戻した_name_の列が_IN9001_の行は_TST施策A_IN9002_の行は_TST施策B_のままになる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                rows = select_rows("SELECT initiative_id, name FROM card.initiatives")
                assert {row["initiative_id"]: row["name"] for row in rows} == {
                    "IN-9001": "TST施策A",
                    "IN-9002": "TST施策B",
                }

            def test_読み戻した_initiatives_の行数が_2_になる(
                self, two_initiatives, seed_path, apply_sql_file, select_rows
            ):
                generate_products.generate_initiatives_seed_sql(
                    two_initiatives, out_path=seed_path
                )

                apply_sql_file(seed_path)
                apply_sql_file(seed_path)

                rows = select_rows("SELECT count(*) AS row_count FROM card.initiatives")
                assert rows[0]["row_count"] == 2

        def test_initiative_id_が_IN9001_で_name_が_TST施策A_の施策を適用済みのとき_name_を_TST施策B_に変えた定義から出力し直した_SQL_を適用すると_IN9001_の行の_name_の列が_TST施策B_になる(
            self,
            build_initiative,
            insert_product_row,
            seed_path,
            updated_seed_path,
            apply_sql_file,
            select_rows,
        ):
            insert_product_row("PD-9001", "SHE")
            generate_products.generate_initiatives_seed_sql(
                [build_initiative(name="TST施策A")], out_path=seed_path
            )
            generate_products.generate_initiatives_seed_sql(
                [build_initiative(name="TST施策B")], out_path=updated_seed_path
            )
            apply_sql_file(seed_path)

            apply_sql_file(updated_seed_path)

            rows = select_rows(
                "SELECT name FROM card.initiatives WHERE initiative_id = 'IN-9001'"
            )
            assert [row["name"] for row in rows] == ["TST施策B"]


class Testプロダクト__プロダクト一覧文書:
    class Test正常系:
        def test_product_id_が_PD9002_PD9001_の順に定義された_2_件のプロダクトを出力すると_プロダクトの節の見出しが_PD9001_の節_PD9002_の節の順に並ぶ(
            self, build_catalog, render_product_list, select_section_titles
        ):
            products, initiatives = build_catalog(2)

            document = render_product_list([products[1], products[0]], initiatives)

            titles = select_section_titles(document)
            assert len(titles) == 2
            assert "PD-9001" in titles[0]
            assert "PD-9002" in titles[1]

        def test_プロダクト一覧文書の題名の見出しは_Overload_Party_Product_List_になる(
            self, build_catalog, render_product_list, select_title_headings
        ):
            products, initiatives = build_catalog(1)

            document = render_product_list(products, initiatives)

            assert select_title_headings(document) == ["Overload Party Product List"]

        def test_プロダクト名が_TSTプロダクト_faction_が_Tenki_product_id_が_PD9001_のプロダクトがあるとき_そのプロダクトの節の見出しは_TSTプロダクトTenki_PD9001_になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_section_titles,
        ):
            products = [build_product(product_name="TSTプロダクト", faction="Tenki")]

            document = render_product_list(products, [build_initiative()])

            assert select_section_titles(document) == [
                "TSTプロダクト（Tenki、PD-9001）"
            ]

        def test_initiative_id_が_IN9001_の施策のとき_施策の表の_ID_列が_IN9001_になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(initiative_id="IN-9001")]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["ID"] for row in rows] == ["IN-9001"]

        def test_施策名が_TST施策A_の施策のとき_施策の表の施策列が_TST施策A_になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(name="TST施策A")]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["施策"] for row in rows] == ["TST施策A"]

        def test_insight_cost_が_150_の施策のとき_施策の表の使用コスト列が_150_になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(insight_cost=150)]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["使用コスト"] for row in rows] == ["150"]

        def test_effect_text_が_TST効果A_の施策のとき_施策の表の効果列が_TST効果A_になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(effect_text="TST効果A")]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["効果"] for row in rows] == ["TST効果A"]

        def test_区分が_routine_の施策のとき_区分列がルーチン1ターン1回になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(kind="routine")]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["区分"] for row in rows] == ["ルーチン（1ターン1回）"]

        def test_区分が_special_の施策のとき_区分列がスペシャル1ゲーム1回になる(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            document = render_product_list(
                [build_product()], [build_initiative(kind="special")]
            )

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["区分"] for row in rows] == ["スペシャル（1ゲーム1回）"]

        def test_定義ファイルで_special_の_IN9001_が_routine_の_IN9002_より先に書かれているとき_表の_ID_列が_IN9002_IN9001_の順に並ぶ(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            initiatives = [
                build_initiative(initiative_id="IN-9001", kind="special"),
                build_initiative(initiative_id="IN-9002", kind="routine"),
            ]

            document = render_product_list([build_product()], initiatives)

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["ID"] for row in rows] == ["IN-9002", "IN-9001"]

        def test_区分が_routine_の_IN9003_IN9001_IN9002_が定義ファイルにこの順に書かれているとき_表の_ID_列が_IN9003_IN9001_IN9002_の順に並ぶ(
            self,
            build_product,
            build_initiative,
            render_product_list,
            select_table_rows,
        ):
            initiatives = [
                build_initiative(initiative_id="IN-9003", kind="routine"),
                build_initiative(initiative_id="IN-9001", kind="routine"),
                build_initiative(initiative_id="IN-9002", kind="routine"),
            ]

            document = render_product_list([build_product()], initiatives)

            rows = select_table_rows(document[-1]["tables"][0])
            assert [row["ID"] for row in rows] == ["IN-9003", "IN-9001", "IN-9002"]


class Testプロダクト__生成コマンドの終了コード:
    class Test正常系:
        def test_product_id_が_PD9001_PD9002_PD9003_PD9004_の_4_件のプロダクトと_initiative_id_が_IN9001_から_IN9008_の_8_件の施策の有効な定義があるとき_実行すると_終了コードが_0_になる(
            self, build_catalog, write_product_definitions, run_generator
        ):
            write_product_definitions(*build_catalog())

            result = run_generator("generate_products.py")

            assert result.returncode == 0

    class Test異常系:
        class Test実行すると:
            @pytest.mark.parametrize(
                "arrange_files",
                [
                    pytest.param(
                        lambda catalog: [
                            ("data/initiatives.yaml", {"initiatives": catalog[1]})
                        ],
                        id="プロダクト定義のファイルが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        lambda catalog: [
                            ("data/products.yaml", {"products": catalog[0]})
                        ],
                        id="施策定義のファイルが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        lambda catalog: [
                            ("data/products.yaml", {"products": []}),
                            ("data/initiatives.yaml", {"initiatives": catalog[1]}),
                        ],
                        id="プロダクトが 1 件も無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        lambda catalog: [
                            ("data/products.yaml", {"products": catalog[0]}),
                            ("data/initiatives.yaml", {"initiatives": []}),
                        ],
                        id="施策が 1 件も無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        lambda catalog: [
                            (
                                "data/products.yaml",
                                {
                                    "products": override_first_item_fields(
                                        catalog[0], is_active="yes"
                                    )
                                },
                            ),
                            ("data/initiatives.yaml", {"initiatives": catalog[1]}),
                        ],
                        id='PD-9001 のプロダクトの is_active が文字列の "yes" のとき、終了コードが 1 になる',
                    ),
                ],
            )
            def test_実行する(
                self,
                workspace,
                build_catalog,
                write_yaml_file,
                run_generator,
                arrange_files,
            ):
                for relative_path, content in arrange_files(build_catalog()):
                    write_yaml_file(workspace / relative_path, content)

                result = run_generator("generate_products.py")

                assert result.returncode == 1


class Testプロダクト__生成コマンドの標準エラー出力:
    class Test正常系:
        class Test有効な定義があるとき_実行すると_標準エラー出力のGeneratedの行のうち:
            def test_プロダクトが_4_件のとき_プロダクトの配布用_JSON初期投入_SQLプロダクト一覧文書を書いた_3_行が_どれも_4_件を示す(
                self,
                build_catalog,
                write_product_definitions,
                run_generator,
                read_generated_counts,
            ):
                write_product_definitions(*build_catalog())

                result = run_generator("generate_products.py")

                assert read_generated_counts(result.stderr, "products") == {
                    "products_gen.json": "4",
                    "products_seed.sql": "4",
                    "PRODUCTS.md": "4",
                }

            def test_施策が_8_件のとき_施策の配布用_JSON初期投入_SQL_を書いた_2_行が_どれも_8_件を示す(
                self,
                build_catalog,
                write_product_definitions,
                run_generator,
                read_generated_counts,
            ):
                write_product_definitions(*build_catalog())

                result = run_generator("generate_products.py")

                assert read_generated_counts(result.stderr, "initiatives") == {
                    "initiatives_gen.json": "8",
                    "initiatives_seed.sql": "8",
                }

    class Test異常系:
        def test_プロダクト定義のファイルが無いとき_標準エラー出力にERROR_に続けてそのファイルの場所と_not_foundが表示される(
            self, workspace, build_catalog, write_yaml_file, run_generator
        ):
            _, initiatives = build_catalog()
            write_yaml_file(
                workspace / "data" / "initiatives.yaml", {"initiatives": initiatives}
            )

            result = run_generator("generate_products.py")

            assert (
                f"ERROR: {workspace / 'data' / 'products.yaml'} not found"
                in result.stderr
            )

        def test_施策定義のファイルが無いとき_標準エラー出力にERROR_に続けてそのファイルの場所と_not_foundが表示される(
            self, workspace, build_catalog, write_yaml_file, run_generator
        ):
            products, _ = build_catalog()
            write_yaml_file(workspace / "data" / "products.yaml", {"products": products})

            result = run_generator("generate_products.py")

            assert (
                f"ERROR: {workspace / 'data' / 'initiatives.yaml'} not found"
                in result.stderr
            )

        def test_プロダクトが_1_件も無いとき_標準エラー出力にERROR_No_products_loaded_from_YAMLが表示される(
            self, build_catalog, write_product_definitions, run_generator
        ):
            _, initiatives = build_catalog()
            write_product_definitions([], initiatives)

            result = run_generator("generate_products.py")

            assert "ERROR: No products loaded from YAML" in result.stderr

        def test_施策が_1_件も無いとき_標準エラー出力にERROR_No_initiatives_loaded_from_YAMLが表示される(
            self, build_catalog, write_product_definitions, run_generator
        ):
            products, _ = build_catalog()
            write_product_definitions(products, [])

            result = run_generator("generate_products.py")

            assert "ERROR: No initiatives loaded from YAML" in result.stderr

        class Testproduct_id_が_PD9001_で_product_name_が_TSTプロダクトA_のプロダクトと_PD9001_に属し_name_が_TST施策A_の施策の_is_active_が_どちらも文字列の_yes_で_ほかは有効な定義のとき:
            def test_標準エラー出力にValidation_failed_with_2_errorsが表示される(
                self, build_catalog, write_product_definitions, run_generator
            ):
                products, initiatives = build_catalog()
                write_product_definitions(
                    override_first_item_fields(products, is_active="yes"),
                    override_first_item_fields(initiatives, is_active="yes"),
                )

                result = run_generator("generate_products.py")

                assert "Validation failed with 2 error(s)" in result.stderr

            def test_標準エラー出力にPD9001_TSTプロダクトA_is_active_must_be_a_booleanとPD9001_TST施策A_is_active_must_be_a_booleanの_2_件が表示される(
                self, build_catalog, write_product_definitions, run_generator
            ):
                products, initiatives = build_catalog()
                write_product_definitions(
                    override_first_item_fields(products, is_active="yes"),
                    override_first_item_fields(initiatives, is_active="yes"),
                )

                result = run_generator("generate_products.py")

                assert "PD-9001 TSTプロダクトA: is_active must be a boolean" in result.stderr
                assert "PD-9001 / TST施策A: is_active must be a boolean" in result.stderr


@pytest.fixture
def read_generated_product_ids(locate_output_file, read_json_file):
    def read_product_ids_of_generated_json():
        return [
            element["product_id"]
            for element in read_json_file(locate_output_file("products_json"))
        ]

    return read_product_ids_of_generated_json


@pytest.fixture
def read_generated_initiative_ids(locate_output_file, read_json_file):
    def read_initiative_ids_of_generated_json():
        return [
            element["initiative_id"]
            for element in read_json_file(locate_output_file("initiatives_json"))
        ]

    return read_initiative_ids_of_generated_json


@pytest.fixture
def read_document_product_ids(
    locate_output_file, parse_markdown_file, select_section_titles
):
    def read_product_ids_of_generated_document():
        document = parse_markdown_file(locate_output_file("products_document"))
        return [
            PRODUCT_ID_PATTERN.findall(title)[0]
            for title in select_section_titles(document)
        ]

    return read_product_ids_of_generated_document


@pytest.fixture
def read_applied_product_ids(locate_output_file, apply_sql_file, select_rows):
    def read_product_ids_after_applying_seed_sql():
        apply_sql_file(locate_output_file("products_seed"))
        rows = select_rows("SELECT product_id FROM card.products ORDER BY product_id")
        return [row["product_id"] for row in rows]

    return read_product_ids_after_applying_seed_sql


@pytest.fixture
def read_applied_initiative_ids(locate_output_file, apply_sql_file, select_rows):
    def read_initiative_ids_after_applying_seed_sqls():
        apply_sql_file(locate_output_file("products_seed"))
        apply_sql_file(locate_output_file("initiatives_seed"))
        rows = select_rows(
            "SELECT initiative_id FROM card.initiatives ORDER BY initiative_id"
        )
        return [row["initiative_id"] for row in rows]

    return read_initiative_ids_after_applying_seed_sqls


class Testプロダクト__生成コマンドの出力:
    class Test正常系:
        class Testproduct_id_が_PD9001_PD9002_PD9003_PD9004_の_4_件のプロダクトと_initiative_id_が_IN9001_から_IN9008_の_8_件の施策の有効な定義があるとき:
            class Test実行すると:
                def test_プロダクトの配布用_JSON_の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_generated_product_ids,
                ):
                    write_product_definitions(*build_catalog())

                    run_generator("generate_products.py")

                    assert read_generated_product_ids() == EXPECTED_PRODUCT_IDS

                def test_施策の配布用_JSON_の_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_generated_initiative_ids,
                ):
                    write_product_definitions(*build_catalog())

                    run_generator("generate_products.py")

                    assert read_generated_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_作られた初期投入_SQL_のうちプロダクトの_SQL_を適用して読み戻した_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_applied_product_ids,
                ):
                    write_product_definitions(*build_catalog())

                    run_generator("generate_products.py")

                    assert read_applied_product_ids() == EXPECTED_PRODUCT_IDS

                def test_作られた初期投入_SQL_のうち施策の_SQL_を適用して読み戻した_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_applied_initiative_ids,
                ):
                    write_product_definitions(*build_catalog())

                    run_generator("generate_products.py")

                    assert read_applied_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_プロダクト一覧文書の節の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_document_product_ids,
                ):
                    write_product_definitions(*build_catalog())

                    run_generator("generate_products.py")

                    assert read_document_product_ids() == EXPECTED_PRODUCT_IDS

            class Test出力先のディレクトリが無いときに実行すると:
                def test_出力先のディレクトリ内にプロダクトと施策の配布用_JSONプロダクトと施策の初期投入_SQLプロダクト一覧文書の_5_つが作られる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    remove_output_directories,
                    run_generator,
                    locate_output_file,
                ):
                    write_product_definitions(*build_catalog())
                    remove_output_directories()

                    run_generator("generate_products.py")

                    assert [
                        output_key
                        for output_key in GENERATED_OUTPUT_KEYS
                        if locate_output_file(output_key).exists()
                    ] == GENERATED_OUTPUT_KEYS

                def test_作られた配布用_JSON_のうちプロダクトの_JSON_の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    remove_output_directories,
                    run_generator,
                    read_generated_product_ids,
                ):
                    write_product_definitions(*build_catalog())
                    remove_output_directories()

                    run_generator("generate_products.py")

                    assert read_generated_product_ids() == EXPECTED_PRODUCT_IDS

            class Test続けて_2_回実行すると:
                def test_2_回目の実行で作られたプロダクトの配布用_JSON_の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_generated_product_ids,
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")

                    second_result = run_generator("generate_products.py")

                    assert second_result.returncode == 0
                    assert read_generated_product_ids() == EXPECTED_PRODUCT_IDS

                def test_2_回目の実行で作られた施策の配布用_JSON_の_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_generated_initiative_ids,
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")

                    second_result = run_generator("generate_products.py")

                    assert second_result.returncode == 0
                    assert read_generated_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_2_回目の実行で作られた初期投入_SQL_のうちプロダクトの_SQL_を適用して読み戻した_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_applied_product_ids,
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")

                    second_result = run_generator("generate_products.py")

                    assert second_result.returncode == 0
                    assert read_applied_product_ids() == EXPECTED_PRODUCT_IDS

                def test_2_回目の実行で作られた初期投入_SQL_のうち施策の_SQL_を適用して読み戻した_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_applied_initiative_ids,
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")

                    second_result = run_generator("generate_products.py")

                    assert second_result.returncode == 0
                    assert read_applied_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_2_回目の実行で作られたプロダクト一覧文書の節の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    read_document_product_ids,
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")

                    second_result = run_generator("generate_products.py")

                    assert second_result.returncode == 0
                    assert read_document_product_ids() == EXPECTED_PRODUCT_IDS

    class Test異常系:
        class TestPD9001_のプロダクトの_is_active_が文字列の_yes_の定義のとき:
            class Test出力先に何も無い状態で実行すると:
                @pytest.mark.parametrize(
                    "output_key",
                    [
                        pytest.param(
                            "products_json",
                            id="プロダクトの配布用 JSON が作られない",
                        ),
                        pytest.param(
                            "initiatives_json",
                            id="施策の配布用 JSON が作られない",
                        ),
                        pytest.param(
                            "products_seed",
                            id="プロダクトの初期投入 SQL が作られない",
                        ),
                        pytest.param(
                            "initiatives_seed",
                            id="施策の初期投入 SQL が作られない",
                        ),
                        pytest.param(
                            "products_document",
                            id="プロダクト一覧文書が作られない",
                        ),
                    ],
                )
                def test_実行する(
                    self,
                    build_catalog,
                    write_product_definitions,
                    run_generator,
                    locate_output_file,
                    output_key,
                ):
                    products, initiatives = build_catalog()
                    write_product_definitions(
                        override_first_item_fields(products, is_active="yes"),
                        initiatives,
                    )

                    run_generator("generate_products.py")

                    assert not locate_output_file(output_key).exists()

        class TestPD9001_から_PD9004_の_4_件のプロダクトと_IN9001_から_IN9008_の_8_件の施策の定義で出力を作ったあと_PD9005_のプロダクトの_is_active_が文字列の_yes_の定義に変えたとき:
            class Test既存の出力があるときに実行すると:
                @pytest.fixture
                def rerun_generator_after_invalidating_fifth_product(
                    self, build_catalog, write_product_definitions, run_generator
                ):
                    write_product_definitions(*build_catalog())
                    run_generator("generate_products.py")
                    fifth_products, fifth_initiatives = build_catalog(5)
                    write_product_definitions(
                        override_last_item_fields(fifth_products, is_active="yes"),
                        fifth_initiatives,
                    )
                    run_generator("generate_products.py")

                def test_プロダクトの配布用_JSON_の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_のままになる(
                    self,
                    rerun_generator_after_invalidating_fifth_product,
                    read_generated_product_ids,
                ):
                    assert read_generated_product_ids() == EXPECTED_PRODUCT_IDS

                def test_施策の配布用_JSON_の_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件のままになる(
                    self,
                    rerun_generator_after_invalidating_fifth_product,
                    read_generated_initiative_ids,
                ):
                    assert read_generated_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_既存の初期投入_SQL_のうちプロダクトの_SQL_を適用して読み戻した_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_になる(
                    self,
                    rerun_generator_after_invalidating_fifth_product,
                    read_applied_product_ids,
                ):
                    assert read_applied_product_ids() == EXPECTED_PRODUCT_IDS

                def test_既存の初期投入_SQL_のうち施策の_SQL_を適用して読み戻した_initiative_id_の一覧が_IN9001_から_IN9008_の_8_件になる(
                    self,
                    rerun_generator_after_invalidating_fifth_product,
                    read_applied_initiative_ids,
                ):
                    assert read_applied_initiative_ids() == EXPECTED_INITIATIVE_IDS

                def test_プロダクト一覧文書の節の_product_id_の一覧が_PD9001_PD9002_PD9003_PD9004_のままになる(
                    self,
                    rerun_generator_after_invalidating_fifth_product,
                    read_document_product_ids,
                ):
                    assert read_document_product_ids() == EXPECTED_PRODUCT_IDS
