# CLAUDE.md - overload-party-card

## 行動制約

- エラーは握りつぶさない
- git tag を手動で打たない（CI が自動作成する）
- TODO スタブを追加しない
- クライアント認証を行わない（ClusterIP のみ、gateway が唯一の呼び出し元）
- カードマスターデータを他サービスに複製しない（本サービスが `card_definitions` の唯一の SSoT）
- SQL の `(player_id, deck_id)` 主キースコープを超える認可チェックを追加しない
- カードデータ変更時は `python3 scripts/generate_cards.py` を実行する
- 型定義変更時は `data/models.yaml` を編集し `python3 scripts/generate_types.py` を実行する
