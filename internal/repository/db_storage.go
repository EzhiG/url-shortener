package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/EzhiG/url-shortener/internal/model"
	"github.com/EzhiG/url-shortener/internal/shortener"
	"github.com/EzhiG/url-shortener/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type sqlExecutor interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	storage := &DBStorage{db: db}

	if err := storage.applyMigrations(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *DBStorage) applyMigrations() error {
	dbDriver, err := postgres.WithInstance(s.db, &postgres.Config{})
	if err != nil {
		return err
	}

	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func (s *DBStorage) Close() error {
	return s.db.Close()
}

func (s *DBStorage) Check() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return s.db.PingContext(ctx)
}

func (s *DBStorage) Save(id, url, userID string) error {
	res, err := s.db.ExecContext(context.Background(), "INSERT INTO urls (short, original, user_id) VALUES ($1, $2, $3) ON CONFLICT (original, user_id) DO NOTHING", id, url, userID)
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

	existedID, ok := s.getByOriginal(context.Background(), s.db, url, userID)

	if !ok {
		return shortener.ErrURLNotFound
	}

	return shortener.NewURLConflictError([]shortener.URLConflictItem{{ShortURL: existedID, OriginalURL: url}})
}

func (s *DBStorage) SaveMany(records map[string]string, userID string) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	ctx := context.Background()
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (short, original, user_id) VALUES($1, $2, $3) ON CONFLICT (original, user_id) DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()

	var conflicts []shortener.URLConflictItem

	for id, url := range records {
		res, err := stmt.ExecContext(ctx, id, url, userID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				return shortener.ErrIDCollision
			}

			return err
		}

		rows, _ := res.RowsAffected()
		if rows > 0 {
			continue
		}

		existedID, ok := s.getByOriginal(ctx, tx, url, userID)
		if !ok {
			return shortener.ErrURLNotFound
		}

		conflicts = append(conflicts, shortener.URLConflictItem{ShortURL: existedID, OriginalURL: url})
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	if len(conflicts) > 0 {
		return shortener.NewURLConflictError(conflicts)
	}

	return nil
}

func (s *DBStorage) Get(id string) (model.URLRecord, bool) {
	var record model.URLRecord
	err := s.db.QueryRowContext(context.Background(), "SELECT original, user_id FROM urls WHERE short = $1", id).Scan(&record.OriginalURL, &record.UserID)
	if err != nil {
		return record, false
	}

	record.ShortURL = id
	return record, true
}

func (s *DBStorage) GetByUserID(userID string) ([]model.URLRecord, error) {
	rows, err := s.db.QueryContext(context.Background(), "SELECT original, short, user_id FROM urls WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]model.URLRecord, 0)
	for rows.Next() {
		var record model.URLRecord
		err := rows.Scan(&record.OriginalURL, &record.ShortURL, &record.UserID)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (s *DBStorage) getByOriginal(ctx context.Context, exec sqlExecutor, original string, userID string) (string, bool) {
	var id string
	err := exec.QueryRowContext(ctx, "SELECT short FROM urls WHERE original = $1 AND user_id = $2", original, userID).Scan(&id)
	if err != nil {
		return "", false
	}

	return id, true
}
