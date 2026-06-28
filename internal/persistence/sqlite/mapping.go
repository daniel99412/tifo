package sqlite

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// EntityMap stores a TIFO ID ↔ external provider ID mapping.
type EntityMap struct {
	TIFOID     string
	EntityType string // "match", "team", "player", "competition"
	Provider   string
	ExternalID string
	Confidence float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MappingDB is a lightweight SQLite-backed mapping store.
type MappingDB struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

// OpenMappingDB opens (or creates) a SQLite mapping database.
func OpenMappingDB(path string) (*MappingDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open mapping db: %w", err)
	}
	m := &MappingDB{db: db, path: path}
	if err := m.migrate(); err != nil {
		return nil, fmt.Errorf("migrate mapping db: %w", err)
	}
	return m, nil
}

func (m *MappingDB) migrate() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS mappings (
			tifo_id     TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			provider    TEXT NOT NULL,
			external_id TEXT NOT NULL,
			confidence  REAL DEFAULT 1.0,
			created_at  TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (entity_type, provider, external_id)
		);
		CREATE INDEX IF NOT EXISTS idx_mappings_tifo ON mappings(tifo_id);
		CREATE INDEX IF NOT EXISTS idx_mappings_provider ON mappings(provider, external_id);
	`)
	return err
}

// GetByExternalID looks up a TIFO ID from a provider's external ID.
func (m *MappingDB) GetByExternalID(entityType, provider, externalID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tifoID string
	var confidence float64
	err := m.db.QueryRow(
		`SELECT tifo_id, confidence FROM mappings WHERE entity_type = ? AND provider = ? AND external_id = ?`,
		entityType, provider, externalID,
	).Scan(&tifoID, &confidence)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return tifoID, nil
}

// GetByTIFOID returns all provider mappings for a TIFO ID.
func (m *MappingDB) GetByTIFOID(entityType, tifoID string) ([]EntityMap, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query(
		`SELECT provider, external_id, confidence, created_at, updated_at FROM mappings WHERE entity_type = ? AND tifo_id = ?`,
		entityType, tifoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EntityMap
	for rows.Next() {
		var em EntityMap
		em.TIFOID = tifoID
		em.EntityType = entityType
		if err := rows.Scan(&em.Provider, &em.ExternalID, &em.Confidence, &em.CreatedAt, &em.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, em)
	}
	return out, nil
}

// Set stores or updates a mapping.
func (m *MappingDB) Set(entityType, provider, externalID, tifoID string, confidence float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec(
		`INSERT INTO mappings (entity_type, provider, external_id, tifo_id, confidence, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		 ON CONFLICT(entity_type, provider, external_id) DO UPDATE SET tifo_id = ?, confidence = ?, updated_at = datetime('now')`,
		entityType, provider, externalID, tifoID, confidence,
		tifoID, confidence,
	)
	return err
}

// Close closes the database.
func (m *MappingDB) Close() error {
	return m.db.Close()
}
