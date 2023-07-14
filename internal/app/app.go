package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/AntonPashechko/ya-diplom/internal/auth"
	"github.com/AntonPashechko/ya-diplom/internal/compress"
	"github.com/AntonPashechko/ya-diplom/internal/config"
	"github.com/AntonPashechko/ya-diplom/internal/deadline"
	"github.com/AntonPashechko/ya-diplom/internal/handlers"
	"github.com/AntonPashechko/ya-diplom/internal/logger"
	"github.com/AntonPashechko/ya-diplom/internal/storage"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	shutdownTime = 5 * time.Second
)

type App struct {
	server     *http.Server
	storage    *storage.MartStorage
	notifyStop context.CancelFunc
}

func Create(cfg *config.Config) (*App, error) {

	//Инициализируем объект для создания/проверки jwt
	auth.Initialize(cfg)

	storage, err := storage.NewMartStorage(cfg.DataBaseDNS)
	if err != nil {
		return nil, fmt.Errorf("cannot create db store: %w", err)
	}

	//Наш роутер, регистрируем хэндлеры
	router := chi.NewRouter()
	//Подключаем middleware логирования
	router.Use(logger.Middleware)
	//Подключаем middleware декомпрессии
	router.Use(compress.Middleware)
	//Подключаем middleware deadline context
	router.Use(deadline.Middleware)

	martHandler := handlers.NewMartHandler(storage)
	martHandler.Register(router)

	return &App{
		server: &http.Server{
			Addr:    cfg.Endpoint,
			Handler: router,
		},
		storage: storage,
	}, nil
}

func (m *App) Run() {
	if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("cannot listen: %s\n", err)
	}
}

func (m *App) ServerDone() <-chan struct{} {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	m.notifyStop = stop
	return ctx.Done()
}

func (m *App) Shutdown() error {
	defer m.notifyStop()
	defer m.storage.Close()

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTime)
	defer cancel()

	if err := m.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
}
