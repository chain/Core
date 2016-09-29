package txdb

import (
	"context"

	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/state"
)

// A Store encapsulates storage for blockchain validation.
// It satisfies the interface protocol.Store, and provides additional
// methods for querying current data.
type Store struct {
	db pg.DB
}

var _ protocol.Store = (*Store)(nil)

// NewStore creates and returns a new Store object.
//
// For testing purposes, it is usually much faster
// and more convenient to use package chain/protocol/memstore
// instead.
func NewStore(db pg.DB) *Store {
	return &Store{db: db}
}

// Height returns the height of the blockchain.
func (s *Store) Height(ctx context.Context) (uint64, error) {
	const q = `SELECT COALESCE(MAX(height), 0) FROM blocks`
	var height uint64
	err := s.db.QueryRow(ctx, q).Scan(&height)
	return height, errors.Wrap(err, "max height sql query")
}

// GetBlock looks up the block with the provided block height.
// If no block is found at that height, it returns an error that
// wraps sql.ErrNoRows.
func (s *Store) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	const q = `SELECT data FROM blocks WHERE height = $1`
	var b bc.Block

	err := s.db.QueryRow(ctx, q, height).Scan(&b)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	return &b, nil
}

// LatestSnapshot returns the most recent state snapshot stored in
// the database and its corresponding block height.
func (s *Store) LatestSnapshot(ctx context.Context) (*state.Snapshot, uint64, error) {
	return getStateSnapshot(ctx, s.db)
}

// SaveBlock persists a new block in the database.
func (s *Store) SaveBlock(ctx context.Context, block *bc.Block) error {
	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (block_hash) DO NOTHING
	`
	_, err := s.db.Exec(ctx, q, block.Hash(), block.Height, block, &block.BlockHeader)
	return errors.Wrap(err, "insert block")
}

// SaveSnapshot saves a state snapshot to the database.
func (s *Store) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	err := storeStateSnapshot(ctx, s.db, snapshot, height)
	return errors.Wrap(err, "saving state tree")
}

func (s *Store) FinalizeBlock(ctx context.Context, height uint64) error {
	_, err := s.db.Exec(ctx, `SELECT pg_notify('newblock', $1)`, height)
	return err
}
