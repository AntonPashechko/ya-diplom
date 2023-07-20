package main

import (
	"context"
	"log"

	"github.com/AntonPashechko/ya-diplom/internal/app"
	"github.com/AntonPashechko/ya-diplom/internal/checker"
	"github.com/AntonPashechko/ya-diplom/internal/config"
	"github.com/AntonPashechko/ya-diplom/internal/logger"
	"github.com/AntonPashechko/ya-diplom/internal/storage"
)

func main() {
	//Инициализируем синглтон логера
	if err := logger.Initialize("info"); err != nil {
		log.Fatalf("cannot initialize logger: %s\n", err)
	}

	cfg, err := config.Create()
	if err != nil {
		log.Fatalf("cannot load config: %s\n", err)
	}

	storage, err := storage.NewMartStorage(cfg.DataBaseDNS)
	if err != nil {
		log.Fatalf("cannot create db store: %s\n", err)
	}
	defer storage.Close()

	app, err := app.Create(cfg, storage)
	if err != nil {
		logger.Error("cannot create app: %s", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accrualChecker := checker.NewAccrualChecker(cfg, storage)
	go accrualChecker.Work(ctx)

	go app.Run()

	logger.Info("Running server: address %s", cfg.Endpoint)

	<-app.ServerDone()

	if err := app.Shutdown(); err != nil {
		logger.Error("Server shutdown failed: %s", err)
	}

	logger.Info("Server has been shutdown")
}
