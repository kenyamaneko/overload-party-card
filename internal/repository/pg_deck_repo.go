package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.DeckRepo = (*PgDeckRepository)(nil)

// PgDeckRepository は PostgreSQL を使用した DeckRepo の実装です。
type PgDeckRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeckRepository は PgDeckRepository を生成します。
func NewPgDeckRepository(pool *pgxpool.Pool) *PgDeckRepository {
	return &PgDeckRepository{pool: pool}
}

// Create はデッキとそのカード構成をトランザクション内で作成します。
func (r *PgDeckRepository) Create(ctx context.Context, deck *apicard.Deck, cards []apicard.DeckCard) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx,
		`INSERT INTO decks (player_id, deck_name, playmat_no, sleeve_no, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING deck_id`,
		deck.PlayerID,
		deck.DeckName,
		deck.PlaymatNo,
		deck.SleeveNo,
		deck.CreatedAt,
		deck.UpdatedAt,
	).Scan(&deck.DeckID)
	if err != nil {
		return fmt.Errorf("insert deck: %w", err)
	}

	for i := range cards {
		cards[i].DeckID = deck.DeckID
	}

	if err := bulkInsertDeckCards(ctx, tx, cards); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// FindByPlayerID は指定プレイヤーの全デッキを返します。
func (r *PgDeckRepository) FindByPlayerID(ctx context.Context, playerID string) ([]*apicard.Deck, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, deck_name, playmat_no, sleeve_no, created_at, updated_at
		 FROM decks WHERE player_id = $1 ORDER BY updated_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query decks: %w", err)
	}
	defer rows.Close()

	decks := make([]*apicard.Deck, 0, 8)
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
func (r *PgDeckRepository) FindByID(ctx context.Context, playerID string, deckID int64) (*apicard.Deck, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT player_id, deck_id, deck_name, playmat_no, sleeve_no, created_at, updated_at
		 FROM decks WHERE player_id = $1 AND deck_id = $2`,
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
func (r *PgDeckRepository) GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]apicard.DeckCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, card_id, art_no, count
		 FROM deck_cards WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)
	if err != nil {
		return nil, fmt.Errorf("query deck cards: %w", err)
	}
	defer rows.Close()

	cards := make([]apicard.DeckCard, 0, 16)
	for rows.Next() {
		var dc apicard.DeckCard
		if err := rows.Scan(&dc.PlayerID, &dc.DeckID, &dc.CardID, &dc.ArtNo, &dc.Count); err != nil {
			return nil, fmt.Errorf("scan deck card: %w", err)
		}
		cards = append(cards, dc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deck cards: %w", err)
	}
	return cards, nil
}

// Update はデッキとそのカード構成をトランザクション内で更新します。
func (r *PgDeckRepository) Update(ctx context.Context, deck *apicard.Deck, cards []apicard.DeckCard) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
		`DELETE FROM deck_cards WHERE player_id = $1 AND deck_id = $2`,
		deck.PlayerID, deck.DeckID,
	)
	if err != nil {
		return fmt.Errorf("delete old deck cards: %w", err)
	}

	deck.UpdatedAt = time.Now()
	_, err = tx.Exec(ctx,
		`UPDATE decks SET deck_name = $1, playmat_no = $2, sleeve_no = $3, updated_at = $4
		 WHERE player_id = $5 AND deck_id = $6`,
		deck.DeckName,
		deck.PlaymatNo,
		deck.SleeveNo,
		deck.UpdatedAt,
		deck.PlayerID,
		deck.DeckID,
	)
	if err != nil {
		return fmt.Errorf("update deck: %w", err)
	}

	if err := bulkInsertDeckCards(ctx, tx, cards); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete は指定デッキを削除します。
func (r *PgDeckRepository) Delete(ctx context.Context, playerID string, deckID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM decks WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	return nil
}

func scanDeck(row pgx.Row) (*apicard.Deck, error) {
	var d apicard.Deck
	err := row.Scan(
		&d.PlayerID,
		&d.DeckID,
		&d.DeckName,
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

func bulkInsertDeckCards(ctx context.Context, db dbtx, cards []apicard.DeckCard) error {
	if len(cards) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO deck_cards (player_id, deck_id, card_id, art_no, count) VALUES ")

	args := make([]interface{}, 0, len(cards)*5)
	for i, c := range cards {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i*5 + 1
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d)", base, base+1, base+2, base+3, base+4)
		args = append(args, c.PlayerID, c.DeckID, c.CardID, c.ArtNo, c.Count)
	}

	_, err := db.Exec(ctx, sb.String(), args...)
	if err != nil {
		return fmt.Errorf("insert deck cards: %w", err)
	}
	return nil
}
