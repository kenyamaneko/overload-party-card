package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.DeckRepository = (*PostgresDeckRepository)(nil)

// PostgresDeckRepository は PostgreSQL を使用した DeckRepository の実装です。
type PostgresDeckRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDeckRepository は PostgresDeckRepository を生成します。
func NewPostgresDeckRepository(pool *pgxpool.Pool) *PostgresDeckRepository {
	return &PostgresDeckRepository{pool: pool}
}

// Create はデッキとそのカード構成を同時に作成し、採番したデッキ ID を返します。
func (r *PostgresDeckRepository) Create(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) (int64, error) {
	var deckID int64
	err := runInTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO card.decks (player_id, deck_name, faction, product_id, routine_id, special_id, playmat_no, sleeve_no, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING deck_id`,
			deck.PlayerID,
			deck.DeckName,
			deck.Faction,
			deck.ProductID,
			deck.RoutineID,
			deck.SpecialID,
			deck.PlaymatNo,
			deck.SleeveNo,
			deck.CreatedAt,
			deck.UpdatedAt,
		).Scan(&deckID)
		if err != nil {
			return fmt.Errorf("insert deck: %w", err)
		}
		return insertDeckCards(ctx, tx, deck.PlayerID, deckID, deckCardEntries)
	})
	if err != nil {
		return 0, err
	}
	return deckID, nil
}

// FindByPlayerID は指定プレイヤーの全デッキを更新日時の降順で返します。
func (r *PostgresDeckRepository) FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, deck_name, faction, product_id, routine_id, special_id, playmat_no, sleeve_no, created_at, updated_at
		 FROM card.decks WHERE player_id = $1 ORDER BY updated_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query decks: %w", err)
	}
	defer rows.Close()

	decks := make([]*domain.Deck, 0)
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, fmt.Errorf("scan deck: %w", err)
		}
		decks = append(decks, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decks: %w", err)
	}
	return decks, nil
}

// FindByID は指定プレイヤーの指定デッキを返します。
// 該当するデッキが無い場合は port.ErrNotFound を返します。
func (r *PostgresDeckRepository) FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT player_id, deck_id, deck_name, faction, product_id, routine_id, special_id, playmat_no, sleeve_no, created_at, updated_at
		 FROM card.decks WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)

	d, err := scanDeck(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("deck %d for player %s: %w", deckID, playerID, port.ErrNotFound)
		}
		return nil, fmt.Errorf("find deck by id: %w", err)
	}
	return d, nil
}

// GetDeckCards は指定デッキのカード構成を返します。
func (r *PostgresDeckRepository) GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, card_id, art_no, count
		 FROM card.deck_cards WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)
	if err != nil {
		return nil, fmt.Errorf("query deck cards: %w", err)
	}
	defer rows.Close()

	cards := make([]domain.DeckCard, 0)
	for rows.Next() {
		var deckCard domain.DeckCard
		if err := rows.Scan(&deckCard.PlayerID, &deckCard.DeckID, &deckCard.CardID, &deckCard.ArtNo, &deckCard.Count); err != nil {
			return nil, fmt.Errorf("scan deck card: %w", err)
		}
		cards = append(cards, deckCard)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deck cards: %w", err)
	}
	return cards, nil
}

// Update はデッキを更新し、カード構成を指定の内容に差し替えます。
// 該当するデッキが無い場合は port.ErrNotFound を返します。
func (r *PostgresDeckRepository) Update(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) error {
	return runInTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`DELETE FROM card.deck_cards WHERE player_id = $1 AND deck_id = $2`,
			deck.PlayerID, deck.DeckID,
		)
		if err != nil {
			return fmt.Errorf("delete old deck cards: %w", err)
		}

		tag, err := tx.Exec(ctx,
			`UPDATE card.decks SET deck_name = $1, faction = $2, product_id = $3, routine_id = $4, special_id = $5, playmat_no = $6, sleeve_no = $7, updated_at = $8
			 WHERE player_id = $9 AND deck_id = $10`,
			deck.DeckName,
			deck.Faction,
			deck.ProductID,
			deck.RoutineID,
			deck.SpecialID,
			deck.PlaymatNo,
			deck.SleeveNo,
			deck.UpdatedAt,
			deck.PlayerID,
			deck.DeckID,
		)
		if err != nil {
			return fmt.Errorf("update deck: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: deck %d", port.ErrNotFound, deck.DeckID)
		}

		return insertDeckCards(ctx, tx, deck.PlayerID, deck.DeckID, deckCardEntries)
	})
}

// Delete は指定デッキとそのカード構成を削除します。
// 該当するデッキが無い場合は port.ErrNotFound を返します。
func (r *PostgresDeckRepository) Delete(ctx context.Context, playerID string, deckID int64) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM card.decks WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: deck %d", port.ErrNotFound, deckID)
	}
	return nil
}

func scanDeck(row pgx.Row) (*domain.Deck, error) {
	var d domain.Deck
	err := row.Scan(
		&d.PlayerID,
		&d.DeckID,
		&d.DeckName,
		&d.Faction,
		&d.ProductID,
		&d.RoutineID,
		&d.SpecialID,
		&d.PlaymatNo,
		&d.SleeveNo,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func insertDeckCards(ctx context.Context, db dbtx, playerID string, deckID int64, deckCardEntries []domain.DeckCardEntry) error {
	rows := make([][]any, len(deckCardEntries))
	for i, entry := range deckCardEntries {
		rows[i] = []any{playerID, deckID, entry.CardID, entry.ArtNo, entry.Count}
	}

	err := insertRows(ctx, db,
		"card.deck_cards",
		[]string{"player_id", "deck_id", "card_id", "art_no", "count"},
		rows,
		"",
	)
	if err != nil {
		return fmt.Errorf("insert deck cards: %w", err)
	}
	return nil
}
