package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EzhiG/url-shortener/internal/shortener"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBStorage struct {
	db *sql.DB
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}

func (s *DBStorage) Check() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return s.db.PingContext(ctx)
}

func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

func (s *DBStorage) Save(id, url string) error {
	res, err := s.db.ExecContext(context.Background(), "INSERT INTO urls (short, original) VALUES ($1, $2) ON CONFLICT (original) DO NOTHING", id, url)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return shortener.ErrIDCollision
		}

		return err
	}
	rows, _ := res.RowsAffected()

	if rows > 0 {
		return nil
	}

	existedID, ok := s.getByOriginal(url)

	if !ok {
		return shortener.ErrURLNotFound
	}

	return shortener.NewURLConflictError(existedID)
}

func (s *DBStorage) SaveMany(records map[string]string) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	ctx := context.Background()
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (short, original) VALUES($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, url := range records {
		_, err := stmt.ExecContext(ctx, id, url)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return shortener.ErrIDCollision
			}

			return err
		}
	}

	return tx.Commit()
}

func (s *DBStorage) Get(id string) (string, bool) {
	var url string
	err := s.db.QueryRowContext(context.Background(), "SELECT original FROM urls WHERE short = $1", id).Scan(&url)
	if err != nil {
		return "", false
	}

	return url, true
}

func (s *DBStorage) getByOriginal(original string) (string, bool) {
	var id string
	err := s.db.QueryRowContext(context.Background(), "SELECT short FROM urls WHERE original = $1", original).Scan(&id)
	if err != nil {
		return "", false
	}

	return id, true
}
