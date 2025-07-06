package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/and161185/loyalty/internal/errs"
	"github.com/and161185/loyalty/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	db *pgxpool.Pool
}

func (store *PostgresStorage) initSchema(ctx context.Context) error {
	const initSchemaQuery = `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS orders (
		number TEXT PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id),
		status TEXT NOT NULL DEFAULT 'NEW',
		accrual NUMERIC,
		uploaded_at TIMESTAMP DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS withdrawals (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id),
		order_number TEXT NOT NULL,
		sum NUMERIC NOT NULL,
		processed_at TIMESTAMP DEFAULT NOW()
	);`

	_, err := store.db.Exec(ctx, initSchemaQuery)
	return err
}

func NewPostgreStorage(ctx context.Context, DatabaseURI string) (*PostgresStorage, error) {
	db, err := pgxpool.New(ctx, DatabaseURI)
	if err != nil {
		return nil, err
	}

	storage := &PostgresStorage{db: db}

	if err := storage.Ping(ctx); err != nil {
		return nil, err
	}

	if err := storage.initSchema(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (store *PostgresStorage) Ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}

func (store *PostgresStorage) CreateUser(ctx context.Context, login string, passwordHash string) error {
	const insertUserQuery = `INSERT INTO users (login, password_hash) VALUES ($1, $2)`

	_, err := store.db.Exec(ctx, insertUserQuery, login, passwordHash)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			// 23505 — уникальное ограничение нарушено
			return errs.ErrLoginAlreadyExists
		}
		return err
	}

	return nil
}

func (s *PostgresStorage) GetUserByLogin(ctx context.Context, login string) (model.User, string, error) {
	const query = `SELECT id, login, password_hash FROM users WHERE login = $1`

	var user model.User
	var hash string

	err := s.db.QueryRow(ctx, query, login).Scan(&user.ID, &user.Login, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, "", errs.ErrUserNotFound
		}
		return model.User{}, "", fmt.Errorf("get user by login: %w", err)
	}

	return user, hash, nil
}

func (s *PostgresStorage) GetUserById(ctx context.Context, id int) (model.User, error) {
	const query = `SELECT id, login FROM users WHERE id = $1`

	var user model.User

	err := s.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errs.ErrUserNotFound
		}
		return model.User{}, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *PostgresStorage) AddOrder(ctx context.Context, user model.User, order model.Order) (int, error) {
	const query = `
		INSERT INTO orders (number, user_id)
		VALUES ($1, $2)
		ON CONFLICT (number) DO NOTHING`

	const checkOrderOwnerQuery = `SELECT user_id, status FROM orders WHERE number = $1`

	orderNumber := order.Number
	userID := user.ID

	cmdTag, err := s.db.Exec(ctx, query, orderNumber, userID)
	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}

	// Если вставка не произошла — заказ уже есть, надо выяснить чей
	if cmdTag.RowsAffected() == 0 {
		var existingUserID int
		var existingStatus string
		err = s.db.QueryRow(ctx, checkOrderOwnerQuery, orderNumber).Scan(&existingUserID, &existingStatus)
		if err != nil {
			return 0, fmt.Errorf("select existing order: %w", err)
		}
		if existingUserID == userID {
			if existingStatus == string(model.Processing) || existingStatus == string(model.Registered) {
				return 202, nil // Уже загружен этим пользователем
			}
			return 200, nil
		}
		return 409, nil // Загружен другим
	}

	return 202, nil // Новый заказ принят
}

func (s *PostgresStorage) GetUserOrders(ctx context.Context, user model.User) ([]model.Order, error) {
	const query = `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := s.db.Query(ctx, query, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get user orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		err := rows.Scan(&o.Number, &o.Status, &o.Accrual, &o.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration: %w", err)
	}

	if len(orders) == 0 {
		return nil, nil // обработаешь в хендлере как 204
	}

	return orders, nil
}

func (s *PostgresStorage) GetUserBalance(ctx context.Context, user model.User) (model.Balance, error) {
	const query = `
		SELECT 
			COALESCE(SUM(accrual), 0) AS total_accrual,
			(SELECT COALESCE(SUM(sum), 0) FROM withdrawals WHERE user_id = $1) AS total_withdrawn
		FROM orders
		WHERE user_id = $1 AND status = 'PROCESSED'
	`

	var accrual, withdrawn float64
	err := s.db.QueryRow(ctx, query, user.ID).Scan(&accrual, &withdrawn)
	if err != nil {
		return model.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	return model.Balance{
		Current:   accrual - withdrawn,
		Withdrawn: withdrawn,
	}, nil
}

func (s *PostgresStorage) WithdrawBalance(ctx context.Context, user model.User, order string, sum float64) error {
	const checkBalanceQuery = `
		SELECT 
			COALESCE(SUM(accrual), 0) - 
			(SELECT COALESCE(SUM(sum), 0) FROM withdrawals WHERE user_id = $1)
		FROM orders
		WHERE user_id = $1 AND status = 'PROCESSED'
	`

	const insertWithdrawalQuery = `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3)
	`

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance float64
	err = tx.QueryRow(ctx, checkBalanceQuery, user.ID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("check balance: %w", err)
	}

	if balance < sum {
		return errs.ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx, insertWithdrawalQuery, user.ID, order, sum)
	if err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetWithdrawals(ctx context.Context, user model.User) ([]model.Withdrawal, error) {
	const query = `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := s.db.Query(ctx, query, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get withdrawals: %w", err)
	}
	defer rows.Close()

	var list []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("scan withdrawal: %w", err)
		}
		list = append(list, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(list) == 0 {
		return nil, nil
	}

	return list, nil
}

func (s *PostgresStorage) GetUnprocessedOrders(ctx context.Context) ([]model.Order, error) {
	const query = `
		SELECT number
		FROM orders	
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get unprocessed orders: %w", err)
	}
	defer rows.Close()

	var list []model.Order
	for rows.Next() {
		var o model.Order
		err := rows.Scan(&o.Number)
		if err != nil {
			return nil, fmt.Errorf("scan nprocessed orders: %w", err)
		}
		list = append(list, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return list, nil
}

func (s *PostgresStorage) UpdateOrder(ctx context.Context, order model.Order) error {
	const query = `
		UPDATE orders 
		SET status = $1, accrual = $2
		WHERE number = $3`

	status := order.Status
	accrual := order.Accrual
	number := order.Number

	_, err := s.db.Exec(ctx, query, status, accrual, number)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	return nil
}
