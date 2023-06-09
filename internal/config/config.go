package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AntonPashechko/ya-diplom/internal/logger"
)

type Config struct {
	Endpoint     string
	DataBaseDNS  string
	JWT_key      []byte        //Ключ для создания/проверки jwt для авторизации
	JWT_duration time.Duration //Время действия jwt для авторизации
}

func Create() (*Config, error) {
	cfg := &Config{}

	var JWT_key, JWT_duration string
	/*Разбираем командную строку сперва в структуру только со string полями*/
	flag.StringVar(&cfg.Endpoint, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&cfg.DataBaseDNS, "d", "", "db dns")
	flag.StringVar(&JWT_key, "k", "aL6HmkWp7D", "JWT key")
	flag.StringVar(&JWT_duration, "t", "60m", "JWT duration")

	flag.Parse()

	/*Но если заданы в окружении - берем оттуда*/
	if addr, exist := os.LookupEnv("RUN_ADDRESS"); exist {
		cfg.Endpoint = addr
	}

	if dns, exist := os.LookupEnv("DATABASE_URI"); exist {
		logger.Info("DATABASE_URI env: %s", dns)
		cfg.DataBaseDNS = dns
	}

	if cfg.DataBaseDNS == `` {
		return nil, fmt.Errorf("db dns is empty")
	}

	if key, exist := os.LookupEnv("JWT_KEY"); exist {
		JWT_key = key
	}

	if duration, exist := os.LookupEnv("JWT_DURATION"); exist {
		JWT_duration = duration
	}

	cfg.JWT_key = []byte(JWT_key)
	if duration, err := time.ParseDuration(JWT_duration); err != nil {
		return nil, fmt.Errorf("JWT DURATION: %w", err)
	} else {
		cfg.JWT_duration = duration
	}

	return cfg, nil
}
