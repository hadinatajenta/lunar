package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

const sopHashSchema = `
CREATE TABLE IF NOT EXISTS sop_page_hashes (
    page_id TEXT NOT NULL,
    region TEXT NOT NULL,
    block_hash TEXT NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (page_id, region)
);
`

type HashStore struct {
	db         *sql.DB
	ensureOnce sync.Once
	ensureErr  error
}

func NewHashStore(db *sql.DB) *HashStore {
	return &HashStore{db: db}
}

func (s *HashStore) ensureSchema(ctx context.Context) error {
	if s.db == nil {
		return errors.New("the SOP hash store database is not configured")
	}
	s.ensureOnce.Do(func() {
		if s.ensureErr = s.migrateLegacyTable(ctx); s.ensureErr != nil {
			return
		}
		_, s.ensureErr = s.db.ExecContext(ctx, sopHashSchema)
	})
	return s.ensureErr
}

func (s *HashStore) migrateLegacyTable(ctx context.Context) error {
	var columnCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('sop_page_hashes')").Scan(&columnCount); err != nil {
		return fmt.Errorf("inspect the SOP hash table: %w", err)
	}
	if columnCount == 0 {
		return nil
	}

	var regionCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('sop_page_hashes') WHERE name = 'region'").Scan(&regionCount); err != nil {
		return fmt.Errorf("inspect the SOP hash table region column: %w", err)
	}
	if regionCount > 0 {
		return nil
	}

	if _, err := s.db.ExecContext(ctx, "ALTER TABLE sop_page_hashes RENAME TO sop_page_hashes_legacy"); err != nil {
		return fmt.Errorf("migrate the legacy SOP hash table: %w", err)
	}
	return nil
}

func (s *HashStore) Get(ctx context.Context, pageID string, region string) (string, bool, error) {
	if err := s.ensureSchema(ctx); err != nil {
		return "", false, err
	}

	var hash string
	err := s.db.QueryRowContext(ctx, "SELECT block_hash FROM sop_page_hashes WHERE page_id = ? AND region = ?", pageID, region).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read the SOP region hash: %w", err)
	}
	return hash, true, nil
}

func (s *HashStore) Set(ctx context.Context, pageID string, region string, hash string) error {
	if err := s.ensureSchema(ctx); err != nil {
		return err
	}

	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO sop_page_hashes (page_id, region, block_hash, updated_at) VALUES (?, ?, ?, ?) ON CONFLICT(page_id, region) DO UPDATE SET block_hash = excluded.block_hash, updated_at = excluded.updated_at",
		pageID,
		region,
		hash,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("store the SOP region hash: %w", err)
	}
	return nil
}
