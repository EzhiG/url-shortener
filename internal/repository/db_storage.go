package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EzhiG/url-shortener/internal/shortener"
	"github.com/jackc/pgx/v5/pgconn"
)

const PGUniqueViolationErrorCode = "23505"

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
	_, err := s.db.ExecContext(context.Background(), "INSERT INTO urls (short, original) VALUES ($1, $2)", id, url)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGUniqueViolationErrorCode {
			return shortener.ErrIDCollision
		}

		return err
	}

	return nil
}

func (s *DBStorage) Get(id string) (string, bool) {
	var url string
	err := s.db.QueryRowContext(context.Background(), "SELECT original FROM urls WHERE short = $1", id).Scan(&url)
	if err != nil {
		return "", false
	}

	return url, true
}
