package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
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
// 採番された deck_id を返します。入力は mutation しません。
func (r *PgDeckRepository) Create(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var deckID int64
	err = tx.QueryRow(ctx,
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
		return 0, fmt.Errorf("insert deck: %w", err)
	}

	cards := buildDeckCards(deck.PlayerID, deckID, deckCardEntries)
	if err := bulkInsertDeckCards(ctx, tx, cards); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return deckID, nil
}

// FindByPlayerID は指定プレイヤーの全デッキを返します。
func (r *PgDeckRepository) FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, deck_name, faction, product_id, routine_id, special_id, playmat_no, sleeve_no, created_at, updated_at
		 FROM card.decks WHERE player_id = $1 ORDER BY updated_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query decks: %w", err)
	}
	defer rows.Close()

	decks := make([]*domain.Deck, 0, 8)
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
func (r *PgDeckRepository) FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
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
func (r *PgDeckRepository) GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, deck_id, card_id, art_no, count
		 FROM card.deck_cards WHERE player_id = $1 AND deck_id = $2`,
		playerID, deckID,
	)
	if err != nil {
		return nil, fmt.Errorf("query deck cards: %w", err)
	}
	defer rows.Close()

	cards := make([]domain.DeckCard, 0, 16)
	for rows.Next() {
		var dc domain.DeckCard
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
// 入力は mutation しません。updated_at は呼び出し側が事前に設定して渡します。
func (r *PgDeckRepository) Update(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx,
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

	cards := buildDeckCards(deck.PlayerID, deck.DeckID, deckCardEntries)
	if err := bulkInsertDeckCards(ctx, tx, cards); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete は指定デッキを削除します。
func (r *PgDeckRepository) Delete(ctx context.Context, playerID string, deckID int64) error {
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

// buildDeckCards は deckCardEntries を deck_cards 行 (PlayerID, DeckID 付き) に変換します。
// 入力 slice は読み取り専用、結果は新規 slice として返します。
func buildDeckCards(playerID string, deckID int64, deckCardEntries []domain.DeckCardEntry) []domain.DeckCard {
	cards := make([]domain.DeckCard, len(deckCardEntries))
	for i, e := range deckCardEntries {
		cards[i] = domain.DeckCard{
			PlayerID: playerID,
			DeckID:   deckID,
			CardID:   e.CardID,
			ArtNo:    e.ArtNo,
			Count:    e.Count,
		}
	}
	return cards
}

func bulkInsertDeckCards(ctx context.Context, db dbtx, cards []domain.DeckCard) error {
	if len(cards) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO card.deck_cards (player_id, deck_id, card_id, art_no, count) VALUES ")

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
