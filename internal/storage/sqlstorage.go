package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AntonPashechko/ya-diplom/internal/models"
)

const (
	checkUserExist = "SELECT COUNT(*) FROM users WHERE login = $1"
	createUser     = "INSERT INTO users (login, password) VALUES($1,$2)"
	getUser        = "SELECT user_id, password FROM users WHERE login = $1"
)

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
			user_id uuid DEFAULT uuid_generate_v4 (),
			login VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255),
			PRIMARY KEY (user_id)
        )
    `)
	if err != nil {
		return fmt.Errorf("cannot create users table: %w", err)
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

func (m *MartStorage) IsUserExist(login string) bool {
	var count int
	row := m.conn.QueryRowContext(context.TODO(), checkUserExist, login)
	row.Scan(&count)

	return count > 0
}

func (m *MartStorage) CreateUser(dto models.AuthDTO) (string, error) {

	if err := dto.GeneratePasswordHash(); err != nil {
		return ``, fmt.Errorf("cannot generate password hash: %w", err)
	}

	_, err := m.conn.ExecContext(context.TODO(), createUser, dto.Login, dto.Password)
	if err != nil {
		return ``, fmt.Errorf("cannot execute create request: %w", err)
	}

	var uuid, password string
	row := m.conn.QueryRowContext(context.TODO(), getUser, dto.Login)
	err = row.Scan(&uuid, &password)
	if err != nil {
		return ``, fmt.Errorf("cannot get created user id: %w", err)
	}

	return uuid, nil
}

func (m *MartStorage) Login(dto models.AuthDTO) (string, error) {
	var passwordHash string
	var uuid string

	row := m.conn.QueryRowContext(context.TODO(), getUser, dto.Login)
	err := row.Scan(&uuid, &passwordHash)
	if err != nil {
		return ``, fmt.Errorf("cannot get user: %w", err)
	}

	if !dto.CheckPassword(passwordHash) {
		return ``, fmt.Errorf("bad password")
	}

	return uuid, nil
}
