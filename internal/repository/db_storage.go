package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

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

	existedID, ok := s.getByOriginal(context.Background(), s.db, url)

	if !ok {
		return shortener.ErrURLNotFound
	}

	return shortener.NewURLConflictError([]shortener.URLConflictItem{{ShortURL: existedID, OriginalURL: url}})
}

func (s *DBStorage) SaveMany(records map[string]string) error {
	tx, err := s.db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	ctx := context.Background()
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (short, original) VALUES($1, $2) ON CONFLICT (original) DO NOTHING")
	if err != nil {
		return err
	}
	defer stmt.Close()

	var conflicts []shortener.URLConflictItem

	for id, url := range records {
		res, err := stmt.ExecContext(ctx, id, url)
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

		existedID, ok := s.getByOriginal(ctx, tx, url)
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

func (s *DBStorage) Get(id string) (string, bool) {
	var url string
	err := s.db.QueryRowContext(context.Background(), "SELECT original FROM urls WHERE short = $1", id).Scan(&url)
	if err != nil {
		return "", false
	}

	return url, true
}

func (s *DBStorage) getByOriginal(ctx context.Context, exec sqlExecutor, original string) (string, bool) {
	var id string
	err := exec.QueryRowContext(ctx, "SELECT short FROM urls WHERE original = $1", original).Scan(&id)
	if err != nil {
		return "", false
	}

	return id, true
}
