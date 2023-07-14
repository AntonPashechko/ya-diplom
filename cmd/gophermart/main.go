package main

import (
	"log"

	"github.com/AntonPashechko/ya-diplom/internal/app"
	"github.com/AntonPashechko/ya-diplom/internal/config"
	"github.com/AntonPashechko/ya-diplom/internal/logger"
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

	app, err := app.Create(cfg)
	if err != nil {
		logger.Error("cannot create app: %s", err)
		return
	}

	go app.Run()

	logger.Info("Running server: address %s", cfg.Endpoint)

	<-app.ServerDone()

	if err := app.Shutdown(); err != nil {
		logger.Error("Server shutdown failed: %s", err)
	}

	logger.Info("Server has been shutdown")
}
