// Package cache embeds the generated card master JSON for local development
// and for tests that seed the in-memory CardCache without touching Postgres.
//
// The JSON is produced by scripts/generate_cards.py from data/cards/*.yaml.
// This package is intentionally tiny so that the module path stays stable
// even when the generator grows new outputs.
package cache

import _ "embed"

//go:embed cards_gen.json
var CardsJSON []byte
