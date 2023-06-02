package config

import (
	"flag"
	"os"

	"github.com/AntonPashechko/ya-diplom/internal/logger"
)

type Config struct {
	Endpoint    string
	DataBaseDNS string
}

func Create() (*Config, error) {
	cfg := &Config{}

	/*Разбираем командную строку сперва в структуру только со string полями*/
	flag.StringVar(&cfg.Endpoint, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.DataBaseDNS, "d", "", "db dns")

	flag.Parse()

	/*Но если заданы в окружении - берем оттуда*/
	if addr, exist := os.LookupEnv("RUN_ADDRESS"); exist {
		cfg.Endpoint = addr
		logger.Info("RUN_ADDRESS env: %s", addr)
	}

	if dns, exist := os.LookupEnv("DATABASE_URI"); exist {
		logger.Info("DATABASE_URI env: %s", dns)
		cfg.DataBaseDNS = dns
	}

	return cfg, nil
}
