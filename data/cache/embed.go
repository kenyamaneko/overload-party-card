// Package cache embeds the generated master JSON for local development
// and for tests that seed the in-memory caches without touching Postgres.
//
// The JSON files are produced by scripts/generate_cards.py (data/cards/*.yaml)
// and scripts/generate_products.py (data/products.yaml, data/initiatives.yaml).
// This package is intentionally tiny so that the module path stays stable
// even when the generators grow new outputs.
package cache

import _ "embed"

//go:embed cards_gen.json
var CardsJSON []byte

//go:embed products_gen.json
var ProductsJSON []byte
