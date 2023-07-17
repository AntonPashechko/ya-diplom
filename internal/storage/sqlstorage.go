package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AntonPashechko/ya-diplom/internal/models"
)

const (
	checkUserExist = `SELECT COUNT(*) FROM users WHERE login = $1`
	createUser     = `INSERT INTO users (login, password) VALUES($1,$2)`
	getUser        = `SELECT id, password FROM users WHERE login = $1`

	getOrderUserID = `SELECT user_id FROM orders WHERE number = $1`
	createOrder    = `INSERT INTO orders (number, user_id) VALUES ($1,$2)`

	getUserOrders = `SELECT o.number, 
			os.status, 
			o.accrual, 
			o.uploaded_at 
		FROM orders o
			LEFT JOIN order_status os ON o.status_id = os.id
		WHERE user_id = $1
		ORDER BY uploaded_at`

	getUserSumAccruals    = `SELECT coalesce(SUM(accrual), 0.00) from orders WHERE user_id = $1`
	getUserSumWithdrawals = `SELECT coalesce(SUM(sum), 0.00) from withdrawals WHERE user_id = $1`

	getUserWithdrawls = `SELECT number, sum, processed_at, 
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY uploaded_at`

	selectUserForUpdate = `SELECT id FROM users WHERE id = $1 for update`

	createWithdraw = `INSERT INTO withdrawals (number, user_id, sum) VALUES ($1,$2,$3)`

	getOrdersForAccrual = `SELECT number 
		FROM orders o 
			LEFT JOIN order_status os ON o.status_id = os.id
		WHERE os.status = 'NEW' OR os.status = 'PROCESSING'`

	updateOrderAccrual = `UPDATE orders 
		SET status_id = (select id from order_status where status = $1), 
		accrual = $2
    	WHERE number = $3`
)

// ErrNotEnoughFunds not enough funds in the account
var ErrNotEnoughFunds = errors.New("not enough funds in the account")

type MartStorage struct {
	conn *sql.DB
}

func NewMartStorage(dns string) (*MartStorage, error) {
	conn, err := sql.Open("pgx", dns)
	if err != nil {
		return nil, fmt.Errorf("cannot create connection db: %w", err)
	}

	storage := &MartStorage{conn: conn}
	if err := storage.applyDBMigrations(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot apply migrations: %w", err)
	}
	return &MartStorage{conn: conn}, nil
}

func (m *MartStorage) applyDBMigrations(ctx context.Context) error {
	tx, err := m.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}

	defer tx.Rollback()

	// это для возможности генерации uuid
	_, err = tx.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		return fmt.Errorf("cannot create uuid extension: %w", err)
	}

	// создаём таблицу для хранения пользователей
	_, err = tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS users (
			id uuid DEFAULT uuid_generate_v4 (),
			login VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255),
			PRIMARY KEY (id)
        )
    `)
	if err != nil {
		return fmt.Errorf("cannot create users table: %w", err)
	}

	// создаём таблицу для хранения статуса обработки заказа
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS order_status (
			id SERIAL,
			status VARCHAR(32) UNIQUE NOT NULL,
			PRIMARY KEY (id)
			)
    `)
	if err != nil {
		return fmt.Errorf("cannot create order_status table: %w", err)
	}

	//Заполним таблицу статусами
	_, err = tx.ExecContext(ctx,
		"INSERT INTO order_status (status) "+
			"VALUES ('NEW'),('PROCESSING'),('INVALID'),('PROCESSED') "+
			"ON CONFLICT (status) DO NOTHING")
	if err != nil {
		return fmt.Errorf("cannot sync order_status: %w", err)
	}

	// создаём таблицу для хранения номеров заказов
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			number VARCHAR(255) UNIQUE NOT NULL,
			user_id uuid,
			status_id int DEFAULT 1,
			accrual double precision DEFAULT 0,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (number),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (status_id) REFERENCES order_status(id)
			)
    `)
	if err != nil {
		return fmt.Errorf("cannot create orders table: %w", err)
	}

	// создаём таблицу для хранения списаний
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS withdrawals (
			number VARCHAR(255) UNIQUE NOT NULL,
			user_id uuid,
			sum double precision,
			processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (number),
			FOREIGN KEY (user_id) REFERENCES users(id)
			)
    `)
	if err != nil {
		return fmt.Errorf("cannot create withdrawals table: %w", err)
	}

	// коммитим транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot comit transaction: %w", err)
	}
	return nil
}

func (m *MartStorage) Close() {
	m.conn.Close()
}

func (m *MartStorage) IsUserExist(ctx context.Context, login string) bool {
	var count int
	row := m.conn.QueryRowContext(ctx, checkUserExist, login)
	row.Scan(&count)

	return count > 0
}

func (m *MartStorage) CreateUser(ctx context.Context, dto models.AuthDTO) (string, error) {

	if err := dto.GeneratePasswordHash(); err != nil {
		return ``, fmt.Errorf("cannot generate password hash: %w", err)
	}

	_, err := m.conn.ExecContext(ctx, createUser, dto.Login, dto.Password)
	if err != nil {
		return ``, fmt.Errorf("cannot execute create request: %w", err)
	}

	var uuid, password string
	row := m.conn.QueryRowContext(ctx, getUser, dto.Login)
	err = row.Scan(&uuid, &password)
	if err != nil {
		return ``, fmt.Errorf("cannot get created user id: %w", err)
	}

	return uuid, nil
}

func (m *MartStorage) Login(ctx context.Context, dto models.AuthDTO) (string, error) {
	var passwordHash string
	var uuid string

	row := m.conn.QueryRowContext(ctx, getUser, dto.Login)
	err := row.Scan(&uuid, &passwordHash)
	if err != nil {
		return ``, fmt.Errorf("cannot get user: %w", err)
	}

	if !dto.CheckPassword(passwordHash) {
		return ``, fmt.Errorf("bad password")
	}

	return uuid, nil
}

func (m *MartStorage) GetExistOrderUser(ctx context.Context, number string) (string, error) {
	var userID string

	row := m.conn.QueryRowContext(ctx, getOrderUserID, number)
	err := row.Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ``, nil
		}
		return ``, fmt.Errorf("cannot check order exist: %w", err)
	}

	return userID, nil
}

func (m *MartStorage) NewOrder(ctx context.Context, number string, userID string) error {

	_, err := m.conn.ExecContext(ctx, createOrder, number, userID)
	if err != nil {
		return fmt.Errorf("cannot execute create order: %w", err)
	}

	return nil
}

func (m *MartStorage) GetUserOrders(ctx context.Context, userID string) ([]models.OrderDTO, error) {
	orders := make([]models.OrderDTO, 0)

	rows, err := m.conn.QueryContext(ctx, getUserOrders, userID)
	if err != nil {
		return nil, fmt.Errorf("cannot query orders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var order models.OrderDTO
		err = rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, fmt.Errorf("cannot scan order row: %w", err)
		}

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("query orders rows contains error: %w", err)
	}
	return orders, nil
}

func (m *MartStorage) GetUserWithdraws(ctx context.Context, userID string) ([]models.WithdrawDTO, error) {
	withdrawls := make([]models.WithdrawDTO, 0)

	rows, err := m.conn.QueryContext(ctx, getUserWithdrawls, userID)
	if err != nil {
		return nil, fmt.Errorf("cannot query withdrawls: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var withdrawl models.WithdrawDTO
		err = rows.Scan(&withdrawl.Order, &withdrawl.Sum, &withdrawl.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("cannot scan withdrawl row: %w", err)
		}

		withdrawls = append(withdrawls, withdrawl)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("query withdrawls rows contains error: %w", err)
	}
	return withdrawls, nil
}

func (m *MartStorage) GetUserBalance(ctx context.Context, userID string) (*models.BalanceDTO, error) {

	balance := &models.BalanceDTO{}

	row := m.conn.QueryRowContext(ctx, getUserSumAccruals, userID)
	err := row.Scan(&balance.Current)
	if err != nil {
		return nil, fmt.Errorf("cannot get user sum accrual: %w", err)
	}

	row = m.conn.QueryRowContext(ctx, getUserSumWithdrawals, userID)
	err = row.Scan(&balance.Withdrawn)
	if err != nil {
		return nil, fmt.Errorf("cannot get user sum withdrawals: %w", err)
	}

	return balance, nil
}

func (m *MartStorage) AddWithdraw(ctx context.Context, dto models.WithdrawDTO, userID string) error {

	//здесь надо безопасно проверять что хватает средств
	// запускаем транзакцию
	tx, err := m.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback()

	//Блокируем пользователя
	var user string
	rowUser := tx.QueryRowContext(ctx, selectUserForUpdate, userID)
	err = rowUser.Scan(&user)
	if err != nil {
		return fmt.Errorf("cannot lock user: %w", err)
	}

	//Проверяем достаточность средств
	var accruals float64
	var withdrawals float64

	row := tx.QueryRowContext(ctx, getUserSumAccruals, userID)
	err = row.Scan(&accruals)
	if err != nil {
		return fmt.Errorf("cannot get user sum accrual: %w", err)
	}

	row = tx.QueryRowContext(ctx, getUserSumWithdrawals, userID)
	err = row.Scan(&withdrawals)
	if err != nil {
		return fmt.Errorf("cannot get user sum withdrawals: %w", err)
	}

	if (accruals - withdrawals) < dto.Sum {
		return ErrNotEnoughFunds
	}

	_, err = tx.ExecContext(ctx, createWithdraw, dto.Order, userID, dto.Sum)
	if err != nil {
		return fmt.Errorf("cannot execute create order: %w", err)
	}

	// завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit the transaction: %w", err)
	}

	return nil
}

// Получения заказов для расчета бонусов в accrual
func (m *MartStorage) GetOrdersForCheck(ctx context.Context) ([]string, error) {
	orderNumbers := make([]string, 0)

	rows, err := m.conn.QueryContext(ctx, getOrdersForAccrual)
	if err != nil {
		return nil, fmt.Errorf("cannot query orders for accrual check: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var number string
		err = rows.Scan(&number)
		if err != nil {
			return nil, fmt.Errorf("cannot scan number: %w", err)
		}

		orderNumbers = append(orderNumbers, number)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("query orders for accrual check rows contains error: %w", err)
	}
	return orderNumbers, nil
}

func (m *MartStorage) UpdateOrderAccrual(ctx context.Context, number string, accrual *models.AccrualDTO) error {

	_, err := m.conn.ExecContext(ctx, updateOrderAccrual, accrual.Status, accrual.Accrual, number)
	if err != nil {
		return fmt.Errorf("cannot exec update order accrual: %w", err)
	}

	return nil
}
