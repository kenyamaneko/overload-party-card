import re

import generate_card_packs
import pytest


@pytest.fixture
def two_packs(build_pack):
    return [
        build_pack(
            pack_id="TST-PACK-0001",
            description="TST説明A",
            is_active=True,
            cards=[
                {"card_id": "TST-0001", "copies": 3},
                {"card_id": "TST-0002", "copies": 1},
            ],
        ),
        build_pack(
            pack_id="TST-PACK-0002",
            description="TST説明B",
            is_active=False,
            cards=[{"card_id": "TST-0001", "copies": 2}],
        ),
    ]


@pytest.fixture
def valid_packs(build_pack):
    return [
        build_pack(pack_id="TST-PACK-0001"),
        build_pack(pack_id="TST-PACK-0002"),
    ]


@pytest.fixture
def read_applied_pack_ids(locate_output_file, apply_sql_file, select_rows):
    def read_pack_ids_after_applying_seed_sql():
        apply_sql_file(locate_output_file("pack_seed"))
        rows = select_rows("SELECT pack_id FROM card.card_pack ORDER BY pack_id")
        return [row["pack_id"] for row in rows]

    return read_pack_ids_after_applying_seed_sql


class Testカードパック__パック定義の検証:
    class Test正常系:
        class Testほかの項目がすべて有効なとき:
            @pytest.mark.parametrize(
                "packs",
                [
                    pytest.param(
                        [
                            {
                                "pack_id": "TST-PACK-0001",
                                "cards": [{"card_id": "TST-0001", "copies": 1}],
                            }
                        ],
                        id="pack_id が TST-PACK-0001 で cards が card_id TST-0001 の copies 1 の 1 件のとき、検証がエラーにならない",
                    ),
                    pytest.param(
                        [
                            {
                                "pack_id": "TST-PACK-0001",
                                "cards": [
                                    {"card_id": "TST-0001", "copies": 1},
                                    {"card_id": "TST-0002", "copies": 1},
                                ],
                            }
                        ],
                        id="pack_id が TST-PACK-0001 で cards が card_id TST-0001 と card_id TST-0002 の 2 件のとき、検証がエラーにならない",
                    ),
                    pytest.param(
                        [
                            {
                                "pack_id": "TST-PACK-0001",
                                "cards": [{"card_id": "TST-0001", "copies": 1}],
                            },
                            {
                                "pack_id": "TST-PACK-0002",
                                "cards": [{"card_id": "TST-0001", "copies": 1}],
                            },
                        ],
                        id="TST-PACK-0001 と TST-PACK-0002 の cards がどちらも card_id TST-0001 を含むとき、検証がエラーにならない",
                    ),
                ],
            )
            def test_検証する(self, packs):
                assert generate_card_packs.validate_packs(packs) is None

    class Test異常系:
        class Testpack_id_が不正なとき:
            @pytest.mark.parametrize(
                "pack_fields",
                [
                    pytest.param(
                        {"omit": ("pack_id",)},
                        id="無いとき、検証が、pack_id が文字列でない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        {"pack_id": ""},
                        id="空文字のとき、検証が、pack_id が文字列でない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        {"pack_id": 1},
                        id="数値の 1 のとき、検証が、pack_id が文字列でない、または空であることを示すエラーになる",
                    ),
                ],
            )
            def test_検証する(self, build_pack, pack_fields):
                with pytest.raises(
                    ValueError, match=re.escape("pack_id must be a non-empty string")
                ):
                    generate_card_packs.validate_packs([build_pack(**pack_fields)])

        def test_2_つのパックの_pack_id_がどちらも_TSTPACK0001_のとき_検証が_pack_id_TSTPACK0001_の重複を示すエラーになる(
            self, build_pack
        ):
            packs = [build_pack(), build_pack()]

            with pytest.raises(
                ValueError, match=re.escape("duplicate pack_id: TST-PACK-0001")
            ):
                generate_card_packs.validate_packs(packs)

        class Testパックの_cards_が不正なとき:
            @pytest.mark.parametrize(
                "pack_fields",
                [
                    pytest.param(
                        {"omit": ("cards",)},
                        id="cards が無いとき、検証が、パックの cards がリストでない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        {"cards": []},
                        id="cards が空のリストのとき、検証が、パックの cards がリストでない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        {"cards": "TST-0001"},
                        id='cards が文字列の "TST-0001" のとき、検証が、パックの cards がリストでない、または空であることを示すエラーになる',
                    ),
                ],
            )
            def test_検証する(self, build_pack, pack_fields):
                with pytest.raises(
                    ValueError,
                    match=re.escape("TST-PACK-0001: cards must be a non-empty list"),
                ):
                    generate_card_packs.validate_packs([build_pack(**pack_fields)])

            def test_cards_の要素が文字列の_TST0001_のとき_検証が_パックの_cards_の要素がキーと値の組でないことを示すエラーになる(
                self, build_pack
            ):
                packs = [build_pack(cards=["TST-0001"])]

                with pytest.raises(
                    ValueError,
                    match=re.escape("TST-PACK-0001: cards entries must be mappings"),
                ):
                    generate_card_packs.validate_packs(packs)

        class Testパックの_card_id_が不正なとき:
            @pytest.mark.parametrize(
                "cards",
                [
                    pytest.param(
                        [{"copies": 1}],
                        id="無いとき、検証が、パックの card_id が文字列でない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        [{"card_id": "", "copies": 1}],
                        id="空文字のとき、検証が、パックの card_id が文字列でない、または空であることを示すエラーになる",
                    ),
                    pytest.param(
                        [{"card_id": 1, "copies": 1}],
                        id="数値の 1 のとき、検証が、パックの card_id が文字列でない、または空であることを示すエラーになる",
                    ),
                ],
            )
            def test_検証する(self, build_pack, cards):
                with pytest.raises(
                    ValueError,
                    match=re.escape(
                        "TST-PACK-0001: card_id must be a non-empty string"
                    ),
                ):
                    generate_card_packs.validate_packs([build_pack(cards=cards)])

        def test_パック_TSTPACK0001_の_cards_に_card_id_TST0001_が_2_件あるとき_検証が_TSTPACK0001_の_card_id_TST0001_の重複を示すエラーになる(
            self, build_pack
        ):
            packs = [
                build_pack(
                    cards=[
                        {"card_id": "TST-0001", "copies": 1},
                        {"card_id": "TST-0001", "copies": 2},
                    ]
                )
            ]

            with pytest.raises(
                ValueError,
                match=re.escape("TST-PACK-0001: duplicate card_id TST-0001"),
            ):
                generate_card_packs.validate_packs(packs)

        class Testパックのカードの_copies_が_1_以上の整数でないとき:
            @pytest.mark.parametrize(
                "cards",
                [
                    pytest.param(
                        [{"card_id": "TST-0001", "copies": 0}],
                        id="0 のとき、検証が、copies の誤りを示すエラーになる",
                    ),
                    pytest.param(
                        [{"card_id": "TST-0001", "copies": 1.5}],
                        id="小数の1点5 のとき、検証が、copies の誤りを示すエラーになる",
                    ),
                    pytest.param(
                        [{"card_id": "TST-0001", "copies": "3"}],
                        id='文字列の "3" のとき、検証が、copies の誤りを示すエラーになる',
                    ),
                    pytest.param(
                        [{"card_id": "TST-0001"}],
                        id="無いとき、検証が、copies の誤りを示すエラーになる",
                    ),
                ],
            )
            def test_検証する(self, build_pack, cards):
                with pytest.raises(
                    ValueError,
                    match=re.escape(
                        "TST-PACK-0001/TST-0001: copies must be a positive int"
                    ),
                ):
                    generate_card_packs.validate_packs([build_pack(cards=cards)])


class Testカードパック__初期投入_SQL:
    class Test正常系:
        class Testpack_id_が_TSTPACK0001_のパックと_TSTPACK0002_のパックの_2_パックの初期投入_SQL_を適用すると:
            def test_読み戻した_card_pack_の_pack_id_の一覧が_TSTPACK0001_TSTPACK0002_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                apply_sql_lines(
                    generate_card_packs.build_card_packs_seed_lines(two_packs)
                )

                rows = select_rows(
                    "SELECT pack_id FROM card.card_pack ORDER BY pack_id"
                )

                assert [row["pack_id"] for row in rows] == [
                    "TST-PACK-0001",
                    "TST-PACK-0002",
                ]

            def test_TSTPACK0001_の_description_が_TST説明A_TSTPACK0002_の_description_が_TST説明B_のとき_読み戻した_description_の列が_TSTPACK0001_の行は_TST説明A_TSTPACK0002_の行は_TST説明B_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                apply_sql_lines(
                    generate_card_packs.build_card_packs_seed_lines(two_packs)
                )

                rows = select_rows("SELECT pack_id, description FROM card.card_pack")

                assert {row["pack_id"]: row["description"] for row in rows} == {
                    "TST-PACK-0001": "TST説明A",
                    "TST-PACK-0002": "TST説明B",
                }

            def test_TSTPACK0001_の_is_active_が_true_TSTPACK0002_の_is_active_が_false_のとき_読み戻した_is_active_の列が_TSTPACK0001_の行は_true_TSTPACK0002_の行は_false_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                apply_sql_lines(
                    generate_card_packs.build_card_packs_seed_lines(two_packs)
                )

                rows = select_rows("SELECT pack_id, is_active FROM card.card_pack")

                assert {row["pack_id"]: row["is_active"] for row in rows} == {
                    "TST-PACK-0001": True,
                    "TST-PACK-0002": False,
                }

            def test_TSTPACK0001_の_cards_が_card_id_TST0001_の_copies_3_と_card_id_TST0002_の_copies_1_TSTPACK0002_の_cards_が_card_id_TST0001_の_copies_2_のとき_読み戻した_card_pack_cards_の行が_TSTPACK0001_は_TST0001_3_と_TST0002_1_TSTPACK0002_は_TST0001_2_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                apply_sql_lines(
                    generate_card_packs.build_card_packs_seed_lines(two_packs)
                )

                rows = select_rows(
                    "SELECT pack_id, card_id, copies FROM card.card_pack_cards "
                    "ORDER BY pack_id, card_id"
                )

                assert [
                    (row["pack_id"], row["card_id"], row["copies"]) for row in rows
                ] == [
                    ("TST-PACK-0001", "TST-0001", 3),
                    ("TST-PACK-0001", "TST-0002", 1),
                    ("TST-PACK-0002", "TST-0001", 2),
                ]

        def test_description_が_TST_と_説明A_の間に単引用符を_1_つ挟んだ説明TST単引用符説明Aのパックの初期投入_SQL_を適用すると_description_の列が同じ説明TST単引用符説明Aになる(
            self, build_pack, apply_sql_lines, select_rows
        ):
            packs = [build_pack(description="TST'説明A")]

            apply_sql_lines(generate_card_packs.build_card_packs_seed_lines(packs))

            rows = select_rows("SELECT description FROM card.card_pack")

            assert [row["description"] for row in rows] == ["TST'説明A"]

        class TestTSTPACK0001_の_description_が_TST説明A_で_cards_が_card_id_TST0001_の_copies_3_と_card_id_TST0002_の_copies_1_TSTPACK0002_の_description_が_TST説明B_で_cards_が_card_id_TST0001_の_copies_2_の_2_パックの初期投入_SQL_を_2_回続けて適用すると:
            def test_読み戻した_description_の列が_TSTPACK0001_の行は_TST説明A_TSTPACK0002_の行は_TST説明B_のままになる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                lines = generate_card_packs.build_card_packs_seed_lines(two_packs)
                apply_sql_lines(lines)
                apply_sql_lines(lines)

                rows = select_rows("SELECT pack_id, description FROM card.card_pack")

                assert {row["pack_id"]: row["description"] for row in rows} == {
                    "TST-PACK-0001": "TST説明A",
                    "TST-PACK-0002": "TST説明B",
                }

            def test_読み戻した_card_pack_の行数が_2_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                lines = generate_card_packs.build_card_packs_seed_lines(two_packs)
                apply_sql_lines(lines)
                apply_sql_lines(lines)

                rows = select_rows("SELECT count(*) AS row_count FROM card.card_pack")

                assert rows[0]["row_count"] == 2

            def test_読み戻した_card_pack_cards_の行が_TSTPACK0001_TST0001_3_TSTPACK0001_TST0002_1_TSTPACK0002_TST0001_2_の_3_行のままになる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                lines = generate_card_packs.build_card_packs_seed_lines(two_packs)
                apply_sql_lines(lines)
                apply_sql_lines(lines)

                rows = select_rows(
                    "SELECT pack_id, card_id, copies FROM card.card_pack_cards "
                    "ORDER BY pack_id, card_id"
                )

                assert [
                    (row["pack_id"], row["card_id"], row["copies"]) for row in rows
                ] == [
                    ("TST-PACK-0001", "TST-0001", 3),
                    ("TST-PACK-0001", "TST-0002", 1),
                    ("TST-PACK-0002", "TST-0001", 2),
                ]

            def test_読み戻した_card_pack_cards_の行数が_3_になる(
                self, two_packs, apply_sql_lines, select_rows
            ):
                lines = generate_card_packs.build_card_packs_seed_lines(two_packs)
                apply_sql_lines(lines)
                apply_sql_lines(lines)

                rows = select_rows(
                    "SELECT count(*) AS row_count FROM card.card_pack_cards"
                )

                assert rows[0]["row_count"] == 3

        def test_TSTPACK0001_の_cards_が_card_id_TST0001_と_TST0002_で適用済みのとき_TST0002_を除いた定義から出力し直した_SQL_を適用すると_card_pack_cards_の_TSTPACK0001_の行の_card_id_が_TST0001_だけになる(
            self, build_pack, apply_sql_lines, select_rows
        ):
            applied = build_pack(
                cards=[
                    {"card_id": "TST-0001", "copies": 1},
                    {"card_id": "TST-0002", "copies": 1},
                ]
            )
            without_second_card = build_pack(
                cards=[{"card_id": "TST-0001", "copies": 1}]
            )
            apply_sql_lines(
                generate_card_packs.build_card_packs_seed_lines([applied])
            )

            apply_sql_lines(
                generate_card_packs.build_card_packs_seed_lines([without_second_card])
            )

            rows = select_rows(
                "SELECT card_id FROM card.card_pack_cards "
                "WHERE pack_id = 'TST-PACK-0001' ORDER BY card_id"
            )
            assert [row["card_id"] for row in rows] == ["TST-0001"]

        def test_TSTPACK0001_の_card_id_TST0001_の_copies_が_3_で適用済みのとき_copies_を_5_に変えた定義から出力し直した_SQL_を適用すると_card_pack_cards_の_TSTPACK0001_と_TST0001_の行の_copies_の列が_5_になる(
            self, build_pack, apply_sql_lines, select_rows
        ):
            applied = build_pack(cards=[{"card_id": "TST-0001", "copies": 3}])
            with_five_copies = build_pack(cards=[{"card_id": "TST-0001", "copies": 5}])
            apply_sql_lines(
                generate_card_packs.build_card_packs_seed_lines([applied])
            )

            apply_sql_lines(
                generate_card_packs.build_card_packs_seed_lines([with_five_copies])
            )

            rows = select_rows(
                "SELECT copies FROM card.card_pack_cards "
                "WHERE pack_id = 'TST-PACK-0001' AND card_id = 'TST-0001'"
            )
            assert [row["copies"] for row in rows] == [5]


class Testカードパック__生成コマンドの終了コード:
    class Test正常系:
        def test_pack_id_が_TSTPACK0001_で_cards_が_card_id_TST0001_の_copies_1_のパック定義があるとき_実行すると_終了コードが_0_になる(
            self, write_pack_definitions, build_pack, run_generator
        ):
            write_pack_definitions([build_pack()])

            result = run_generator("generate_card_packs.py")

            assert result.returncode == 0

    class Test異常系:
        class Test実行すると:
            @pytest.mark.parametrize(
                "definition_files",
                [
                    pytest.param(
                        [],
                        id="パック定義のファイルが無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [("data/card_packs.yaml", {"TST_UNKNOWN": []})],
                        id="パック定義に packs が無いとき、終了コードが 1 になる",
                    ),
                    pytest.param(
                        [
                            (
                                "data/card_packs.yaml",
                                {
                                    "packs": [
                                        {
                                            "pack_id": "TST-PACK-0001",
                                            "cards": [
                                                {"card_id": "TST-0001", "copies": 0}
                                            ],
                                        }
                                    ]
                                },
                            )
                        ],
                        id="TST-PACK-0001 の card_id TST-0001 の copies が 0 のとき、終了コードが 1 になる",
                    ),
                ],
            )
            def test_実行する(
                self, workspace, write_yaml_file, run_generator, definition_files
            ):
                for relative_path, content in definition_files:
                    write_yaml_file(workspace / relative_path, content)

                result = run_generator("generate_card_packs.py")

                assert result.returncode == 1


class Testカードパック__生成コマンドの標準出力:
    class Test正常系:
        def test_2_パックの有効なパック定義があるとき_実行すると_標準出力に2_packsが表示される(
            self, write_pack_definitions, valid_packs, run_generator
        ):
            write_pack_definitions(valid_packs)

            result = run_generator("generate_card_packs.py")

            assert "2 packs" in result.stdout

        def test_cards_の要素が_TSTPACK0001_に_2_件_TSTPACK0002_に_1_件ある有効なパック定義のとき_実行すると_標準出力に3_packcard_rowsが表示される(
            self, write_pack_definitions, build_pack, run_generator
        ):
            write_pack_definitions(
                [
                    build_pack(
                        pack_id="TST-PACK-0001",
                        cards=[
                            {"card_id": "TST-0001", "copies": 1},
                            {"card_id": "TST-0002", "copies": 1},
                        ],
                    ),
                    build_pack(
                        pack_id="TST-PACK-0002",
                        cards=[{"card_id": "TST-0001", "copies": 1}],
                    ),
                ]
            )

            result = run_generator("generate_card_packs.py")

            assert "3 pack-card rows" in result.stdout


class Testカードパック__生成コマンドの標準エラー出力:
    class Test異常系:
        def test_パック定義のファイルが無いとき_標準エラー出力の末尾にFileNotFoundErrorと見つからなかったパック定義のファイルの場所が表示される(
            self, workspace, run_generator
        ):
            result = run_generator("generate_card_packs.py")

            last_line = result.stderr.rstrip().splitlines()[-1]
            assert "FileNotFoundError" in last_line
            assert str(workspace / "data" / "card_packs.yaml") in last_line

        def test_パック定義に_packs_が無いとき_標準エラー出力の末尾にtoplevel_packs_key_is_requiredが表示される(
            self, workspace, write_yaml_file, run_generator
        ):
            write_yaml_file(workspace / "data" / "card_packs.yaml", {"TST_UNKNOWN": []})

            result = run_generator("generate_card_packs.py")

            last_line = result.stderr.rstrip().splitlines()[-1]
            assert "top-level 'packs' key is required" in last_line

        def test_パック_TSTPACK0001_の_card_id_TST0001_の_copies_が_0_のとき_標準エラー出力にTSTPACK0001TST0001_copies_must_be_a_positive_intが表示される(
            self, write_pack_definitions, build_pack, run_generator
        ):
            write_pack_definitions(
                [build_pack(cards=[{"card_id": "TST-0001", "copies": 0}])]
            )

            result = run_generator("generate_card_packs.py")

            assert (
                "TST-PACK-0001/TST-0001: copies must be a positive int"
                in result.stderr
            )


class Testカードパック__生成コマンドの出力:
    class Test正常系:
        class Testpack_id_が_TSTPACK0001_のパックと_TSTPACK0002_のパックの有効なパック定義があるとき:
            def test_実行すると_作られた初期投入_SQL_を適用して読み戻した_pack_id_の一覧が_TSTPACK0001_TSTPACK0002_になる(
                self,
                write_pack_definitions,
                valid_packs,
                run_generator,
                read_applied_pack_ids,
            ):
                write_pack_definitions(valid_packs)

                run_generator("generate_card_packs.py")

                assert read_applied_pack_ids() == ["TST-PACK-0001", "TST-PACK-0002"]

            class Test出力先のディレクトリが無いときに実行すると:
                def test_出力先のディレクトリ内に初期投入_SQL_が作られる(
                    self,
                    write_pack_definitions,
                    valid_packs,
                    remove_output_directories,
                    run_generator,
                    locate_output_file,
                ):
                    write_pack_definitions(valid_packs)
                    remove_output_directories()

                    run_generator("generate_card_packs.py")

                    assert locate_output_file("pack_seed").exists()

                def test_作られた初期投入_SQL_を適用して読み戻した_pack_id_の一覧が_TSTPACK0001_TSTPACK0002_になる(
                    self,
                    write_pack_definitions,
                    valid_packs,
                    remove_output_directories,
                    run_generator,
                    read_applied_pack_ids,
                ):
                    write_pack_definitions(valid_packs)
                    remove_output_directories()

                    run_generator("generate_card_packs.py")

                    assert read_applied_pack_ids() == [
                        "TST-PACK-0001",
                        "TST-PACK-0002",
                    ]

            def test_続けて_2_回実行すると_2_回目の実行で作られた初期投入_SQL_を適用して読み戻した_pack_id_の一覧が_TSTPACK0001_TSTPACK0002_になる(
                self,
                write_pack_definitions,
                valid_packs,
                run_generator,
                read_applied_pack_ids,
            ):
                write_pack_definitions(valid_packs)
                run_generator("generate_card_packs.py")

                second_result = run_generator("generate_card_packs.py")

                assert second_result.returncode == 0
                assert read_applied_pack_ids() == ["TST-PACK-0001", "TST-PACK-0002"]

    class Test異常系:
        def test_パック_TSTPACK0001_の_card_id_TST0001_の_copies_が_0_のパック定義のとき_出力先に何も無い状態で実行すると_初期投入_SQL_が作られない(
            self, write_pack_definitions, build_pack, run_generator, locate_output_file
        ):
            write_pack_definitions(
                [build_pack(cards=[{"card_id": "TST-0001", "copies": 0}])]
            )

            run_generator("generate_card_packs.py")

            assert not locate_output_file("pack_seed").exists()

        def test_TSTPACK0001_と_TSTPACK0002_のパック定義で出力を作ったあと_TSTPACK0003_の_card_id_TST0001_の_copies_が_0_のパック定義に変えたとき_既存の出力があるときに実行すると_既存の初期投入_SQL_を適用して読み戻した_pack_id_の一覧が_TSTPACK0001_TSTPACK0002_になる(
            self,
            write_pack_definitions,
            valid_packs,
            build_pack,
            run_generator,
            read_applied_pack_ids,
        ):
            write_pack_definitions(valid_packs)
            run_generator("generate_card_packs.py")
            write_pack_definitions(
                [
                    *valid_packs,
                    build_pack(
                        pack_id="TST-PACK-0003",
                        cards=[{"card_id": "TST-0001", "copies": 0}],
                    ),
                ]
            )

            run_generator("generate_card_packs.py")

            assert read_applied_pack_ids() == ["TST-PACK-0001", "TST-PACK-0002"]
