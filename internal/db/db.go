package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"

	"vshell/internal/crypto"
)

var (
	once     sync.Once
	database *DB
)

type DB struct {
	*sql.DB
	crypto *crypto.CryptoService
}

func New() (*DB, error) {
	var initErr error
	once.Do(func() {
		dbPath, err := dbFilePath()
		if err != nil {
			initErr = err
			return
		}

		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			initErr = err
			return
		}

		sqlDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
		if err != nil {
			initErr = err
			return
		}

		sqlDB.SetMaxOpenConns(1)

		database = &DB{
			DB:     sqlDB,
			crypto: crypto.New(),
		}

		initErr = database.migrate()
	})
	if initErr != nil {
		return nil, initErr
	}
	return database, nil
}

func dbFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config dir: %w", err)
	}
	return filepath.Join(configDir, "vshell", "vshell.db"), nil
}

func (db *DB) Crypto() *crypto.CryptoService {
	return db.crypto
}
