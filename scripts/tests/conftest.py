import itertools
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

import psycopg
import pytest
import yaml
from psycopg.rows import dict_row
from testcontainers.community.postgres import PostgresContainer

SCRIPTS_DIRECTORY = Path(__file__).resolve().parent.parent
SCHEMA_PATH = SCRIPTS_DIRECTORY.parent / "db" / "schema.sql"
COPIED_SCRIPT_NAMES = (
    "generation_io.py",
    "generate_cards.py",
    "generate_products.py",
    "generate_card_packs.py",
)
OUTPUT_DIRECTORY_NAMES = ("data/cache", "db/seed", "docs")
OUTPUT_RELATIVE_PATHS = {
    "cards_json": "data/cache/cards_gen.json",
    "cards_document": "docs/CARDS.md",
    "cards_seed": "db/seed/cards_seed.sql",
    "products_json": "data/cache/products_gen.json",
    "initiatives_json": "data/cache/initiatives_gen.json",
    "products_seed": "db/seed/products_seed.sql",
    "initiatives_seed": "db/seed/initiatives_seed.sql",
    "products_document": "docs/PRODUCTS.md",
    "pack_seed": "db/seed/card_packs_seed.sql",
}
FACTION_FILE_STEMS = {
    "SHE": "sd",
    "Tenki": "tenki",
    "Sugar": "sugar",
    "Tuners": "tuners",
    "Neutral": "neutral",
}
CATALOG_FACTIONS = ["SHE", "Tenki", "Sugar", "Tuners"]
CATALOG_INITIATIVE_KINDS = ["routine", "special"]
FIRST_TEST_NUMBER = 9001
POSTGRES_IMAGE = "postgres:16-alpine"
GENERATOR_TIMEOUT_SECONDS = 60
HEADING_PATTERN = re.compile(r"^(#+)\s+(.*)$")
GENERATED_LINE_PATTERN = re.compile(
    r"^Generated (?P<count>\d+) (?P<kind>\w+) → (?P<path>.+)$"
)
BEFORE_FIRST_HEADING_LEVEL = 0
TITLE_LEVEL = 1
SECTION_LEVEL = 2
CATEGORY_LEVEL = 3
TABLE_HEADER_ROW_COUNT = 2
CARD_TYPES_FOR_GAME_CONSTANTS = [
    "Compute",
    "DataResource",
    "Platform",
    "Attachment",
    "Strategy",
    "Reactive",
    "Incident",
]
SUBTYPES_FOR_GAME_CONSTANTS = {
    "Compute": ["VM", "Container", "Orchestrator", "Serverless", "AI/ML"],
    "DataResource": ["Database", "ObjectStorage", "CacheDB"],
}
RESOURCE_GROUPS_FOR_GAME_CONSTANTS = {}
TRIGGER_TYPES_FOR_GAME_LOGIC_CONSTANTS = ["on_deploy"]
EFFECT_OPS_FOR_GAME_LOGIC_CONSTANTS = [
    "draw",
    "gain_budget",
    "heal_damage",
    "apply_buff",
]
BUFF_TYPES_FOR_GAME_LOGIC_CONSTANTS = ["TST_BUFF_A"]
EFFECT_DURATIONS_FOR_GAME_LOGIC_CONSTANTS = ["TST_DURATION_A"]
SELECTOR_KEYWORDS_FOR_GAME_LOGIC_CONSTANTS = ["TST_SELECTOR_A"]
PLAYER_REFS_FOR_GAME_LOGIC_CONSTANTS = ["TST_OWNER_A"]


@pytest.fixture
def workspace(tmp_path):
    """スクリプトを写した一時ディレクトリ。スクリプトの親の親として data/・docs/・db/ を持つ。"""
    root = tmp_path / "work"
    (root / "scripts").mkdir(parents=True)
    for script_name in COPIED_SCRIPT_NAMES:
        shutil.copy(SCRIPTS_DIRECTORY / script_name, root / "scripts" / script_name)
    for directory_name in OUTPUT_DIRECTORY_NAMES:
        (root / directory_name).mkdir(parents=True)
    return root


@pytest.fixture
def common_repository(tmp_path):
    """環境変数 COMMON_REPO が指す一時ディレクトリ。"""
    root = tmp_path / "common"
    (root / "data").mkdir(parents=True)
    return root


@pytest.fixture
def game_constants_path(common_repository):
    """COMMON_REPO の下のゲーム定数のファイルの場所。"""
    return common_repository / "data" / "game_design_constants.yaml"


@pytest.fixture
def battle_repository(tmp_path):
    """環境変数 BATTLE_REPO が指す一時ディレクトリ。"""
    root = tmp_path / "battle"
    (root / "data").mkdir(parents=True)
    return root


@pytest.fixture
def game_logic_constants_path(battle_repository):
    """BATTLE_REPO の下のゲームロジック定数のファイルの場所。"""
    return battle_repository / "data" / "game_logic_constants.yaml"


@pytest.fixture
def faction_names():
    """陣営名を、カード一覧文書の節が並ぶ順に並べたもの。"""
    return list(FACTION_FILE_STEMS)


@pytest.fixture
def locate_faction_file(workspace):
    """陣営名から、その陣営の定義ファイルの場所を返す関数を返す。"""

    def locate_faction_definition_file(faction):
        return workspace / "data" / "cards" / f"{FACTION_FILE_STEMS[faction]}.yaml"

    return locate_faction_definition_file


@pytest.fixture
def locate_output_file(workspace):
    """出力の種類から、生成コマンドの出力先の場所を返す関数を返す。"""

    def locate_generated_file(output_key):
        return workspace / OUTPUT_RELATIVE_PATHS[output_key]

    return locate_generated_file


@pytest.fixture
def remove_output_directories(workspace):
    """生成コマンドの出力先のディレクトリをすべて削除する関数を返す。"""

    def remove_generated_directories():
        for directory_name in OUTPUT_DIRECTORY_NAMES:
            shutil.rmtree(workspace / directory_name)

    return remove_generated_directories


@pytest.fixture
def seed_path(tmp_path):
    """初期投入 SQL を書き出す一時ファイルの場所。"""
    return tmp_path / "seed.sql"


@pytest.fixture
def updated_seed_path(tmp_path):
    """定義を変えて出力し直した初期投入 SQL を書き出す一時ファイルの場所。"""
    return tmp_path / "updated_seed.sql"


@pytest.fixture
def generated_json_path(tmp_path):
    """配布用 JSON を書き出す一時ファイルの場所。"""
    return tmp_path / "generated.json"


@pytest.fixture
def read_json_file():
    """JSON ファイルを読み込む関数を返す。"""

    def read_json_value(path):
        return json.loads(Path(path).read_text(encoding="utf-8"))

    return read_json_value


@pytest.fixture
def read_generated_counts():
    """標準エラー出力の「Generated」の行から、出力先のファイル名ごとの件数を読む関数を返す。

    Args:
        kind: 件数の単位を示す語 (cards・products・initiatives)。
    """

    def read_counts_by_file_name(stderr, kind):
        matches = [
            GENERATED_LINE_PATTERN.match(line) for line in stderr.splitlines()
        ]
        return {
            Path(match.group("path")).name: match.group("count")
            for match in matches
            if match and match.group("kind") == kind
        }

    return read_counts_by_file_name


@pytest.fixture
def write_yaml_file():
    """YAML ファイルを書き出す関数を返す。"""

    def dump_yaml_value(path, value):
        Path(path).parent.mkdir(parents=True, exist_ok=True)
        Path(path).write_text(
            yaml.safe_dump(value, allow_unicode=True, sort_keys=False),
            encoding="utf-8",
        )

    return dump_yaml_value


@pytest.fixture
def write_game_constants(game_constants_path, write_yaml_file):
    """COMMON_REPO の下にゲーム定数を書き出す関数を返す。"""

    def write_game_constants_file(
        *,
        card_types=CARD_TYPES_FOR_GAME_CONSTANTS,
        subtypes=SUBTYPES_FOR_GAME_CONSTANTS,
        resource_groups=RESOURCE_GROUPS_FOR_GAME_CONSTANTS,
    ):
        write_yaml_file(
            game_constants_path,
            {
                "card_types": card_types,
                "subtypes": subtypes,
                "resource_groups": resource_groups,
            },
        )

    return write_game_constants_file


@pytest.fixture
def write_game_logic_constants(game_logic_constants_path, write_yaml_file):
    """BATTLE_REPO の下にゲームロジック定数を書き出す関数を返す。"""

    def write_game_logic_constants_file(
        *,
        trigger_types=TRIGGER_TYPES_FOR_GAME_LOGIC_CONSTANTS,
        effect_ops=EFFECT_OPS_FOR_GAME_LOGIC_CONSTANTS,
        buff_types=BUFF_TYPES_FOR_GAME_LOGIC_CONSTANTS,
        effect_durations=EFFECT_DURATIONS_FOR_GAME_LOGIC_CONSTANTS,
        selector_keywords=SELECTOR_KEYWORDS_FOR_GAME_LOGIC_CONSTANTS,
        player_refs=PLAYER_REFS_FOR_GAME_LOGIC_CONSTANTS,
    ):
        write_yaml_file(
            game_logic_constants_path,
            {
                "trigger_types": trigger_types,
                "effect_ops": effect_ops,
                "buff_types": buff_types,
                "effect_durations": effect_durations,
                "selector_keywords": selector_keywords,
                "player_refs": player_refs,
            },
        )

    return write_game_logic_constants_file


@pytest.fixture
def write_card_definitions(locate_faction_file, write_yaml_file):
    """5 つの陣営の定義ファイルを書き出す関数を返す。陣営ごとのカードは辞書で渡す。"""

    def write_faction_definition_files(cards_by_faction):
        for faction in FACTION_FILE_STEMS:
            write_yaml_file(
                locate_faction_file(faction),
                {
                    "faction": faction,
                    "display_name": faction,
                    "cards": cards_by_faction.get(faction, []),
                },
            )

    return write_faction_definition_files


@pytest.fixture
def write_product_definitions(workspace, write_yaml_file):
    """プロダクト定義と施策定義のファイルを書き出す関数を返す。"""

    def write_product_and_initiative_files(products, initiatives):
        write_yaml_file(workspace / "data" / "products.yaml", {"products": products})
        write_yaml_file(
            workspace / "data" / "initiatives.yaml", {"initiatives": initiatives}
        )

    return write_product_and_initiative_files


@pytest.fixture
def write_pack_definitions(workspace, write_yaml_file):
    """パック定義のファイルを書き出す関数を返す。"""

    def write_pack_definition_file(packs):
        write_yaml_file(workspace / "data" / "card_packs.yaml", {"packs": packs})

    return write_pack_definition_file


@pytest.fixture
def run_generator(workspace, common_repository, battle_repository):
    """一時ディレクトリに写したスクリプトを引数なしで実行する関数を返す。

    Args:
        script_name: 実行するスクリプトのファイル名。
        sets_common_repo: false のとき、環境変数 COMMON_REPO を子プロセスに渡さない。
        sets_battle_repo: false のとき、環境変数 BATTLE_REPO を子プロセスに渡さない。
    """

    def run_generator_script(
        script_name, *, sets_common_repo=True, sets_battle_repo=True
    ):
        environment = {
            name: value
            for name, value in os.environ.items()
            if name not in ("COMMON_REPO", "BATTLE_REPO")
        }
        environment["PYTHONIOENCODING"] = "utf-8"
        environment.update(
            {"COMMON_REPO": str(common_repository)} if sets_common_repo else {}
        )
        environment.update(
            {"BATTLE_REPO": str(battle_repository)} if sets_battle_repo else {}
        )
        return subprocess.run(
            [sys.executable, str(workspace / "scripts" / script_name)],
            cwd=workspace,
            env=environment,
            capture_output=True,
            text=True,
            encoding="utf-8",
            check=False,
            timeout=GENERATOR_TIMEOUT_SECONDS,
        )

    return run_generator_script


@pytest.fixture(scope="session")
def postgres_connection_url():
    """テストコンテナで起動した PostgreSQL の接続先。Docker が使えないときは失敗する。"""
    with PostgresContainer(POSTGRES_IMAGE, driver=None) as container:
        yield container.get_connection_url()


@pytest.fixture
def database(postgres_connection_url):
    """db/schema.sql を適用した、空のスキーマの PostgreSQL への接続。"""
    with psycopg.connect(
        postgres_connection_url, autocommit=True, row_factory=dict_row
    ) as connection:
        connection.execute("DROP SCHEMA IF EXISTS card CASCADE")
        connection.execute(SCHEMA_PATH.read_text(encoding="utf-8"))
        yield connection


@pytest.fixture
def apply_sql_file(database):
    """SQL ファイルを DB に適用する関数を返す。"""

    def apply_sql_file_to_database(path):
        database.execute(Path(path).read_text(encoding="utf-8"))

    return apply_sql_file_to_database


@pytest.fixture
def apply_sql_lines(database):
    """SQL の行のリストを DB に適用する関数を返す。"""

    def apply_sql_lines_to_database(lines):
        database.execute("\n".join(lines))

    return apply_sql_lines_to_database


@pytest.fixture
def select_rows(database):
    """SELECT の結果を、列名をキーにした辞書のリストで返す関数を返す。"""

    def select_rows_from_database(query):
        return database.execute(query).fetchall()

    return select_rows_from_database


@pytest.fixture
def insert_product_row(database):
    """施策の親になるプロダクトの行を DB へ直接挿入する関数を返す。"""

    def insert_parent_product_row(product_id, faction):
        database.execute(
            "INSERT INTO card.products (product_id, faction, product_name) "
            "VALUES (%s, %s, %s)",
            (product_id, faction, f"{product_id}の親"),
        )

    return insert_parent_product_row


@pytest.fixture
def build_card():
    """検証を通るカード定義を組み立てる関数を返す。

    Args:
        omit: 取り除く項目名。
        fields: 上書きする項目。
    """

    def build_valid_card(*, omit=(), **fields):
        card = {
            "card_no": 1,
            "card_id": "SH-9001",
            "card_name": "TSTカードA",
            "const_name": "TstCardA",
            "card_type": "Platform",
            "resizable": False,
            "elastic": False,
            "restriction": "unlimited",
            "is_active": True,
            "stats": {},
        }
        card.update(fields)
        return {name: value for name, value in card.items() if name not in omit}

    return build_valid_card


@pytest.fixture
def build_compute_card(build_card):
    """Compute のカード定義を組み立てる関数を返す。"""

    def build_valid_compute_card(*, omit=(), **fields):
        return build_card(
            omit=omit,
            **{
                "card_type": "Compute",
                "subtype": "VM",
                "stats": {
                    "throughput": 120,
                    "availability": 99,
                    "maintenance_cost": 15,
                    "sla_penalty": 25,
                },
                **fields,
            },
        )

    return build_valid_compute_card


@pytest.fixture
def build_data_resource_card(build_card):
    """DataResource のカード定義を組み立てる関数を返す。"""

    def build_valid_data_resource_card(**fields):
        return build_card(
            **{
                "card_type": "DataResource",
                "subtype": "Database",
                "stats": {
                    "yield": 80,
                    "availability": 99,
                    "maintenance_cost": 15,
                    "sla_penalty": 25,
                },
                **fields,
            },
        )

    return build_valid_data_resource_card


@pytest.fixture
def build_card_of_type(build_card):
    """card_type に合う stats を持つカード定義を組み立てる関数を返す。subtype は fields で渡す。"""
    stats_by_card_type = {
        "Compute": {
            "throughput": 120,
            "availability": 99,
            "maintenance_cost": 15,
            "sla_penalty": 25,
        },
        "DataResource": {
            "yield": 80,
            "availability": 99,
            "maintenance_cost": 15,
            "sla_penalty": 25,
        },
    }

    def build_valid_card_of_type(card_type, *, omit=(), **fields):
        return build_card(
            omit=omit,
            **{
                "card_type": card_type,
                "stats": stats_by_card_type.get(card_type, {}),
                **fields,
            },
        )

    return build_valid_card_of_type


@pytest.fixture
def build_effect_taxonomy():
    """ゲーム定数とゲームロジック定数から読み込む形の、効果に書ける値の定義を組み立てる関数を返す。

    Args:
        card_types: ゲーム定数の card_types。
        subtypes_by_card_type: card_type ごとのゲーム定数の subtypes。
        triggers: ゲームロジック定数の trigger_types。
        ops: ゲームロジック定数の effect_ops。
        buffs: ゲームロジック定数の buff_types。
        durations: ゲームロジック定数の effect_durations。
        selector_keywords: ゲームロジック定数の selector_keywords。
        selector_owners: ゲームロジック定数の player_refs。
    """
    import generate_cards

    def build_taxonomy_from_values(
        *,
        card_types=(),
        subtypes_by_card_type=None,
        triggers=(),
        ops=(),
        buffs=(),
        durations=(),
        selector_keywords=(),
        selector_owners=(),
    ):
        return generate_cards.EffectTaxonomy(
            card_types=frozenset(card_types),
            subtypes_by_card_type={
                card_type: frozenset(subtypes)
                for card_type, subtypes in (subtypes_by_card_type or {}).items()
            },
            resource_groups={},
            triggers=frozenset(triggers),
            ops=frozenset(ops),
            buffs=frozenset(buffs),
            durations=frozenset(durations),
            selector_keywords=frozenset(selector_keywords),
            selector_owners=frozenset(selector_owners),
        )

    return build_taxonomy_from_values


@pytest.fixture
def taxonomy(build_effect_taxonomy):
    """カード定義の検証に使う、カードタイプと subtype の定義。"""
    return build_effect_taxonomy(
        card_types=CARD_TYPES_FOR_GAME_CONSTANTS,
        subtypes_by_card_type=SUBTYPES_FOR_GAME_CONSTANTS,
    )


@pytest.fixture
def has_error():
    """誤りの一覧に、ラベルで始まり文言を含む要素があるかを調べる関数を返す。

    Args:
        errors: 検証が返した誤りの一覧。
        phrases: 同じ要素に含まれる文言。
        label: 要素の先頭のラベル。空文字のときはラベルを問わない。
    """

    def has_error_matching(errors, *phrases, label=""):
        return any(
            error.startswith(label) and all(phrase in error for phrase in phrases)
            for error in errors
        )

    return has_error_matching


@pytest.fixture
def build_pack():
    """検証を通るパック定義を組み立てる関数を返す。"""

    def build_valid_pack(*, omit=(), **fields):
        pack = {
            "pack_id": "TST-PACK-0001",
            "description": "TST説明A",
            "is_active": True,
            "cards": [{"card_id": "TST-0001", "copies": 1}],
        }
        pack.update(fields)
        return {name: value for name, value in pack.items() if name not in omit}

    return build_valid_pack


@pytest.fixture
def build_product():
    """検証を通るプロダクト定義を組み立てる関数を返す。"""

    def build_valid_product(**fields):
        return {
            "product_id": "PD-9001",
            "faction": "SHE",
            "product_name": "TSTプロダクトA",
            **fields,
        }

    return build_valid_product


@pytest.fixture
def build_initiative():
    """検証を通る施策定義を組み立てる関数を返す。"""

    def build_valid_initiative(**fields):
        return {
            "initiative_id": "IN-9001",
            "product_id": "PD-9001",
            "kind": "routine",
            "name": "TST施策A",
            "insight_cost": 100,
            "effect_text": "TST効果A",
            "effect": {"ops": [{"draw": {"count": 1}}]},
            **fields,
        }

    return build_valid_initiative


@pytest.fixture
def build_catalog(build_product, build_initiative):
    """SHE・Tenki・Sugar・Tuners の順に 1 件ずつのプロダクトと、各プロダクトのルーチンとスペシャルの施策を組み立てる関数を返す。"""

    def build_valid_catalog(product_count=len(CATALOG_FACTIONS)):
        products = [
            build_product(
                product_id=f"PD-{FIRST_TEST_NUMBER + index}",
                faction=CATALOG_FACTIONS[index % len(CATALOG_FACTIONS)],
                product_name=f"TSTプロダクト{chr(ord('A') + index)}",
            )
            for index in range(product_count)
        ]
        initiatives = [
            build_initiative(
                initiative_id=(
                    f"IN-{FIRST_TEST_NUMBER + len(CATALOG_INITIATIVE_KINDS) * index + offset}"
                ),
                product_id=f"PD-{FIRST_TEST_NUMBER + index}",
                kind=kind,
                name=(
                    "TST施策"
                    f"{chr(ord('A') + len(CATALOG_INITIATIVE_KINDS) * index + offset)}"
                ),
            )
            for index in range(product_count)
            for offset, kind in enumerate(CATALOG_INITIATIVE_KINDS)
        ]
        return products, initiatives

    return build_valid_catalog


@pytest.fixture
def parse_markdown_file():
    """Markdown 文書を、見出しごとのブロックと表のセルに分けて読む関数を返す。

    Returns:
        ブロックのリスト。各ブロックは見出しの level (0 は見出しの前)、title、
        見出しの直下の表を除く text_lines、表の tables (header と rows) を持つ。
        セルの前後の空白と太字の ** は取り除く。
    """

    def split_table_row(line):
        return [
            cell.strip().replace("**", "")
            for cell in line.strip().strip("|").split("|")
        ]

    def build_table_from_lines(table_lines):
        rows = [split_table_row(line) for line in table_lines]
        return {"header": rows[0], "rows": rows[TABLE_HEADER_ROW_COUNT:]}

    def build_block_from_lines(lines):
        heading = HEADING_PATTERN.match(lines[0])
        body = lines[1:] if heading else lines
        table_groups = [
            list(group)
            for is_table, group in itertools.groupby(
                body, key=lambda line: line.startswith("|")
            )
            if is_table
        ]
        return {
            "level": len(heading.group(1)) if heading else BEFORE_FIRST_HEADING_LEVEL,
            "title": heading.group(2).strip() if heading else "",
            "text_lines": [line for line in body if not line.startswith("|")],
            "tables": [build_table_from_lines(group) for group in table_groups],
        }

    def parse_markdown_blocks(path):
        lines = Path(path).read_text(encoding="utf-8").splitlines()
        heading_indexes = [
            index for index, line in enumerate(lines) if HEADING_PATTERN.match(line)
        ]
        starts = [0, *heading_indexes]
        ends = [*heading_indexes, len(lines)]
        return [
            build_block_from_lines(lines[start:end])
            for start, end in zip(starts, ends, strict=True)
            if lines[start:end]
        ]

    return parse_markdown_blocks


@pytest.fixture
def select_table_rows():
    """表を、列名をキーにした辞書のリストにする関数を返す。"""

    def select_rows_of_table(table):
        return [dict(zip(table["header"], row, strict=True)) for row in table["rows"]]

    return select_rows_of_table


@pytest.fixture
def select_title_headings():
    """文書の題名の見出しの title を集める関数を返す。"""

    def select_titles_of_title_level(document):
        return [block["title"] for block in document if block["level"] == TITLE_LEVEL]

    return select_titles_of_title_level


@pytest.fixture
def select_section_titles():
    """文書の節の見出しの title を集める関数を返す。"""

    def select_titles_of_section_level(document):
        return [block["title"] for block in document if block["level"] == SECTION_LEVEL]

    return select_titles_of_section_level


@pytest.fixture
def select_opening_lines():
    """文書の冒頭 (最初の節の見出しより前) の表以外の行を、太字の記号と前後の空白を除いて集める関数を返す。"""

    def select_lines_before_first_section(document):
        first_section_index = next(
            index for index, block in enumerate(document) if block["level"] >= SECTION_LEVEL
        )
        return [
            line.replace("*", "").strip()
            for block in document[:first_section_index]
            for line in block["text_lines"]
        ]

    return select_lines_before_first_section


@pytest.fixture
def select_section_blocks():
    """節の見出しの title が指定の語で始まる節のブロックを、次の節の手前まで集める関数を返す。"""

    def select_blocks_of_section(document, title_prefix):
        starts = [
            index
            for index, block in enumerate(document)
            if block["level"] == SECTION_LEVEL
            and block["title"].startswith(title_prefix)
        ]
        following = document[starts[0] + 1 :]
        section_length = next(
            (
                index
                for index, block in enumerate(following)
                if block["level"] <= SECTION_LEVEL
            ),
            len(following),
        )
        return [document[starts[0]], *following[:section_length]]

    return select_blocks_of_section


@pytest.fixture
def select_category_blocks():
    """節のブロックから、カテゴリの見出しのブロックを集める関数を返す。"""

    def select_blocks_of_category_level(section):
        return [block for block in section if block["level"] == CATEGORY_LEVEL]

    return select_blocks_of_category_level


@pytest.fixture
def select_category_table(select_category_blocks):
    """節のブロックから、見出しの title が指定のカテゴリ名で始まる表を返す関数を返す。"""

    def select_table_of_category(section, category):
        return next(
            block["tables"][0]
            for block in select_category_blocks(section)
            if block["title"].startswith(category)
        )

    return select_table_of_category


@pytest.fixture
def select_first_table_of_titled_block():
    """文書から、見出しの title が指定の語で始まるブロックの先頭の表を返す関数を返す。"""

    def select_first_table(document, title_prefix):
        return next(
            block["tables"][0]
            for block in document
            if block["title"].startswith(title_prefix)
        )

    return select_first_table


@pytest.fixture
def render_card_list(tmp_path, parse_markdown_file):
    """陣営ごとのカードからカード一覧文書を書き出し、読み込んだ結果を返す関数を返す。

    Args:
        cards_by_faction: 陣営名からカードのリストへの辞書。陣営の表示名は陣営名と同じにする。
    """
    import generate_cards

    def render_card_list_document(cards_by_faction):
        faction_data = {
            faction: {
                "display_name": faction,
                "cards": [{**card, "faction": faction} for card in cards],
            }
            for faction, cards in cards_by_faction.items()
        }
        all_cards = [
            card for data in faction_data.values() for card in data["cards"]
        ]
        out_path = tmp_path / "CARDS.md"
        generate_cards.generate_cards_md(all_cards, faction_data, out_path=out_path)
        return parse_markdown_file(out_path)

    return render_card_list_document


@pytest.fixture
def render_product_list(tmp_path, parse_markdown_file):
    """プロダクトの一覧からプロダクト一覧文書を書き出し、読み込んだ結果を返す関数を返す。"""
    import generate_products

    def render_product_list_document(products, initiatives):
        out_path = tmp_path / "PRODUCTS.md"
        generate_products.generate_products_md(products, initiatives, out_path=out_path)
        return parse_markdown_file(out_path)

    return render_product_list_document
