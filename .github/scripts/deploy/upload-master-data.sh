#!/usr/bin/env bash
# upload-master-data.sh — マスターデータを、サービスから読むバケットへ反映する。
set -euo pipefail

: "${MASTER_DATA_BUCKET:?MASTER_DATA_BUCKET env required}"

gcloud storage cp data/cache/cards_gen.json "gs://${MASTER_DATA_BUCKET}/cards.json"
gcloud storage cp data/cache/initiatives_gen.json "gs://${MASTER_DATA_BUCKET}/initiatives.json"
